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
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api/apiutil"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/kas"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/metric"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/rpclimiter"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/tracing"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/wstunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"gitlab.com/gitlab-org/gitaly/client"
	grpccorrelation "gitlab.com/gitlab-org/labkit/correlation/grpc"
	"gitlab.com/gitlab-org/labkit/log"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
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

	defaultGitalyGlobalApiRefillRate    float64 = 10.0 // type matches protobuf type from kascfg.TokenBucketRateLimitCF
	defaultGitalyGlobalApiBucketSize    int32   = 50   // type matches protobuf type from kascfg.TokenBucketRateLimitCF
	defaultGitalyPerServerApiRate       float64 = 5.0  // type matches protobuf type from kascfg.TokenBucketRateLimitCF
	defaultGitalyPerServerApiBucketSize int32   = 10   // type matches protobuf type from kascfg.TokenBucketRateLimitCF

	correlationClientName = "gitlab-kas"
	tracingServiceName    = "gitlab-kas"
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

func kasUserAgent() string {
	return fmt.Sprintf("gitlab-kas/%s/%s", cmd.Version, cmd.Commit)
}

func (a *ConfiguredApp) startMetricsServer(st stager.Stager, gatherer prometheus.Gatherer) {
	obsCfg := a.Configuration.Observability
	if obsCfg.Prometheus.Disabled {
		return
	}
	stage := st.NextStage()
	stage.Go(func(ctx context.Context) error {
		lis, err := net.Listen(obsCfg.Listen.Network, obsCfg.Listen.Address)
		if err != nil {
			return err
		}
		defer lis.Close() // nolint: errcheck

		log.WithFields(log.Fields{
			"address":  lis.Addr().String(),
			"network":  lis.Addr().Network(),
			"url_path": obsCfg.Prometheus.UrlPath,
		}).Info("Listening for Prometheus connections")

		metricSrv := &metric.Server{
			Name:     kasUserAgent(),
			Listener: lis,
			UrlPath:  obsCfg.Prometheus.UrlPath,
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
		// Tracing
		tracer, closer, err := tracing.ConstructTracer(tracingServiceName, cfg.Observability.Tracing.ConnectionString)
		if err != nil {
			return fmt.Errorf("tracing: %v", err)
		}
		defer closer.Close() // nolint: errcheck

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

		userAgent := kasUserAgent()
		gitalyClientPool := constructGitalyPool(cfg.Gitaly, csh, tracer, userAgent)
		defer gitalyClientPool.Close() // nolint: errcheck
		gitLabClient := gitlab.NewClient(
			gitLabUrl,
			decodedAuthSecret,
			gitlab.WithCorrelationClientName(correlationClientName),
			gitlab.WithUserAgent(userAgent),
			gitlab.WithTracer(tracer),
		)
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
			UsageReportingPeriod:         cfg.Observability.UsageReportingPeriod.AsDuration(),
			Registerer:                   registerer,
		})
		if err != nil {
			return fmt.Errorf("kas.NewServer: %v", err)
		}
		defer cleanup()
		grpcServer := grpc.NewServer(
			grpc.StatsHandler(ssh),
			// TODO construct independent interceptors with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
			grpc.ChainStreamInterceptor(
				grpc_prometheus.StreamServerInterceptor, // This one should be the first one to measure all invocations
				apiutil.StreamAgentMetaInterceptor(),    // This one should be the second one to ensure agent presents a token
				grpccorrelation.StreamServerCorrelationInterceptor(),
				grpc_opentracing.StreamServerInterceptor(grpc_opentracing.WithTracer(tracer)),
			),
			grpc.ChainUnaryInterceptor(
				grpc_prometheus.UnaryServerInterceptor, // This one should be the first one to measure all invocations
				apiutil.UnaryAgentMetaInterceptor(),    // This one should be the second one to ensure agent presents a token
				grpccorrelation.UnaryServerCorrelationInterceptor(),
				grpc_opentracing.UnaryServerInterceptor(grpc_opentracing.WithTracer(tracer)),
			),
			grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             20 * time.Second,
				PermitWithoutStream: true,
			}),
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

	if cfg.Observability == nil {
		cfg.Observability = &kascfg.ObservabilityCF{}
	}
	defaultDuration(&cfg.Observability.UsageReportingPeriod, defaultUsageReportingPeriod)
	if cfg.Observability.Listen == nil {
		cfg.Observability.Listen = &kascfg.ObservabilityListenCF{}
	}
	defaultString(&cfg.Observability.Listen.Network, defaultPrometheusListenNetwork)
	defaultString(&cfg.Observability.Listen.Address, defaultPrometheusListenAddress)
	if cfg.Observability.Prometheus == nil {
		cfg.Observability.Prometheus = &kascfg.PrometheusCF{}
	}
	defaultString(&cfg.Observability.Prometheus.UrlPath, defaultPrometheusListenUrlPath)

	if cfg.Observability.Tracing == nil {
		cfg.Observability.Tracing = &kascfg.TracingCF{}
	}

	if cfg.Gitaly == nil {
		cfg.Gitaly = &kascfg.GitalyCF{}
	}
	if cfg.Gitaly.GlobalApiRateLimit == nil {
		cfg.Gitaly.GlobalApiRateLimit = &kascfg.TokenBucketRateLimitCF{}
	}
	defaultFloat64(&cfg.Gitaly.GlobalApiRateLimit.RefillRatePerSecond, defaultGitalyGlobalApiRefillRate)
	defaultInt32(&cfg.Gitaly.GlobalApiRateLimit.BucketSize, defaultGitalyGlobalApiBucketSize)
	if cfg.Gitaly.PerServerApiRateLimit == nil {
		cfg.Gitaly.PerServerApiRateLimit = &kascfg.TokenBucketRateLimitCF{}
	}
	defaultFloat64(&cfg.Gitaly.PerServerApiRateLimit.RefillRatePerSecond, defaultGitalyPerServerApiRate)
	defaultInt32(&cfg.Gitaly.PerServerApiRateLimit.BucketSize, defaultGitalyPerServerApiBucketSize)
	return nil
}

func constructGitalyPool(g *kascfg.GitalyCF, csh stats.Handler, tracer opentracing.Tracer, userAgent string) *client.Pool {
	globalGitalyRpcLimiter := rate.NewLimiter(
		rate.Limit(g.GlobalApiRateLimit.RefillRatePerSecond),
		int(g.GlobalApiRateLimit.BucketSize),
	)
	return client.NewPoolWithOptions(
		client.WithDialOptions(
			grpc.WithUserAgent(userAgent),
			grpc.WithStatsHandler(csh),
			// Don't put interceptors here as order is important. Put them below.
		),
		client.WithDialer(func(ctx context.Context, address string, dialOptions []grpc.DialOption) (*grpc.ClientConn, error) {
			perServerGitalyRpcLimiter := rate.NewLimiter(
				rate.Limit(g.PerServerApiRateLimit.RefillRatePerSecond),
				int(g.PerServerApiRateLimit.BucketSize))
			// TODO construct independent interceptors with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
			opts := []grpc.DialOption{
				grpc.WithChainStreamInterceptor(
					grpc_prometheus.StreamClientInterceptor,
					grpccorrelation.StreamClientCorrelationInterceptor(grpccorrelation.WithClientName(correlationClientName)),
					grpc_opentracing.StreamClientInterceptor(grpc_opentracing.WithTracer(tracer)),
					rpclimiter.StreamClientInterceptor(globalGitalyRpcLimiter),
					rpclimiter.StreamClientInterceptor(perServerGitalyRpcLimiter),
				),
				grpc.WithChainUnaryInterceptor(
					grpc_prometheus.UnaryClientInterceptor,
					grpccorrelation.UnaryClientCorrelationInterceptor(grpccorrelation.WithClientName(correlationClientName)),
					grpc_opentracing.UnaryClientInterceptor(grpc_opentracing.WithTracer(tracer)),
					rpclimiter.UnaryClientInterceptor(globalGitalyRpcLimiter),
					rpclimiter.UnaryClientInterceptor(perServerGitalyRpcLimiter),
				),
			}
			opts = append(opts, dialOptions...)
			return client.DialContext(ctx, address, opts)
		}),
	)
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

func defaultFloat64(s *float64, defaultValue float64) {
	if *s == 0 {
		*s = defaultValue
	}
}

func defaultInt32(s *int32, defaultValue int32) {
	if *s == 0 {
		*s = defaultValue
	}
}
