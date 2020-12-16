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
	redigo "github.com/gomodule/redigo/redis"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/kas"
	agent_configuration_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker"
	gitlab_access_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitlab_access/server"
	gitops_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/server"
	google_profiler_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/google_profiler/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	observability_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/observability/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/usage_metrics"
	usage_metrics_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/usage_metrics/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/redis"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/metric"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/tlstool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/tracing"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/wstunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"gitlab.com/gitlab-org/gitaly/client"
	grpccorrelation "gitlab.com/gitlab-org/labkit/correlation/grpc"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip" // Install the gzip compressor
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/stats"
)

const (
	authSecretLength      = 32
	defaultMaxMessageSize = 10 * 1024 * 1024

	kasName = "gitlab-kas"
)

type ConfiguredApp struct {
	Log           *zap.Logger
	Configuration *kascfg.ConfigurationFile
}

func (a *ConfiguredApp) Run(ctx context.Context) error {
	// Metrics
	// TODO use an independent registry with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
	// reg := prometheus.NewPedanticRegistry()
	registerer := prometheus.DefaultRegisterer
	gatherer := prometheus.DefaultGatherer
	ssh := metric.ServerStatsHandler()
	csh := metric.ClientStatsHandler()
	//goCollector := prometheus.NewGoCollector()
	cleanup, err := metric.Register(registerer, ssh, csh)
	if err != nil {
		return err
	}
	defer cleanup()

	cfg := a.Configuration

	// Tracing
	tracer, closer, err := tracing.ConstructTracer(kasName, cfg.Observability.Tracing.ConnectionString)
	if err != nil {
		return fmt.Errorf("tracing: %v", err)
	}
	defer closer.Close() // nolint: errcheck

	// GitLab REST client
	gitLabClient, err := a.constructGitLabClient(tracer)
	if err != nil {
		return err
	}

	// Sentry
	errTracker, err := a.constructErrorTracker()
	if err != nil {
		return fmt.Errorf("error tracker: %v", err)
	}

	// Redis
	redisPool, err := a.constructRedisPool()
	if err != nil {
		return err
	}

	// Interceptors
	interceptorsCtx, interceptorsCancel := context.WithCancel(context.Background())
	defer interceptorsCancel()

	// Agent API server
	agentServer, err := a.constructAgentServer(interceptorsCtx, tracer, redisPool, ssh)
	if err != nil {
		return err
	}

	// gRPC listener
	lis, err := net.Listen(cfg.Agent.Listen.Network.String(), cfg.Agent.Listen.Address)
	if err != nil {
		return err
	}
	defer lis.Close() // nolint: errcheck

	a.Log.Info("Listening for agentk connections",
		logz.NetNetworkFromAddr(lis.Addr()),
		logz.NetAddressFromAddr(lis.Addr()),
		logz.IsWebSocket(cfg.Agent.Listen.Websocket),
	)

	if cfg.Agent.Listen.Websocket {
		wsWrapper := wstunnel.ListenerWrapper{
			// TODO set timeouts
			ReadLimit: defaultMaxMessageSize,
		}
		lis = wsWrapper.Wrap(lis)
	}

	// Gitaly client
	gitalyClientPool := a.constructGitalyPool(csh, tracer)
	defer gitalyClientPool.Close() // nolint: errcheck

	// Usage tracker
	usageTracker := usage_metrics.NewUsageTracker()

	// Agent tracker
	agentTracker := a.constructAgentTracker(redisPool)

	// Module factories
	factories := []modserver.Factory{
		&observability_server.Factory{
			Gatherer: gatherer,
		},
		&google_profiler_server.Factory{},
		&agent_configuration_server.Factory{
			AgentTracker: agentTracker,
		},
		&gitops_server.Factory{},
		&usage_metrics_server.Factory{
			UsageTracker: usageTracker,
		},
		&gitlab_access_server.Factory{},
	}

	// Configuration for modules
	modconfig := &modserver.Config{
		Log: a.Log,
		Api: kas.NewAPI(kas.APIConfig{
			GitLabClient:           gitLabClient,
			ErrorTracker:           errTracker,
			AgentInfoCacheTtl:      cfg.Agent.InfoCacheTtl.AsDuration(),
			AgentInfoCacheErrorTtl: cfg.Agent.InfoCacheErrorTtl.AsDuration(),
		}),
		Config:       cfg,
		GitLabClient: gitLabClient,
		Registerer:   registerer,
		UsageTracker: usageTracker,
		AgentServer:  agentServer,
		Gitaly: &gitaly.Pool{
			ClientPool: gitalyClientPool,
		},
		KasName:  kasName,
		Version:  cmd.Version,
		CommitId: cmd.Commit,
	}

	// Start things up
	st := stager.New()

	// Agent tracker stage
	stage := st.NextStage()
	stage.Go(agentTracker.Run)

	// Modules stage
	stage = st.NextStage()
	for _, factory := range factories {
		module := factory.New(modconfig)
		stage.Go(func(ctx context.Context) error {
			err := module.Run(ctx)
			if err != nil {
				return fmt.Errorf("%s: %v", module.Name(), err)
			}
			return nil
		})
	}

	// gRPC server stage
	stage = st.NextStage()
	stage.Go(func(ctx context.Context) error {
		return agentServer.Serve(lis)
	})
	stage.Go(func(ctx context.Context) error {
		<-ctx.Done() // can be cancelled because Serve() failed or main ctx was canceled or some stage failed
		interceptorsCancel()
		agentServer.GracefulStop()
		return nil
	})
	return st.Run(ctx)
}

func (a *ConfiguredApp) constructAgentServer(interceptorsCtx context.Context, tracer opentracing.Tracer, redisPool redis.Pool, ssh stats.Handler) (*grpc.Server, error) {
	cfg := a.Configuration
	// TODO construct independent metrics interceptors with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
	grpcStreamServerInterceptors := []grpc.StreamServerInterceptor{
		grpc_prometheus.StreamServerInterceptor,   // This one should be the first one to measure all invocations
		grpctool.StreamServerAgentMDInterceptor(), // This one should be the second one to ensure agent presents a token
		grpccorrelation.StreamServerCorrelationInterceptor(grpccorrelation.WithoutPropagation()),
		grpc_opentracing.StreamServerInterceptor(grpc_opentracing.WithTracer(tracer)),
		grpctool.StreamServerCtxAugmentingInterceptor(grpctool.JoinContexts(interceptorsCtx)),
	}
	grpcUnaryServerInterceptors := []grpc.UnaryServerInterceptor{
		grpc_prometheus.UnaryServerInterceptor,   // This one should be the first one to measure all invocations
		grpctool.UnaryServerAgentMDInterceptor(), // This one should be the second one to ensure agent presents a token
		grpccorrelation.UnaryServerCorrelationInterceptor(grpccorrelation.WithoutPropagation()),
		grpc_opentracing.UnaryServerInterceptor(grpc_opentracing.WithTracer(tracer)),
		grpctool.UnaryServerCtxAugmentingInterceptor(grpctool.JoinContexts(interceptorsCtx)),
	}

	if redisPool != nil {
		agentConnectionLimiter := redis.NewTokenLimiter(
			a.Log,
			redisPool,
			cfg.Redis.KeyPrefix+":agent_limit",
			uint64(cfg.Agent.Listen.ConnectionsPerTokenPerMinute),
			func(ctx context.Context) string { return string(api.AgentTokenFromContext(ctx)) },
		)
		grpcStreamServerInterceptors = append(grpcStreamServerInterceptors, grpctool.StreamServerLimitingInterceptor(agentConnectionLimiter))
		grpcUnaryServerInterceptors = append(grpcUnaryServerInterceptors, grpctool.UnaryServerLimitingInterceptor(agentConnectionLimiter))
	}

	maxConnectionAge := cfg.Agent.Listen.MaxConnectionAge.AsDuration()
	serverOpts := []grpc.ServerOption{
		grpc.StatsHandler(ssh),
		grpc.ChainStreamInterceptor(grpcStreamServerInterceptors...),
		grpc.ChainUnaryInterceptor(grpcUnaryServerInterceptors...),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             20 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			// MaxConnectionAge should be below maxConnectionAge so that when kas closes a long running response
			// stream, gRPC will close the underlying connection. -20% to account for jitter (see doc for the field)
			// and ensure it's somewhat below maxConnectionAge.
			// See https://github.com/grpc/grpc-go/blob/v1.33.1/internal/transport/http2_server.go#L949-L1047 to better understand how this all works.
			MaxConnectionAge: time.Duration(0.8 * float64(maxConnectionAge)),
			// Give pending RPCs plenty of time to complete.
			// In practice it will happen in 10-30% of maxConnectionAge time (see above).
			MaxConnectionAgeGrace: maxConnectionAge,
			// trying to stay below 60 seconds (typical load-balancer timeout)
			Time: 50 * time.Second,
		}),
	}

	certFile := cfg.Agent.Listen.CertificateFile
	keyFile := cfg.Agent.Listen.KeyFile
	switch {
	case certFile != "" && keyFile != "":
		config, err := tlstool.DefaultServerTLSConfig(certFile, keyFile)
		if err != nil {
			return nil, err
		}
		serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(config)))
	case certFile == "" && keyFile == "":
	default:
		return nil, fmt.Errorf("both certificate_file (%s) and key_file (%s) must be either set or not set", certFile, keyFile)
	}

	return grpc.NewServer(serverOpts...), nil
}

func (a *ConfiguredApp) constructAgentTracker(redisPool *redigo.Pool) agent_tracker.Tracker {
	if redisPool == nil {
		return nopAgentTracker{}
	}
	cfg := a.Configuration
	return agent_tracker.NewRedisTracker(
		a.Log,
		redisPool,
		cfg.Redis.KeyPrefix+":agent_tracker",
		cfg.Agent.RedisConnInfoTtl.AsDuration(),
		cfg.Agent.RedisConnInfoRefresh.AsDuration(),
		cfg.Agent.RedisConnInfoGc.AsDuration(),
	)
}

func (a *ConfiguredApp) constructErrorTracker() (errortracking.Tracker, error) {
	s := a.Configuration.Observability.Sentry
	if s.Dsn == "" {
		return nopErrTracker{}, nil
	}
	a.Log.Debug("Initializing Sentry error tracking", logz.SentryDSN(s.Dsn), logz.SentryEnv(s.Environment))
	tracker, err := errortracking.NewTracker(
		errortracking.WithSentryDSN(s.Dsn),
		errortracking.WithVersion(kasUserAgent()),
		errortracking.WithSentryEnvironment(s.Environment),
	)
	if err != nil {
		return nil, err
	}
	return tracker, nil
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

func (a *ConfiguredApp) constructGitLabClient(tracer opentracing.Tracer) (*gitlab.Client, error) {
	cfg := a.Configuration

	gitLabUrl, err := url.Parse(cfg.Gitlab.Address)
	if err != nil {
		return nil, err
	}
	// TLS cert for talking to GitLab/Workhorse.
	clientTLSConfig, err := tlstool.DefaultClientTLSConfigWithCACert(cfg.Gitlab.CaCertificateFile)
	if err != nil {
		return nil, err
	}
	// Secret for JWT signing
	decodedAuthSecret, err := a.loadAuthSecret()
	if err != nil {
		return nil, fmt.Errorf("authentication secret: %v", err)
	}
	return gitlab.NewClient(
		gitLabUrl,
		decodedAuthSecret,
		gitlab.WithCorrelationClientName(kasName),
		gitlab.WithUserAgent(kasUserAgent()),
		gitlab.WithTracer(tracer),
		gitlab.WithLogger(a.Log),
		gitlab.WithTLSConfig(clientTLSConfig),
		gitlab.WithRateLimiter(rate.NewLimiter(
			rate.Limit(cfg.Gitlab.ApiRateLimit.RefillRatePerSecond),
			int(cfg.Gitlab.ApiRateLimit.BucketSize),
		)),
	), nil
}

func (a *ConfiguredApp) constructGitalyPool(csh stats.Handler, tracer opentracing.Tracer) *client.Pool {
	g := a.Configuration.Gitaly
	globalGitalyRpcLimiter := rate.NewLimiter(
		rate.Limit(g.GlobalApiRateLimit.RefillRatePerSecond),
		int(g.GlobalApiRateLimit.BucketSize),
	)
	return client.NewPoolWithOptions(
		client.WithDialOptions(
			grpc.WithUserAgent(kasUserAgent()),
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
					grpccorrelation.StreamClientCorrelationInterceptor(grpccorrelation.WithClientName(kasName)),
					grpc_opentracing.StreamClientInterceptor(grpc_opentracing.WithTracer(tracer)),
					grpctool.StreamClientLimitingInterceptor(globalGitalyRpcLimiter),
					grpctool.StreamClientLimitingInterceptor(perServerGitalyRpcLimiter),
				),
				grpc.WithChainUnaryInterceptor(
					grpc_prometheus.UnaryClientInterceptor,
					grpccorrelation.UnaryClientCorrelationInterceptor(grpccorrelation.WithClientName(kasName)),
					grpc_opentracing.UnaryClientInterceptor(grpc_opentracing.WithTracer(tracer)),
					grpctool.UnaryClientLimitingInterceptor(globalGitalyRpcLimiter),
					grpctool.UnaryClientLimitingInterceptor(perServerGitalyRpcLimiter),
				),
			}
			opts = append(opts, dialOptions...)
			return client.DialContext(ctx, address, opts)
		}),
	)
}

func (a *ConfiguredApp) constructRedisPool() (*redigo.Pool, error) {
	cfg := a.Configuration.Redis
	if cfg == nil {
		return nil, nil
	}
	redisCfg := &redis.Config{
		// Url is parsed below
		Password:       cfg.Password,
		MaxIdle:        int32(cfg.MaxIdle),
		MaxActive:      int32(cfg.MaxActive),
		ReadTimeout:    cfg.ReadTimeout.AsDuration(),
		WriteTimeout:   cfg.WriteTimeout.AsDuration(),
		KeepAlive:      cfg.Keepalive.AsDuration(),
		SentinelMaster: cfg.SentinelMaster,
		// Sentinels is parsed below
	}
	if cfg.Url != "" {
		var err error
		redisCfg.URL, err = url.Parse(cfg.Url)
		if err != nil {
			return nil, fmt.Errorf("kas.redis.NewPool: redis.url is not a valid URL: %v", err)
		}
	}
	for i, addr := range cfg.Sentinels {
		u, err := url.Parse(addr)
		if err != nil {
			return nil, fmt.Errorf("kas.redis.NewPool: redis.sentinels[%d] is not a valid URL: %v", i, err)
		}
		redisCfg.Sentinels = append(redisCfg.Sentinels, u)
	}
	return redis.NewPool(redisCfg), nil
}

func kasUserAgent() string {
	return fmt.Sprintf("gitlab-kas/%s/%s", cmd.Version, cmd.Commit)
}

var (
	_ errortracking.Tracker = nopErrTracker{}
	_ agent_tracker.Tracker = nopAgentTracker{}
)

// nopErrTracker is the state of the art error tracking facility.
type nopErrTracker struct {
}

func (n nopErrTracker) Capture(err error, opts ...errortracking.CaptureOption) {
}

type nopAgentTracker struct {
}

func (n nopAgentTracker) Run(ctx context.Context) error {
	return nil
}

func (n nopAgentTracker) RegisterConnection(ctx context.Context, info *agent_tracker.ConnectedAgentInfo) bool {
	return true
}

func (n nopAgentTracker) UnregisterConnection(ctx context.Context, info *agent_tracker.ConnectedAgentInfo) bool {
	return true
}
