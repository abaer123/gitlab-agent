package kasapp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ash2k/stager"
	"github.com/dgrijalva/jwt-go/v4"
	"github.com/go-redis/redis/v8"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
	agent_configuration_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_configuration/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_tracker"
	agent_tracker_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_tracker/server"
	configuration_project_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/configuration_project/server"
	gitlab_access_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitlab_access/server"
	gitops_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops/server"
	google_profiler_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/google_profiler/server"
	kubernetes_api_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/kubernetes_api/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/observability"
	observability_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/observability/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel"
	reverse_tunnel_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel/tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/usage_metrics"
	usage_metrics_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/usage_metrics/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/filez"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/metric"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/redistool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/tlstool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/tracing"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/wstunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/kascfg"
	"gitlab.com/gitlab-org/gitaly/v14/client"
	grpccorrelation "gitlab.com/gitlab-org/labkit/correlation/grpc"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"gitlab.com/gitlab-org/labkit/monitoring"
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

	envVarOwnPrivateApiUrl = "OWN_PRIVATE_API_URL"

	kasName = "gitlab-kas"
)

type ConfiguredApp struct {
	Log           *zap.Logger
	Configuration *kascfg.ConfigurationFile
}

func (a *ConfiguredApp) Run(ctx context.Context) (retErr error) {
	// This should become required later
	ownPrivateApiUrl := os.Getenv(envVarOwnPrivateApiUrl)
	// TODO make it mandatory?
	if ownPrivateApiUrl == "" {
		a.Log.Warn(envVarOwnPrivateApiUrl + " is not set, this kas instance will not be accessible to other kas instances")
	}
	// Metrics
	// TODO use an independent registry with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
	// reg := prometheus.NewPedanticRegistry()
	registerer := prometheus.DefaultRegisterer
	gatherer := prometheus.DefaultGatherer
	ssh := metric.ServerStatsHandler()
	csh := metric.ClientStatsHandler()
	//goCollector := prometheus.NewGoCollector()
	cleanup, err := metric.Register(registerer, ssh, csh, gitlabBuildInfoGauge())
	if err != nil {
		return err
	}
	defer cleanup()

	cfg := a.Configuration

	// Tracing
	tracer, closer, err := tracing.ConstructTracer(kasName, cfg.Observability.Tracing.ConnectionString)
	if err != nil {
		return fmt.Errorf("tracing: %w", err)
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
		return fmt.Errorf("error tracker: %w", err)
	}

	// Redis
	redisClient, err := a.constructRedisClient()
	if err != nil {
		return err
	}

	// Interceptors
	interceptorsCtx, interceptorsCancel := context.WithCancel(context.Background())
	defer interceptorsCancel()

	// Server for handling agentk requests
	agentServer, err := a.constructAgentServer(interceptorsCtx, tracer, redisClient, ssh)
	if err != nil {
		return fmt.Errorf("agent server: %w", err)
	}

	// Server for handling external requests e.g. from GitLab
	apiServer, err := a.constructApiServer(interceptorsCtx, tracer, ssh)
	if err != nil {
		return fmt.Errorf("API server: %w", err)
	}

	// Server for handling API requests from other kas instances
	privateApiServer, err := a.constructPrivateApiServer(interceptorsCtx, tracer, ssh)
	if err != nil {
		return fmt.Errorf("private API server: %w", err)
	}

	// Internal gRPC client->listener pipe
	internalListener := grpctool.NewDialListener()

	// Construct connection to internal gRPC server
	internalServerConn, err := a.constructInternalServerConn(ctx, tracer, internalListener.DialContext)
	if err != nil {
		return err
	}
	defer errz.SafeClose(internalServerConn, &retErr)

	// Reverse gRPC tunnel tracker
	tunnelTracker := a.constructTunnelTracker(redisClient)

	// Tunnel registry
	tunnelRegistry, err := reverse_tunnel.NewTunnelRegistry(a.Log, tunnelTracker, ownPrivateApiUrl)
	if err != nil {
		return err
	}

	// Construct internal gRPC server
	internalServer := a.constructInternalServer(interceptorsCtx, tracer)

	// Kas to agentk router
	kasToAgentRouter, err := a.constructKasToAgentRouter(tracer, tunnelTracker, tunnelRegistry, internalServer, privateApiServer)
	if err != nil {
		return err
	}

	// Agent tracker
	agentTracker := a.constructAgentTracker(redisClient)

	// Usage tracker
	usageTracker := usage_metrics.NewUsageTracker()

	// Gitaly client
	gitalyClientPool := a.constructGitalyPool(csh, tracer)
	defer errz.SafeClose(gitalyClientPool, &retErr)

	// Module factories
	factories := []modserver.Factory{
		&observability_server.Factory{
			Gatherer:       gatherer,
			LivenessProbe:  observability.NoopProbe,
			ReadinessProbe: constructReadinessProbe(redisClient),
		},
		&google_profiler_server.Factory{},
		&agent_configuration_server.Factory{
			AgentRegisterer: agentTracker,
		},
		&configuration_project_server.Factory{},
		&gitops_server.Factory{},
		&usage_metrics_server.Factory{
			UsageTracker: usageTracker,
		},
		&gitlab_access_server.Factory{},
		&agent_tracker_server.Factory{
			AgentQuerier: agentTracker,
		},
		&reverse_tunnel_server.Factory{
			TunnelHandler: tunnelRegistry,
		},
		&kubernetes_api_server.Factory{},
	}

	// Construct modules
	serverApi := newAPI(apiConfig{
		GitLabClient:           gitLabClient,
		ErrorTracker:           errTracker,
		AgentInfoCacheTtl:      cfg.Agent.InfoCacheTtl.AsDuration(),
		AgentInfoCacheErrorTtl: cfg.Agent.InfoCacheErrorTtl.AsDuration(),
	})
	poolWrapper := &gitaly.Pool{
		ClientPool: gitalyClientPool,
	}
	modules := make([]modserver.Module, 0, len(factories))
	for _, factory := range factories {
		// factory.New() must be called from the main goroutine because it may mutate a gRPC server (register an API)
		// and that can only be done before Serve() is called on the server.
		module, err := factory.New(&modserver.Config{
			Log:              a.Log.With(logz.ModuleName(factory.Name())),
			Api:              serverApi,
			Config:           cfg,
			GitLabClient:     gitLabClient,
			Registerer:       registerer,
			UsageTracker:     usageTracker,
			AgentServer:      agentServer,
			ApiServer:        apiServer,
			RegisterAgentApi: kasToAgentRouter.RegisterAgentApi,
			AgentConn:        internalServerConn,
			Gitaly:           poolWrapper,
			KasName:          kasName,
			Version:          cmd.Version,
			CommitId:         cmd.Commit,
		})
		if err != nil {
			return fmt.Errorf("%s: %w", factory.Name(), err)
		}
		modules = append(modules, module)
	}

	// Start things up. Stages are shut down in reverse order.
	return cmd.RunStages(ctx,
		// connRegistry depends on tunnelTracker so it must be stopped last
		func(stage stager.Stage) {
			stage.Go(tunnelTracker.Run)
		},
		// Start things that modules use.
		func(stage stager.Stage) {
			stage.Go(agentTracker.Run)
			stage.Go(tunnelRegistry.Run)
		},
		// Start modules.
		func(stage stager.Stage) {
			for _, module := range modules {
				module := module // closure captures the right variable
				stage.Go(func(ctx context.Context) error {
					err := module.Run(ctx)
					if err != nil {
						return fmt.Errorf("%s: %w", module.Name(), err)
					}
					return nil
				})
			}
		},
		// Start gRPC servers.
		func(stage stager.Stage) {
			a.startAgentServer(stage, agentServer, interceptorsCancel)
			a.startApiServer(stage, apiServer, interceptorsCancel)
			a.startPrivateApiServer(stage, privateApiServer, interceptorsCancel)
			a.startInternalServer(stage, internalServer, internalListener, interceptorsCancel)
		},
	)
}

func (a *ConfiguredApp) constructKasToAgentRouter(tracer opentracing.Tracer, tunnelQuerier tracker.Querier, tunnelFinder reverse_tunnel.TunnelFinder, internalServer, privateApiServer grpc.ServiceRegistrar) (kasRouter, error) {
	// TODO this should become required
	if a.Configuration.PrivateApi == nil {
		return nopKasRouter{}, nil
	}
	listenCfg := a.Configuration.PrivateApi.Listen
	jwtSecret, err := filez.LoadBase64Secret(listenCfg.AuthenticationSecretFile)
	if err != nil {
		return nil, fmt.Errorf("auth secret file: %w", err)
	}
	gatewayKasVisitor, err := grpctool.NewStreamVisitor(&GatewayKasResponse{})
	if err != nil {
		return nil, err
	}
	return &router{
		kasPool: &defaultKasPool{
			dialOpts: []grpc.DialOption{
				grpc.WithInsecure(), // TODO support TLS
				grpc.WithPerRPCCredentials(&grpctool.JwtCredentials{
					Secret:   jwtSecret,
					Audience: kasName,
					Issuer:   kasName,
					Insecure: true, // TODO support TLS
				}),
				grpc.WithChainStreamInterceptor(
					grpc_prometheus.StreamClientInterceptor,
					grpccorrelation.StreamClientCorrelationInterceptor(grpccorrelation.WithClientName(kasName)),
					grpc_opentracing.StreamClientInterceptor(grpc_opentracing.WithTracer(tracer)),
					grpctool.StreamClientValidatingInterceptor,
				),
				grpc.WithChainUnaryInterceptor(
					grpc_prometheus.UnaryClientInterceptor,
					grpccorrelation.UnaryClientCorrelationInterceptor(grpccorrelation.WithClientName(kasName)),
					grpc_opentracing.UnaryClientInterceptor(grpc_opentracing.WithTracer(tracer)),
					grpctool.UnaryClientValidatingInterceptor,
				),
			},
		},
		tunnelQuerier: tunnelQuerier,
		tunnelFinder:  tunnelFinder,
		backoff: retry.NewExponentialBackoffFactory(
			routingInitBackoff,
			routingMaxBackoff,
			routingResetDuration,
			routingBackoffFactor,
			routingJitter,
		),
		internalServer:            internalServer,
		privateApiServer:          privateApiServer,
		gatewayKasVisitor:         gatewayKasVisitor,
		routeAttemptInterval:      routeAttemptInterval,
		getTunnelsAttemptInterval: getTunnelsAttemptInterval,
	}, nil
}

func (a *ConfiguredApp) startAgentServer(stage stager.Stage, agentServer *grpc.Server, interceptorsCancel context.CancelFunc) {
	grpctool.StartServer(stage, agentServer, interceptorsCancel, func() (net.Listener, error) {
		listenCfg := a.Configuration.Agent.Listen
		lis, err := net.Listen(listenCfg.Network.String(), listenCfg.Address)
		if err != nil {
			return nil, err
		}

		a.Log.Info("Agentk API endpoint is up",
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
		a.Log.Info("API endpoint is up",
			logz.NetNetworkFromAddr(lis.Addr()),
			logz.NetAddressFromAddr(lis.Addr()),
		)
		return lis, nil
	})
}

func (a *ConfiguredApp) startPrivateApiServer(stage stager.Stage, apiServer *grpc.Server, interceptorsCancel context.CancelFunc) {
	// TODO this should become required
	if a.Configuration.PrivateApi == nil {
		return
	}
	grpctool.StartServer(stage, apiServer, interceptorsCancel, func() (net.Listener, error) {
		listenCfg := a.Configuration.PrivateApi.Listen
		lis, err := net.Listen(listenCfg.Network.String(), listenCfg.Address)
		if err != nil {
			return nil, err
		}
		a.Log.Info("Private API endpoint is up",
			logz.NetNetworkFromAddr(lis.Addr()),
			logz.NetAddressFromAddr(lis.Addr()),
		)
		return lis, nil
	})
}

func (a *ConfiguredApp) startInternalServer(stage stager.Stage, internalServer *grpc.Server, internalListener net.Listener, interceptorsCancel context.CancelFunc) {
	grpctool.StartServer(stage, internalServer, interceptorsCancel, func() (net.Listener, error) {
		return internalListener, nil
	})
}

func (a *ConfiguredApp) constructAgentServer(interceptorsCtx context.Context, tracer opentracing.Tracer, redisClient redis.UniversalClient, ssh stats.Handler) (*grpc.Server, error) {
	listenCfg := a.Configuration.Agent.Listen
	// TODO construct independent metrics interceptors with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
	grpcStreamServerInterceptors := []grpc.StreamServerInterceptor{
		grpc_prometheus.StreamServerInterceptor,   // 1. measure all invocations
		grpctool.StreamServerAgentMDInterceptor(), // 2. ensure agent presents a token
		grpccorrelation.StreamServerCorrelationInterceptor( // 3. add correlation id
			grpccorrelation.WithoutPropagation(),
			grpccorrelation.WithReversePropagation()),
		grpctool.StreamServerLoggerInterceptor(a.Log), // 4. inject logger with correlation id
		grpc_validator.StreamServerInterceptor(),
		grpc_opentracing.StreamServerInterceptor(grpc_opentracing.WithTracer(tracer)),
		grpctool.StreamServerCtxAugmentingInterceptor(grpctool.JoinContexts(interceptorsCtx)), // Last because it starts an extra goroutine
	}
	grpcUnaryServerInterceptors := []grpc.UnaryServerInterceptor{
		grpc_prometheus.UnaryServerInterceptor,   // 1. measure all invocations
		grpctool.UnaryServerAgentMDInterceptor(), // 2. ensure agent presents a token
		grpccorrelation.UnaryServerCorrelationInterceptor( // 3. add correlation id
			grpccorrelation.WithoutPropagation(),
			grpccorrelation.WithReversePropagation()),
		grpctool.UnaryServerLoggerInterceptor(a.Log), // 4. inject logger with correlation id
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

	credsOpt, err := maybeTlsCreds(listenCfg.CertificateFile, listenCfg.KeyFile)
	if err != nil {
		return nil, err
	}
	serverOpts = append(serverOpts, credsOpt...)

	return grpc.NewServer(serverOpts...), nil
}

func (a *ConfiguredApp) constructApiServer(interceptorsCtx context.Context, tracer opentracing.Tracer, ssh stats.Handler) (*grpc.Server, error) {
	// TODO this should become required
	if a.Configuration.Api == nil {
		return grpc.NewServer(), nil
	}
	listenCfg := a.Configuration.Api.Listen
	jwtSecret, err := filez.LoadBase64Secret(listenCfg.AuthenticationSecretFile)
	if err != nil {
		return nil, fmt.Errorf("auth secret file: %w", err)
	}

	jwtAuther := grpctool.NewJWTAuther(jwtSecret, jwt.WithAudience(kasName))

	// TODO construct independent metrics interceptors with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
	grpcStreamServerInterceptors := []grpc.StreamServerInterceptor{
		grpc_prometheus.StreamServerInterceptor,              // 1. measure all invocations
		grpccorrelation.StreamServerCorrelationInterceptor(), // 2. add correlation id
		grpctool.StreamServerLoggerInterceptor(a.Log),        // 3. inject logger with correlation id
		jwtAuther.StreamServerInterceptor,                    // 4. auth and maybe log
		grpc_validator.StreamServerInterceptor(),
		grpc_opentracing.StreamServerInterceptor(grpc_opentracing.WithTracer(tracer)),
		grpctool.StreamServerCtxAugmentingInterceptor(grpctool.JoinContexts(interceptorsCtx)), // Last because it starts an extra goroutine
	}
	grpcUnaryServerInterceptors := []grpc.UnaryServerInterceptor{
		grpc_prometheus.UnaryServerInterceptor,              // 1. measure all invocations
		grpccorrelation.UnaryServerCorrelationInterceptor(), // 2. add correlation id
		grpctool.UnaryServerLoggerInterceptor(a.Log),        // 3. inject logger with correlation id
		jwtAuther.UnaryServerInterceptor,                    // 4. auth and maybe log
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

	credsOpt, err := maybeTlsCreds(listenCfg.CertificateFile, listenCfg.KeyFile)
	if err != nil {
		return nil, err
	}
	serverOpts = append(serverOpts, credsOpt...)

	return grpc.NewServer(serverOpts...), nil
}

func (a *ConfiguredApp) constructPrivateApiServer(interceptorsCtx context.Context, tracer opentracing.Tracer, ssh stats.Handler) (*grpc.Server, error) {
	// TODO this should become required
	if a.Configuration.PrivateApi == nil {
		return grpc.NewServer(), nil
	}
	listenCfg := a.Configuration.PrivateApi.Listen
	jwtSecret, err := filez.LoadBase64Secret(listenCfg.AuthenticationSecretFile)
	if err != nil {
		return nil, fmt.Errorf("auth secret file: %w", err)
	}

	jwtAuther := grpctool.NewJWTAuther(jwtSecret, jwt.WithAudience(kasName), jwt.WithIssuer(kasName))

	// TODO construct independent metrics interceptors with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
	grpcStreamServerInterceptors := []grpc.StreamServerInterceptor{
		grpc_prometheus.StreamServerInterceptor,              // 1. measure all invocations
		grpccorrelation.StreamServerCorrelationInterceptor(), // 2. add correlation id
		grpctool.StreamServerLoggerInterceptor(a.Log),        // 3. inject logger with correlation id
		jwtAuther.StreamServerInterceptor,                    // 4. auth and maybe log
		grpc_validator.StreamServerInterceptor(),
		grpc_opentracing.StreamServerInterceptor(grpc_opentracing.WithTracer(tracer)),
		grpctool.StreamServerCtxAugmentingInterceptor(grpctool.JoinContexts(interceptorsCtx)), // Last because it starts an extra goroutine
	}
	grpcUnaryServerInterceptors := []grpc.UnaryServerInterceptor{
		grpc_prometheus.UnaryServerInterceptor,              // 1. measure all invocations
		grpccorrelation.UnaryServerCorrelationInterceptor(), // 2. add correlation id
		grpctool.UnaryServerLoggerInterceptor(a.Log),        // 3. inject logger with correlation id
		jwtAuther.UnaryServerInterceptor,                    // 4. auth and maybe log
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
		grpc.ForceServerCodec(grpctool.RawCodecWithProtoFallback{}),
	}
	credsOpt, err := maybeTlsCreds(listenCfg.CertificateFile, listenCfg.KeyFile)
	if err != nil {
		return nil, err
	}
	serverOpts = append(serverOpts, credsOpt...)

	return grpc.NewServer(serverOpts...), nil
}

func (a *ConfiguredApp) constructInternalServer(interceptorsCtx context.Context, tracer opentracing.Tracer) *grpc.Server {
	// TODO construct independent metrics interceptors with https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/32
	return grpc.NewServer(
		grpc.ChainStreamInterceptor(
			grpccorrelation.StreamServerCorrelationInterceptor(), // 1. add correlation id
			grpctool.StreamServerLoggerInterceptor(a.Log),        // 2. inject logger with correlation id
			grpc_opentracing.StreamServerInterceptor(grpc_opentracing.WithTracer(tracer)),
			grpctool.StreamServerCtxAugmentingInterceptor(grpctool.JoinContexts(interceptorsCtx)), // Last because it starts an extra goroutine
		),
		grpc.ChainUnaryInterceptor(
			grpccorrelation.UnaryServerCorrelationInterceptor(), // 1. add correlation id
			grpctool.UnaryServerLoggerInterceptor(a.Log),        // 2. inject logger with correlation id
			grpc_opentracing.UnaryServerInterceptor(grpc_opentracing.WithTracer(tracer)),
			grpctool.UnaryServerCtxAugmentingInterceptor(grpctool.JoinContexts(interceptorsCtx)), // Last because it starts an extra goroutine
		),
		grpc.ForceServerCodec(grpctool.RawCodec{}),
	)
}

func (a *ConfiguredApp) constructInternalServerConn(ctx context.Context, tracer opentracing.Tracer, dialContext func(ctx context.Context, addr string) (net.Conn, error)) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, "pipe",
		grpc.WithContextDialer(dialContext),
		grpc.WithInsecure(),
		grpc.WithChainStreamInterceptor(
			grpccorrelation.StreamClientCorrelationInterceptor(grpccorrelation.WithClientName(kasName)),
			grpc_opentracing.StreamClientInterceptor(grpc_opentracing.WithTracer(tracer)),
			grpctool.StreamClientValidatingInterceptor,
		),
		grpc.WithChainUnaryInterceptor(
			grpccorrelation.UnaryClientCorrelationInterceptor(grpccorrelation.WithClientName(kasName)),
			grpc_opentracing.UnaryClientInterceptor(grpc_opentracing.WithTracer(tracer)),
			grpctool.UnaryClientValidatingInterceptor,
		),
	)
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

func (a *ConfiguredApp) constructTunnelTracker(redisClient redis.UniversalClient) tracker.Tracker {
	if redisClient == nil {
		return nopTunnelTracker{}
	}
	cfg := a.Configuration
	return tracker.NewRedisTracker(
		a.Log,
		redisClient,
		cfg.Redis.KeyPrefix+":tunnel_tracker",
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

func (a *ConfiguredApp) loadGitLabClientAuthSecret() ([]byte, error) {
	decodedAuthSecret, err := filez.LoadBase64Secret(a.Configuration.Gitlab.AuthenticationSecretFile)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	if len(decodedAuthSecret) != authSecretLength {
		return nil, fmt.Errorf("decoding: expecting %d bytes, was %d", authSecretLength, len(decodedAuthSecret))
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
	decodedAuthSecret, err := a.loadGitLabClientAuthSecret()
	if err != nil {
		return nil, fmt.Errorf("authentication secret: %w", err)
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
	var err error
	var tlsConfig *tls.Config
	if cfg.Tls != nil && cfg.Tls.Enabled {
		tlsConfig, err = tlstool.DefaultClientTLSConfigWithCACertKeyPair(cfg.Tls.CaCertificateFile, cfg.Tls.CaCertificateFile, cfg.Tls.KeyFile)
		if err != nil {
			return nil, err
		}
	}
	var password string
	if cfg.PasswordFile != "" {
		passwordBytes, err := os.ReadFile(cfg.PasswordFile)
		if err != nil {
			return nil, err
		}
		password = string(passwordBytes)
	}
	switch v := cfg.RedisConfig.(type) {
	case *kascfg.RedisCF_Server:
		if tlsConfig != nil {
			tlsConfig.ServerName = strings.Split(v.Server.Address, ":")[0]
		}
		return redis.NewClient(&redis.Options{
			Addr:         v.Server.Address,
			PoolSize:     poolSize,
			DialTimeout:  dialTimeout,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
			Username:     cfg.Username,
			Password:     password,
			Network:      cfg.Network,
			TLSConfig:    tlsConfig,
		}), nil
	case *kascfg.RedisCF_Sentinel:
		var sentinelPassword string
		if v.Sentinel.SentinelPasswordFile != "" {
			sentinelPasswordBytes, err := os.ReadFile(v.Sentinel.SentinelPasswordFile)
			if err != nil {
				return nil, err
			}
			sentinelPassword = string(sentinelPasswordBytes)
		}
		return redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       v.Sentinel.MasterName,
			SentinelAddrs:    v.Sentinel.Addresses,
			DialTimeout:      dialTimeout,
			ReadTimeout:      readTimeout,
			WriteTimeout:     writeTimeout,
			PoolSize:         poolSize,
			IdleTimeout:      idleTimeout,
			Username:         cfg.Username,
			Password:         password,
			SentinelPassword: sentinelPassword,
			TLSConfig:        tlsConfig,
		}), nil
	default:
		// This should never happen
		return nil, fmt.Errorf("unexpected Redis config type: %T", cfg.RedisConfig)
	}
}

func constructReadinessProbe(redisClient redis.UniversalClient) observability.Probe {
	if redisClient == nil {
		return observability.NoopProbe
	}
	return func(ctx context.Context) error {
		status := redisClient.Ping(ctx)
		err := status.Err()
		if err != nil {
			return fmt.Errorf("redis: %w", err)
		}
		return nil
	}
}

func gitlabBuildInfoGauge() prometheus.Gauge {
	buildInfoGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: monitoring.GitlabBuildInfoGaugeMetricName,
		Help: "Current build info for this GitLab Service",
		ConstLabels: prometheus.Labels{
			"version": cmd.Version,
			"built":   cmd.BuildTime,
		},
	})
	buildInfoGauge.Set(1)
	return buildInfoGauge
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

func maybeTlsCreds(certFile, keyFile string) ([]grpc.ServerOption, error) {
	switch {
	case certFile != "" && keyFile != "":
		config, err := tlstool.DefaultServerTLSConfig(certFile, keyFile)
		if err != nil {
			return nil, err
		}
		return []grpc.ServerOption{grpc.Creds(credentials.NewTLS(config))}, nil
	case certFile == "" && keyFile == "":
		return nil, nil
	default:
		return nil, fmt.Errorf("both certificate_file (%s) and key_file (%s) must be either set or not set", certFile, keyFile)
	}
}

func kasUserAgent() string {
	return fmt.Sprintf("gitlab-kas/%s/%s", cmd.Version, cmd.Commit)
}

var (
	_ errortracking.Tracker = nopErrTracker{}
	_ agent_tracker.Tracker = nopAgentTracker{}
	_ tracker.Tracker       = nopTunnelTracker{}
	_ kasRouter             = nopKasRouter{}
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

func (n nopAgentTracker) GetConnectionsByAgentId(ctx context.Context, agentId int64, cb agent_tracker.ConnectedAgentInfoCallback) error {
	return nil
}

func (n nopAgentTracker) GetConnectionsByProjectId(ctx context.Context, projectId int64, cb agent_tracker.ConnectedAgentInfoCallback) error {
	return nil
}

type nopTunnelTracker struct {
}

func (n nopTunnelTracker) RegisterTunnel(ctx context.Context, info *tracker.TunnelInfo) bool {
	return true
}

func (n nopTunnelTracker) UnregisterTunnel(ctx context.Context, info *tracker.TunnelInfo) bool {
	return true
}

func (n nopTunnelTracker) GetTunnelsByAgentId(ctx context.Context, agentId int64, cb tracker.GetTunnelsByAgentIdCallback) error {
	return nil
}

func (n nopTunnelTracker) Run(ctx context.Context) error {
	return nil
}

type nopKasRouter struct {
}

func (r nopKasRouter) RegisterAgentApi(desc *grpc.ServiceDesc) {
}
