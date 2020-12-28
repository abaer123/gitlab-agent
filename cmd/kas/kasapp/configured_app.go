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
	"github.com/go-redis/redis/v8"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	agent_configuration_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker"
	agent_tracker_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker/server"
	gitlab_access_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitlab_access/server"
	gitops_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/server"
	google_profiler_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/google_profiler/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	observability_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/observability/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/usage_metrics"
	usage_metrics_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/usage_metrics/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/metric"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/redistool"
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

func (a *ConfiguredApp) Run(ctx context.Context) (retErr error) {
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
	defer errz.SafeClose(closer, &retErr)

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
	redisClient, err := a.constructRedisClient()
	if err != nil {
		return err
	}

	// Interceptors
	interceptorsCtx, interceptorsCancel := context.WithCancel(context.Background())
	defer interceptorsCancel()

	// Agent API server
	agentServer, err := a.constructAgentServer(interceptorsCtx, tracer, redisClient, ssh)
	if err != nil {
		return err
	}

	// API server
	apiServer, err := a.constructApiServer(interceptorsCtx, tracer, ssh)
	if err != nil {
		return err
	}

	// Gitaly client
	gitalyClientPool := a.constructGitalyPool(csh, tracer)
	defer errz.SafeClose(gitalyClientPool, &retErr)

	// Usage tracker
	usageTracker := usage_metrics.NewUsageTracker()

	// Agent tracker
	agentTracker := a.constructAgentTracker(redisClient)

	// Module factories
	factories := []modserver.Factory{
		&observability_server.Factory{
			Gatherer: gatherer,
		},
		&google_profiler_server.Factory{},
		&agent_configuration_server.Factory{
			AgentRegisterer: agentTracker,
		},
		&gitops_server.Factory{},
		&usage_metrics_server.Factory{
			UsageTracker: usageTracker,
		},
		&gitlab_access_server.Factory{},
		&agent_tracker_server.Factory{
			AgentQuerier: agentTracker,
		},
	}

	// Configuration for modules
	modconfig := &modserver.Config{
		Log: a.Log,
		Api: newAPI(apiConfig{
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
		ApiServer:    apiServer,
		Gitaly: &gitaly.Pool{
			ClientPool: gitalyClientPool,
		},
		KasName:  kasName,
		Version:  cmd.Version,
		CommitId: cmd.Commit,
	}

	// Construct modules
	modules := make([]modserver.Module, 0, len(factories))
	for _, factory := range factories {
		// factory.New() must be called from the main goroutine because it may mutate a gRPC server (register an API)
		// and that can only be done before Serve() is called on the server.
		module, err := factory.New(modconfig)
		if err != nil {
			return fmt.Errorf("%T: %v", factory, err)
		}
		modules = append(modules, module)
	}

	// Start things up. Stages are shut down in reverse order.
	return cmd.RunStages(ctx,
		// Start agent tracker.
		func(stage stager.Stage) {
			stage.Go(agentTracker.Run)
		},
		// Start modules.
		func(stage stager.Stage) {
			for _, module := range modules {
				module := module // closure captures the right variable
				stage.Go(func(ctx context.Context) error {
					err := module.Run(ctx)
					if err != nil {
						return fmt.Errorf("%s: %v", module.Name(), err)
					}
					return nil
				})
			}
		},
		// Start gRPC servers.
		func(stage stager.Stage) {
			a.startAgentServer(stage, agentServer, interceptorsCancel)
			a.startApiServer(stage, apiServer, interceptorsCancel)
		},
	)
}

func (a *ConfiguredApp) startAgentServer(stage stager.Stage, agentServer *grpc.Server, interceptorsCancel context.CancelFunc) {
	grpctool.StartServer(stage, agentServer, interceptorsCancel, func() (net.Listener, error) {
		listenCfg := a.Configuration.Agent.Listen
		lis, err := net.Listen(listenCfg.Network.String(), listenCfg.Address)
		if err != nil {
			return nil, err
		}

		a.Log.Info("Listening for agentk connections",
			logz.NetNetworkFromAddr(lis.Addr()),
			logz.NetAddressFromAddr(lis.Addr()),
			logz.IsWebSocket(listenCfg.Websocket),
		)

		if listenCfg.Websocket {
			wsWrapper := wstunnel.ListenerWrapper{
				// TODO set timeouts
				ReadLimit: defaultMaxMessageSize,
			}
			lis = wsWrapper.Wrap(lis)
		}
		return lis, nil
	})
}

func (a *ConfiguredApp) startApiServer(stage stager.Stage, apiServer *grpc.Server, interceptorsCancel context.CancelFunc) {
	// TODO this should become required
	if a.Configuration.Api == nil {
		return
	}
	grpctool.StartServer(stage, apiServer, interceptorsCancel, func() (net.Listener, error) {
		listenCfg := a.Configuration.Api.Listen
		lis, err := net.Listen(listenCfg.Network.String(), listenCfg.Address)
		if err != nil {
			return nil, err
		}
		a.Log.Info("Listening for API connections",
			logz.NetNetworkFromAddr(lis.Addr()),
			logz.NetAddressFromAddr(lis.Addr()),
		)
		return lis, nil
	})
}

func (a *ConfiguredApp) constructAgentServer(interceptorsCtx context.Context, tracer opentracing.Tracer, redisClient redis.UniversalClient, ssh stats.Handler) (*grpc.Server, error) {
	listenCfg := a.Configuration.Agent.Listen
	// TODO construct independent metrics interceptors with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
	grpcStreamServerInterceptors := []grpc.StreamServerInterceptor{
		grpc_prometheus.StreamServerInterceptor,                                                  // 1. measure all invocations
		grpctool.StreamServerAgentMDInterceptor(),                                                // 2. ensure agent presents a token
		grpccorrelation.StreamServerCorrelationInterceptor(grpccorrelation.WithoutPropagation()), // 3. add correlation id
		grpctool.StreamServerCtxAugmentingInterceptor(grpctool.LoggerInjector(a.Log)),            // 4. inject logger with correlation id
		grpc_validator.StreamServerInterceptor(),
		grpc_opentracing.StreamServerInterceptor(grpc_opentracing.WithTracer(tracer)),
		grpctool.StreamServerCtxAugmentingInterceptor(grpctool.JoinContexts(interceptorsCtx)), // Last because it starts an extra goroutine
	}
	grpcUnaryServerInterceptors := []grpc.UnaryServerInterceptor{
		grpc_prometheus.UnaryServerInterceptor,                                                  // 1. measure all invocations
		grpctool.UnaryServerAgentMDInterceptor(),                                                // 2. ensure agent presents a token
		grpccorrelation.UnaryServerCorrelationInterceptor(grpccorrelation.WithoutPropagation()), // 3. add correlation id
		grpctool.UnaryServerCtxAugmentingInterceptor(grpctool.LoggerInjector(a.Log)),            // 4. inject logger with correlation id
		grpc_validator.UnaryServerInterceptor(),
		grpc_opentracing.UnaryServerInterceptor(grpc_opentracing.WithTracer(tracer)),
		grpctool.UnaryServerCtxAugmentingInterceptor(grpctool.JoinContexts(interceptorsCtx)), // Last because it starts an extra goroutine
	}

	if redisClient != nil {
		agentConnectionLimiter := redistool.NewTokenLimiter(
			a.Log,
			redisClient,
			a.Configuration.Redis.KeyPrefix+":agent_limit",
			uint64(listenCfg.ConnectionsPerTokenPerMinute),
			func(ctx context.Context) string { return string(api.AgentTokenFromContext(ctx)) },
		)
		grpcStreamServerInterceptors = append(grpcStreamServerInterceptors, grpctool.StreamServerLimitingInterceptor(agentConnectionLimiter))
		grpcUnaryServerInterceptors = append(grpcUnaryServerInterceptors, grpctool.UnaryServerLimitingInterceptor(agentConnectionLimiter))
	}

	serverOpts := []grpc.ServerOption{
		grpc.StatsHandler(ssh),
		grpc.ChainStreamInterceptor(grpcStreamServerInterceptors...),
		grpc.ChainUnaryInterceptor(grpcUnaryServerInterceptors...),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             20 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepaliveParams(listenCfg.MaxConnectionAge.AsDuration())),
	}

	certFile := listenCfg.CertificateFile
	keyFile := listenCfg.KeyFile
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

func (a *ConfiguredApp) constructApiServer(interceptorsCtx context.Context, tracer opentracing.Tracer, ssh stats.Handler) (*grpc.Server, error) {
	// TODO this should become required
	if a.Configuration.Api == nil {
		return grpc.NewServer(), nil
	}
	listenCfg := a.Configuration.Api.Listen
	jwtSecret, err := ioutil.ReadFile(listenCfg.AuthenticationSecretFile)
	if err != nil {
		return nil, fmt.Errorf("auth secret file: %v", err)
	}

	jwtAuther := grpctool.JWTAuther{
		Secret:   jwtSecret,
		Audience: kasName,
	}

	// TODO construct independent metrics interceptors with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
	grpcStreamServerInterceptors := []grpc.StreamServerInterceptor{
		grpc_prometheus.StreamServerInterceptor,                                       // 1. measure all invocations
		grpccorrelation.StreamServerCorrelationInterceptor(),                          // 2. add correlation id
		grpctool.StreamServerCtxAugmentingInterceptor(grpctool.LoggerInjector(a.Log)), // 3. inject logger with correlation id
		jwtAuther.StreamServerInterceptor,                                             // 4. auth and maybe log
		grpc_validator.StreamServerInterceptor(),
		grpc_opentracing.StreamServerInterceptor(grpc_opentracing.WithTracer(tracer)),
		grpctool.StreamServerCtxAugmentingInterceptor(grpctool.JoinContexts(interceptorsCtx)), // Last because it starts an extra goroutine
	}
	grpcUnaryServerInterceptors := []grpc.UnaryServerInterceptor{
		grpc_prometheus.UnaryServerInterceptor,                                       // 1. measure all invocations
		grpccorrelation.UnaryServerCorrelationInterceptor(),                          // 2. add correlation id
		grpctool.UnaryServerCtxAugmentingInterceptor(grpctool.LoggerInjector(a.Log)), // 3. inject logger with correlation id
		jwtAuther.UnaryServerInterceptor,                                             // 4. auth and maybe log
		grpc_validator.UnaryServerInterceptor(),
		grpc_opentracing.UnaryServerInterceptor(grpc_opentracing.WithTracer(tracer)),
		grpctool.UnaryServerCtxAugmentingInterceptor(grpctool.JoinContexts(interceptorsCtx)), // Last because it starts an extra goroutine
	}
	serverOpts := []grpc.ServerOption{
		grpc.StatsHandler(ssh),
		grpc.ChainStreamInterceptor(grpcStreamServerInterceptors...),
		grpc.ChainUnaryInterceptor(grpcUnaryServerInterceptors...),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             20 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepaliveParams(listenCfg.MaxConnectionAge.AsDuration())),
	}

	certFile := listenCfg.CertificateFile
	keyFile := listenCfg.KeyFile
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

func (a *ConfiguredApp) constructAgentTracker(redisClient redis.UniversalClient) agent_tracker.Tracker {
	if redisClient == nil {
		return nopAgentTracker{}
	}
	cfg := a.Configuration
	return agent_tracker.NewRedisTracker(
		a.Log,
		redisClient,
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

func (a *ConfiguredApp) constructRedisClient() (redis.UniversalClient, error) {
	cfg := a.Configuration.Redis
	if cfg == nil {
		return nil, nil
	}
	poolSize := int(cfg.PoolSize)
	dialTimeout := cfg.DialTimeout.AsDuration()
	readTimeout := cfg.ReadTimeout.AsDuration()
	writeTimeout := cfg.WriteTimeout.AsDuration()
	idleTimeout := cfg.IdleTimeout.AsDuration()
	switch v := cfg.RedisConfig.(type) {
	case *kascfg.RedisCF_Server:
		opts, err := redis.ParseURL(v.Server.Url)
		if err != nil {
			return nil, err
		}
		if opts.TLSConfig != nil {
			tlsCfg := tlstool.DefaultClientTLSConfig()
			tlsCfg.ServerName = opts.TLSConfig.ServerName
			opts.TLSConfig = tlsCfg
		}
		opts.PoolSize = poolSize
		opts.DialTimeout = dialTimeout
		opts.ReadTimeout = readTimeout
		opts.WriteTimeout = writeTimeout
		opts.IdleTimeout = idleTimeout
		return redis.NewClient(opts), nil
	case *kascfg.RedisCF_Sentinel:
		return redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    v.Sentinel.MasterName,
			SentinelAddrs: v.Sentinel.Addresses,
			DialTimeout:   dialTimeout,
			ReadTimeout:   readTimeout,
			WriteTimeout:  writeTimeout,
			PoolSize:      poolSize,
			IdleTimeout:   idleTimeout,
		}), nil
	case *kascfg.RedisCF_Cluster:
		return redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        v.Cluster.Addresses,
			DialTimeout:  dialTimeout,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			PoolSize:     poolSize,
			IdleTimeout:  idleTimeout,
		}), nil
	default:
		// This should never happen
		return nil, fmt.Errorf("unexpected Redis config type: %T", cfg.RedisConfig)
	}
}

func keepaliveParams(maxConnectionAge time.Duration) keepalive.ServerParameters {
	return keepalive.ServerParameters{
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
	}
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

func (n nopAgentTracker) GetConnectionsByAgentId(ctx context.Context, agentId int64) ([]*agent_tracker.ConnectedAgentInfo, error) {
	return nil, nil
}

func (n nopAgentTracker) GetConnectionsByProjectId(ctx context.Context, projectId int64) ([]*agent_tracker.ConnectedAgentInfo, error) {
	return nil, nil
}
