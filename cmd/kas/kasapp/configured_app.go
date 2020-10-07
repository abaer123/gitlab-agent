package kasapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"time"

	"github.com/ash2k/stager"
	"github.com/golang/protobuf/ptypes/duration"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/kas"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/metric"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/wstunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/labkit/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
	"google.golang.org/protobuf/types/known/durationpb"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	authSecretLength      = 32
	defaultMaxMessageSize = 10 * 1024 * 1024

	defaultListenNetwork = "tcp"
	defaultListenAddress = "127.0.0.1:8150"
	defaultGitLabAddress = "http://localhost:8080"

	defaultAgentConfigurationPollPeriod = 20 * time.Second

	defaultAgentInfoCacheTTL      = 5 * time.Minute
	defaultAgentInfoCacheErrorTTL = 1 * time.Minute

	defaultGitOpsPollPeriod               = 20 * time.Second
	defaultGitOpsProjectInfoCacheTTL      = 5 * time.Minute
	defaultGitOpsProjectInfoCacheErrorTTL = 1 * time.Minute

	defaultUsageReportingPeriod    = 1 * time.Minute
	defaultPrometheusListenNetwork = "tcp"
	defaultPrometheusListenAddress = "127.0.0.1:8151"
	defaultPrometheusListenUrlPath = "/metrics"
)

type ConfiguredApp struct {
	Configuration *kascfg.ConfigurationFile
}

func (a *ConfiguredApp) Run(ctx context.Context) error {
	// Metrics
	// TODO use an independent registry with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
	// reg := prometheus.NewPedanticRegistry()
	reg := prometheus.DefaultRegisterer
	gatherer := prometheus.DefaultGatherer
	ssh := metric.ServerStatsHandler()
	csh := metric.ClientStatsHandler()
	//goCollector := prometheus.NewGoCollector()
	cleanup, err := metric.Register(reg, ssh, csh)
	if err != nil {
		return err
	}
	defer cleanup()

	// Start things up
	st := stager.New()
	a.startMetricsServer(st, gatherer)
	a.startGrpcServer(st, reg, ssh, csh)
	return st.Run(ctx)
}

func kasVersionString() string {
	return fmt.Sprintf("gitlab-kas/%s/%s", cmd.Version, cmd.Commit)
}

func (a *ConfiguredApp) startMetricsServer(st stager.Stager, gatherer prometheus.Gatherer) {
	promCfg := a.Configuration.Metrics.PrometheusListen
	if promCfg.Disabled {
		return
	}
	stage := st.NextStage()
	stage.Go(func(ctx context.Context) error {
		lis, err := net.Listen(promCfg.Network, promCfg.Address)
		if err != nil {
			return err
		}
		defer lis.Close() // nolint: errcheck

		log.WithFields(log.Fields{
			"address":  lis.Addr().String(),
			"network":  lis.Addr().Network(),
			"url_path": promCfg.UrlPath,
		}).Info("Listening for Prometheus connections")

		metricSrv := &metric.Server{
			Name:     kasVersionString(),
			Listener: lis,
			UrlPath:  promCfg.UrlPath,
			Gatherer: gatherer,
		}
		return metricSrv.Run(ctx)
	})
}

func (a *ConfiguredApp) startGrpcServer(st stager.Stager, registerer prometheus.Registerer, ssh, csh stats.Handler) {
	stage := st.NextStage()
	stage.Go(func(ctx context.Context) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		cfg := a.Configuration

		gitLabUrl, err := url.Parse(cfg.Gitlab.Address)
		if err != nil {
			return err
		}
		// Secret for JWT signing
		decodedAuthSecret, err := a.loadAuthSecret()
		if err != nil {
			return fmt.Errorf("authentication secret: %v", err)
		}
		// gRPC listener
		lis, err := net.Listen(cfg.Listen.Network, cfg.Listen.Address)
		if err != nil {
			return err
		}
		defer lis.Close() // nolint: errcheck

		log.WithFields(log.Fields{
			"address":   lis.Addr().String(),
			"network":   lis.Addr().Network(),
			"websocket": cfg.GetListen().GetWebsocket(),
		}).Info("Listening for agentk connections")

		if cfg.GetListen().GetWebsocket() {
			wsWrapper := wstunnel.ListenerWrapper{
				// TODO set timeouts
				ReadLimit: defaultMaxMessageSize,
			}
			lis = wsWrapper.Wrap(lis)
		}

		ver := kasVersionString()
		gitalyClientPool := client.NewPool(
			grpc.WithUserAgent(ver),
			grpc.WithStatsHandler(csh),
			// TODO construct independent interceptors with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
			grpc.WithChainStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
			grpc.WithChainUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		)
		defer gitalyClientPool.Close() // nolint: errcheck
		gitLabClient := gitlab.NewClient(gitLabUrl, decodedAuthSecret, ver)
		gitLabCachingClient := gitlab.NewCachingClient(gitLabClient, gitlab.CacheOptions{
			CacheTTL:      cfg.Agent.InfoCacheTtl.AsDuration(),
			CacheErrorTTL: cfg.Agent.InfoCacheErrorTtl.AsDuration(),
		}, gitlab.CacheOptions{
			CacheTTL:      cfg.Agent.Gitops.ProjectInfoCacheTtl.AsDuration(),
			CacheErrorTTL: cfg.Agent.Gitops.ProjectInfoCacheErrorTtl.AsDuration(),
		})
		srv, cleanup, err := kas.NewServer(kas.ServerConfig{
			Context: ctx,
			GitalyPool: &gitaly.Pool{
				ClientPool: gitalyClientPool,
			},
			GitLabClient:                 gitLabCachingClient,
			AgentConfigurationPollPeriod: cfg.Agent.Configuration.PollPeriod.AsDuration(),
			GitopsPollPeriod:             cfg.Agent.Gitops.PollPeriod.AsDuration(),
			UsageReportingPeriod:         cfg.Metrics.UsageReportingPeriod.AsDuration(),
			Registerer:                   registerer,
		})
		if err != nil {
			return fmt.Errorf("kas.NewServer: %v", err)
		}
		defer cleanup()
		grpcServer := grpc.NewServer(
			grpc.StatsHandler(ssh),
			// TODO construct independent interceptors with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
			grpc.ChainStreamInterceptor(grpc_prometheus.StreamServerInterceptor),
			grpc.ChainUnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		)
		agentrpc.RegisterKasServer(grpcServer, srv)

		var wg wait.Group
		defer wg.Wait() // wait for grpcServer to shutdown
		defer cancel()  // cancel ctx
		wg.Start(func() {
			<-ctx.Done() // can be cancelled because Serve() failed or because main ctx was cancelled
			grpcServer.GracefulStop()
		})
		return grpcServer.Serve(lis)
	})
}

func (a *ConfiguredApp) loadAuthSecret() ([]byte, error) {
	encodedAuthSecret, err := ioutil.ReadFile(a.Configuration.Gitlab.AuthenticationSecretFile) // nolint: gosec
	if err != nil {
		return nil, fmt.Errorf("read file: %v", err)
	}
	decodedAuthSecret := make([]byte, authSecretLength)

	n, err := base64.StdEncoding.Decode(decodedAuthSecret, encodedAuthSecret)
	if err != nil {
		return nil, fmt.Errorf("decoding: %v", err)
	}
	if n != authSecretLength {
		return nil, fmt.Errorf("decoding: expecting %d bytes, was %d", authSecretLength, n)
	}
	return decodedAuthSecret, nil
}

func ApplyDefaultsToKasConfigurationFile(cfg *kascfg.ConfigurationFile) error {
	if cfg.Listen == nil {
		cfg.Listen = &kascfg.ListenCF{}
	}
	defaultString(&cfg.Listen.Network, defaultListenNetwork)
	defaultString(&cfg.Listen.Address, defaultListenAddress)
	if cfg.Gitlab == nil {
		cfg.Gitlab = &kascfg.GitLabCF{}
	}
	defaultString(&cfg.Gitlab.Address, defaultGitLabAddress)
	if cfg.Agent == nil {
		cfg.Agent = &kascfg.AgentCF{}
	}
	if cfg.Agent.Configuration == nil {
		cfg.Agent.Configuration = &kascfg.AgentConfigurationCF{}
	}
	defaultDuration(&cfg.Agent.Configuration.PollPeriod, defaultAgentConfigurationPollPeriod)
	if cfg.Agent.Gitops == nil {
		cfg.Agent.Gitops = &kascfg.GitopsCF{}
	}
	defaultDuration(&cfg.Agent.Gitops.PollPeriod, defaultGitOpsPollPeriod)
	defaultDuration(&cfg.Agent.Gitops.ProjectInfoCacheTtl, defaultGitOpsProjectInfoCacheTTL)
	defaultDuration(&cfg.Agent.Gitops.ProjectInfoCacheErrorTtl, defaultGitOpsProjectInfoCacheErrorTTL)

	defaultDuration(&cfg.Agent.InfoCacheTtl, defaultAgentInfoCacheTTL)
	defaultDuration(&cfg.Agent.InfoCacheErrorTtl, defaultAgentInfoCacheErrorTTL)

	if cfg.Metrics == nil {
		cfg.Metrics = &kascfg.MetricsCF{}
	}
	defaultDuration(&cfg.Metrics.UsageReportingPeriod, defaultUsageReportingPeriod)
	if cfg.Metrics.PrometheusListen == nil {
		cfg.Metrics.PrometheusListen = &kascfg.PrometheusListenCF{}
	}

	defaultString(&cfg.Metrics.PrometheusListen.Network, defaultPrometheusListenNetwork)
	defaultString(&cfg.Metrics.PrometheusListen.Address, defaultPrometheusListenAddress)
	defaultString(&cfg.Metrics.PrometheusListen.UrlPath, defaultPrometheusListenUrlPath)
	return nil
}

func defaultDuration(d **duration.Duration, defaultValue time.Duration) {
	if *d == nil {
		*d = durationpb.New(defaultValue)
	}
}

func defaultString(s *string, defaultValue string) {
	if *s == "" {
		*s = defaultValue
	}
}
