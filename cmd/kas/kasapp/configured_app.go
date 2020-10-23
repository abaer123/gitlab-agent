package kasapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"time"

	"cloud.google.com/go/profiler"
	"github.com/ash2k/stager"
	"github.com/getsentry/sentry-go"
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
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/redis"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/grpctools"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/metric"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/sentryapi"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/tracing"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/wstunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"gitlab.com/gitlab-org/gitaly/client"
	grpccorrelation "gitlab.com/gitlab-org/labkit/correlation/grpc"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/stats"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	authSecretLength      = 32
	defaultMaxMessageSize = 10 * 1024 * 1024

	correlationClientName     = "gitlab-kas"
	tracingServiceName        = "gitlab-kas"
	googleProfilerServiceName = "gitlab-kas"
)

type ConfiguredApp struct {
	Configuration *kascfg.ConfigurationFile
	Log           *zap.Logger
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
	a.startGoogleProfiler(st)
	a.startMetricsServer(st, gatherer, reg)
	a.startGrpcServer(st, reg, ssh, csh)
	return st.Run(ctx)
}

func kasUserAgent() string {
	return fmt.Sprintf("gitlab-kas/%s/%s", cmd.Version, cmd.Commit)
}

func (a *ConfiguredApp) constructSentryHub() (sentryapi.Hub, error) {
	s := a.Configuration.Observability.Sentry
	if s.Dsn == "" {
		return sentryapi.NewHub(nil, sentry.NewScope()), nil
	}

	version := kasUserAgent()

	a.Log.Debug("Initializing Sentry error tracking", logz.SentryDSN(s.Dsn), logz.SentryEnv(s.Environment))
	c, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:         s.Dsn,
		Release:     version,
		Environment: s.Environment,
	})
	if err != nil {
		return nil, err
	}
	return sentryapi.NewHub(c, sentry.NewScope()), nil
}

func (a *ConfiguredApp) startGoogleProfiler(st stager.Stager) {
	cfg := a.Configuration.Observability.GoogleProfiler
	if !cfg.Enabled {
		return
	}
	stage := st.NextStage()
	stage.Go(func(ctx context.Context) error {
		config := profiler.Config{
			Service:        googleProfilerServiceName,
			ServiceVersion: cmd.Version,
			MutexProfiling: true, // like in LabKit
			ProjectID:      cfg.ProjectId,
		}
		var opts []option.ClientOption
		if cfg.CredentialsFile != "" {
			opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
		}
		err := profiler.Start(config, opts...)
		if err != nil {
			return fmt.Errorf("google profiler: %v", err)
		}
		return nil
	})
}

func (a *ConfiguredApp) startMetricsServer(st stager.Stager, gatherer prometheus.Gatherer, registerer prometheus.Registerer) {
	cfg := a.Configuration.Observability
	if cfg.Prometheus.Disabled && cfg.Pprof.Disabled {
		return
	}
	if cfg.Prometheus.Disabled {
		// Do not expose Prometheus if it is disabled
		gatherer = nil
		registerer = nil
	}
	stage := st.NextStage()
	stage.Go(func(ctx context.Context) error {
		lis, err := net.Listen(cfg.Listen.Network, cfg.Listen.Address)
		if err != nil {
			return err
		}
		defer lis.Close() // nolint: errcheck

		a.Log.Info(fmt.Sprintf("Observability endpoint is up. Prometheus enabled: %t, pprof enabled: %t", !cfg.Prometheus.Disabled, !cfg.Pprof.Disabled),
			logz.NetNetworkFromAddr(lis.Addr()),
			logz.NetAddressFromAddr(lis.Addr()),
			logz.UrlPath(cfg.Prometheus.UrlPath),
		)

		metricSrv := &metric.Server{
			Name:          kasUserAgent(),
			Listener:      lis,
			UrlPath:       cfg.Prometheus.UrlPath,
			Gatherer:      gatherer,
			Registerer:    registerer,
			PprofDisabled: cfg.Pprof.Disabled,
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

		// Sentry
		hub, err := a.constructSentryHub()
		if err != nil {
			return fmt.Errorf("Sentry: %v", err)
		}

		// gRPC listener
		lis, err := net.Listen(cfg.Listen.Network, cfg.Listen.Address)
		if err != nil {
			return err
		}
		defer lis.Close() // nolint: errcheck

		a.Log.Info("Listening for agentk connections",
			logz.NetNetworkFromAddr(lis.Addr()),
			logz.NetAddressFromAddr(lis.Addr()),
			logz.IsWebSocket(cfg.Listen.Websocket),
		)

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
			gitlab.WithLogger(a.Log),
		)
		gitLabCachingClient := gitlab.NewCachingClient(gitLabClient, gitlab.CacheOptions{
			CacheTTL:      cfg.Agent.InfoCacheTtl.AsDuration(),
			CacheErrorTTL: cfg.Agent.InfoCacheErrorTtl.AsDuration(),
		}, gitlab.CacheOptions{
			CacheTTL:      cfg.Agent.Gitops.ProjectInfoCacheTtl.AsDuration(),
			CacheErrorTTL: cfg.Agent.Gitops.ProjectInfoCacheErrorTtl.AsDuration(),
		})

		srv, cleanup, err := kas.NewServer(kas.ServerConfig{
			Log: a.Log,
			GitalyPool: &gitaly.Pool{
				ClientPool: gitalyClientPool,
			},
			GitLabClient:                 gitLabCachingClient,
			AgentConfigurationPollPeriod: cfg.Agent.Configuration.PollPeriod.AsDuration(),
			GitopsPollPeriod:             cfg.Agent.Gitops.PollPeriod.AsDuration(),
			UsageReportingPeriod:         cfg.Observability.UsageReportingPeriod.AsDuration(),
			Registerer:                   registerer,
			Sentry:                       hub,
		})
		if err != nil {
			return fmt.Errorf("kas.NewServer: %v", err)
		}
		defer cleanup()

		// TODO construct independent metrics interceptors with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
		grpcStreamServerInterceptors := []grpc.StreamServerInterceptor{
			grpc_prometheus.StreamServerInterceptor, // This one should be the first one to measure all invocations
			apiutil.StreamAgentMetaInterceptor(),    // This one should be the second one to ensure agent presents a token
			grpccorrelation.StreamServerCorrelationInterceptor(),
			grpc_opentracing.StreamServerInterceptor(grpc_opentracing.WithTracer(tracer)),
			grpctools.StreamServerCtxAugmentingInterceptor(grpctools.JoinContexts(ctx)),
		}
		grpcUnaryServerInterceptors := []grpc.UnaryServerInterceptor{
			grpc_prometheus.UnaryServerInterceptor, // This one should be the first one to measure all invocations
			apiutil.UnaryAgentMetaInterceptor(),    // This one should be the second one to ensure agent presents a token
			grpccorrelation.UnaryServerCorrelationInterceptor(),
			grpc_opentracing.UnaryServerInterceptor(grpc_opentracing.WithTracer(tracer)),
			grpctools.UnaryServerCtxAugmentingInterceptor(grpctools.JoinContexts(ctx)),
		}
		if cfg.Redis != nil {
			redisCfg := &redis.Config{
				// Url is parsed below
				Password:       cfg.Redis.Password,
				MaxIdle:        cfg.Redis.MaxIdle,
				MaxActive:      cfg.Redis.MaxActive,
				ReadTimeout:    cfg.Redis.ReadTimeout.AsDuration(),
				WriteTimeout:   cfg.Redis.WriteTimeout.AsDuration(),
				KeepAlive:      cfg.Redis.Keepalive.AsDuration(),
				SentinelMaster: cfg.Redis.SentinelMaster,
				// Sentinels is parsed below
			}
			if cfg.Redis.Url != "" {
				redisCfg.URL, err = url.Parse(cfg.Redis.Url)
				if err != nil {
					return fmt.Errorf("kas.redis.NewPool: redis.url is not a valid URL: %v", err)
				}
			}
			for i, addr := range cfg.Redis.Sentinels {
				u, err := url.Parse(addr)
				if err != nil {
					return fmt.Errorf("kas.redis.NewPool: redis.sentinels[%d] is not a valid URL: %v", i, err)
				}
				redisCfg.Sentinels = append(redisCfg.Sentinels, u)
			}
			redisPool := redis.NewPool(redisCfg)
			agentConnectionLimiter := redis.NewTokenLimiter(
				a.Log,
				redisPool,
				cfg.Agent.Limits.RedisKeyPrefix,
				uint64(cfg.Agent.Limits.ConnectionsPerTokenPerMinute),
				func(ctx context.Context) string { return string(apiutil.AgentTokenFromContext(ctx)) },
			)
			grpcStreamServerInterceptors = append(grpcStreamServerInterceptors, grpctools.StreamServerLimitingInterceptor(agentConnectionLimiter))
			grpcUnaryServerInterceptors = append(grpcUnaryServerInterceptors, grpctools.UnaryServerLimitingInterceptor(agentConnectionLimiter))
		}

		grpcServer := grpc.NewServer(
			grpc.StatsHandler(ssh),
			grpc.ChainStreamInterceptor(grpcStreamServerInterceptors...),
			grpc.ChainUnaryInterceptor(grpcUnaryServerInterceptors...),
			grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             20 * time.Second,
				PermitWithoutStream: true,
			}),
		)
		agentrpc.RegisterKasServer(grpcServer, srv)

		var wg wait.Group
		defer wg.Wait() // wait for grpcServer to shutdown
		defer cancel()  // cancel ctx
		wg.StartWithContext(ctx, srv.Run)
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
					grpctools.StreamClientLimitingInterceptor(globalGitalyRpcLimiter),
					grpctools.StreamClientLimitingInterceptor(perServerGitalyRpcLimiter),
				),
				grpc.WithChainUnaryInterceptor(
					grpc_prometheus.UnaryClientInterceptor,
					grpccorrelation.UnaryClientCorrelationInterceptor(grpccorrelation.WithClientName(correlationClientName)),
					grpc_opentracing.UnaryClientInterceptor(grpc_opentracing.WithTracer(tracer)),
					grpctools.UnaryClientLimitingInterceptor(globalGitalyRpcLimiter),
					grpctools.UnaryClientLimitingInterceptor(perServerGitalyRpcLimiter),
				),
			}
			opts = append(opts, dialOptions...)
			return client.DialContext(ctx, address, opts)
		}),
	)
}
