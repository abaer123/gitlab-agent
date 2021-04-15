package agentkapp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/ash2k/stager"
	"github.com/go-logr/zapr"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc"
	cilium_agent "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/cilium_alert/agent"
	gitlab_access_rpc "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitlab_access/rpc"
	gitops_agent "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/agent"
	kubernetes_api_agent "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/kubernetes_api/agent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modshared"
	observability_agent "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/observability/agent"
	reverse_tunnel_agent "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/agent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/tlstool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/wstunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	grpccorrelation "gitlab.com/gitlab-org/labkit/correlation/grpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // Install the GCP auth plugin
	"k8s.io/klog/v2"
	"nhooyr.io/websocket"
)

const (
	defaultRefreshConfigurationRetryPeriod    = 10 * time.Second
	defaultGetObjectsToSynchronizeRetryPeriod = 10 * time.Second

	defaultLoggingLevel agentcfg.LoggingLevelEnum = 0 // whatever is 0 is the default value

	defaultMaxMessageSize = 10 * 1024 * 1024
	agentName             = "gitlab-agent"

	envVarPodNamespace = "POD_NAMESPACE"
	envVarPodName      = "POD_NAME"
)

type App struct {
	Log       *zap.Logger
	LogLevel  zap.AtomicLevel
	AgentMeta *modshared.AgentMeta
	// KasAddress specifies the address of kas.
	KasAddress      string
	CACertFile      string
	TokenFile       string
	K8sClientGetter resource.RESTClientGetter
}

func (a *App) Run(ctx context.Context) (retErr error) {
	defer errz.SafeCall(a.Log.Sync, &retErr)
	// Kubernetes uses klog so here we pipe all logs from it to our logger via an adapter.
	klog.SetLogger(zapr.NewLogger(a.Log))

	// Construct gRPC connection to gitlab-kas
	kasConn, err := a.constructKasConnection(ctx)
	if err != nil {
		return err
	}
	defer errz.SafeClose(kasConn, &retErr)

	// Interceptors
	interceptorsCtx, interceptorsCancel := context.WithCancel(context.Background())
	defer interceptorsCancel()

	// Internal gRPC client->listener pipe
	internalListener := grpctool.NewDialListener()

	// Construct internal gRPC server
	internalServer := a.constructInternalServer(interceptorsCtx)

	// Construct connection to internal gRPC server
	internalServerConn, err := a.constructInternalServerConn(ctx, internalListener.DialContext)
	if err != nil {
		return err
	}

	// Construct agent modules
	modules, err := a.constructModules(internalServer, kasConn, internalServerConn)
	if err != nil {
		return err
	}
	runner := newModuleRunner(a.Log, modules, &rpc.ConfigurationWatcher{
		Log:         a.Log,
		AgentMeta:   a.AgentMeta,
		Client:      rpc.NewAgentConfigurationClient(kasConn),
		RetryPeriod: defaultRefreshConfigurationRetryPeriod,
	})

	// Start things up. Stages are shut down in reverse order.
	return cmd.RunStages(ctx,
		// Start modules.
		func(stage stager.Stage) {
			stage.Go(runner.RunModules)
		},
		func(stage stager.Stage) {
			// Start internal gRPC server.
			a.startInternalServer(stage, internalServer, internalListener, interceptorsCancel)
			// Start configuration refresh.
			stage.Go(runner.RunConfigurationRefresh)
		},
	)
}

func (a *App) constructModules(internalServer *grpc.Server, kasConn, internalServerConn grpc.ClientConnInterface) ([]modagent.Module, error) {
	sv, err := grpctool.NewStreamVisitor(&grpctool.HttpResponse{})
	if err != nil {
		return nil, err
	}
	factories := []modagent.Factory{
		//  Should be the first to configure logging ASAP
		&observability_agent.Factory{
			LogLevel: a.LogLevel,
		},
		&gitops_agent.Factory{
			GetObjectsToSynchronizeRetryPeriod: defaultGetObjectsToSynchronizeRetryPeriod,
		},
		&cilium_agent.Factory{},
		&reverse_tunnel_agent.Factory{
			InternalServerConn: internalServerConn,
		},
		&kubernetes_api_agent.Factory{},
	}
	modules := make([]modagent.Module, 0, len(factories))
	for _, factory := range factories {
		moduleName := factory.Name()
		module, err := factory.New(&modagent.Config{
			Log:       a.Log.With(logz.ModuleName(moduleName)),
			AgentMeta: a.AgentMeta,
			Api: &agentAPI{
				ModuleName:      moduleName,
				Client:          gitlab_access_rpc.NewGitlabAccessClient(kasConn),
				ResponseVisitor: sv,
			},
			K8sClientGetter: a.K8sClientGetter,
			KasConn:         kasConn,
			Server:          internalServer,
			AgentName:       agentName,
		})
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}
	return modules, nil
}

func (a *App) constructKasConnection(ctx context.Context) (*grpc.ClientConn, error) {
	tokenData, err := ioutil.ReadFile(a.TokenFile)
	if err != nil {
		return nil, fmt.Errorf("token file: %v", err)
	}
	tlsConfig, err := tlstool.DefaultClientTLSConfigWithCACert(a.CACertFile)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(a.KasAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid gitlab-kas address: %v", err)
	}
	userAgent := fmt.Sprintf("agentk/%s/%s", a.AgentMeta.Version, a.AgentMeta.CommitId)
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
		grpc.WithUserAgent(userAgent),
		// keepalive.ClientParameters must be specified at least as large as what is allowed by the
		// server-side grpc.KeepaliveEnforcementPolicy
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			// kas allows min 20 seconds, trying to stay below 60 seconds (typical load-balancer timeout) and
			// above kas' server keepalive Time so that kas pings the client sometimes. This helps mitigate
			// reverse-proxies' enforced server response timeout.
			Time:                55 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithChainStreamInterceptor(
			grpc_prometheus.StreamClientInterceptor,
			grpccorrelation.StreamClientCorrelationInterceptor(grpccorrelation.WithClientName(agentName)),
			grpctool.StreamClientValidatingInterceptor,
		),
		grpc.WithChainUnaryInterceptor(
			grpc_prometheus.UnaryClientInterceptor,
			grpccorrelation.UnaryClientCorrelationInterceptor(grpccorrelation.WithClientName(agentName)),
			grpctool.UnaryClientValidatingInterceptor,
		),
	}
	var addressToDial string
	// "grpcs" is the only scheme where encryption is done by gRPC.
	// "wss" is secure too but gRPC cannot know that, so we tell it it's not.
	secure := u.Scheme == "grpcs"
	switch u.Scheme {
	case "ws", "wss":
		addressToDial = a.KasAddress
		dialer := net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		opts = append(opts, grpc.WithContextDialer(wstunnel.DialerForGRPC(defaultMaxMessageSize, &websocket.DialOptions{
			HTTPClient: &http.Client{
				Transport: &http.Transport{
					Proxy:                 http.ProxyFromEnvironment,
					DialContext:           dialer.DialContext,
					TLSClientConfig:       tlsConfig,
					MaxIdleConns:          10,
					IdleConnTimeout:       90 * time.Second,
					TLSHandshakeTimeout:   10 * time.Second,
					ResponseHeaderTimeout: 20 * time.Second,
					ExpectContinueTimeout: 20 * time.Second,
				},
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			},
			HTTPHeader: http.Header{
				"User-Agent": []string{userAgent},
			},
		})))
	case "grpc":
		addressToDial = u.Host
	case "grpcs":
		addressToDial = u.Host
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	default:
		return nil, fmt.Errorf("unsupported scheme in GitLab Kubernetes Agent Server address: %q", u.Scheme)
	}
	if !secure {
		opts = append(opts, grpc.WithInsecure())
	}
	opts = append(opts, grpc.WithPerRPCCredentials(grpctool.NewTokenCredentials(api.AgentToken(tokenData), !secure)))
	conn, err := grpc.DialContext(ctx, addressToDial, opts...)
	if err != nil {
		return nil, fmt.Errorf("gRPC.dial: %v", err)
	}
	return conn, nil
}

func (a *App) constructInternalServer(interceptorsCtx context.Context) *grpc.Server {
	return grpc.NewServer(
		grpc.ChainStreamInterceptor(
			grpc_prometheus.StreamServerInterceptor,                                       // 1. measure all invocations
			grpccorrelation.StreamServerCorrelationInterceptor(),                          // 2. add correlation id
			grpctool.StreamServerCtxAugmentingInterceptor(grpctool.LoggerInjector(a.Log)), // 3. inject logger with correlation id
			grpc_validator.StreamServerInterceptor(),
			grpctool.StreamServerCtxAugmentingInterceptor(grpctool.JoinContexts(interceptorsCtx)), // Last because it starts an extra goroutine
		),
		grpc.ChainUnaryInterceptor(
			grpc_prometheus.UnaryServerInterceptor,                                       // 1. measure all invocations
			grpccorrelation.UnaryServerCorrelationInterceptor(),                          // 2. add correlation id
			grpctool.UnaryServerCtxAugmentingInterceptor(grpctool.LoggerInjector(a.Log)), // 3. inject logger with correlation id
			grpc_validator.UnaryServerInterceptor(),
			grpctool.UnaryServerCtxAugmentingInterceptor(grpctool.JoinContexts(interceptorsCtx)), // Last because it starts an extra goroutine
		),
	)
}

func (a *App) startInternalServer(stage stager.Stage, internalServer *grpc.Server, internalListener net.Listener, interceptorsCancel context.CancelFunc) {
	grpctool.StartServer(stage, internalServer, interceptorsCancel, func() (net.Listener, error) {
		return internalListener, nil
	})
}

func (a *App) constructInternalServerConn(ctx context.Context, dialContext func(ctx context.Context, addr string) (net.Conn, error)) (grpc.ClientConnInterface, error) {
	return grpc.DialContext(ctx, "pipe",
		grpc.WithContextDialer(dialContext),
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(grpctool.RawCodec{})),
	)
}

func NewFromFlags(flagset *pflag.FlagSet, programName string, arguments []string) (cmd.Runnable, error) {
	log, level, err := logger()
	if err != nil {
		return nil, err
	}
	app := &App{
		Log:      log,
		LogLevel: level,
		AgentMeta: &modshared.AgentMeta{
			Version:      cmd.Version,
			CommitId:     cmd.Commit,
			PodNamespace: os.Getenv(envVarPodNamespace),
			PodName:      os.Getenv(envVarPodName),
		},
	}
	flagset.StringVar(&app.KasAddress, "kas-address", "", "GitLab Kubernetes Agent Server address")
	flagset.StringVar(&app.CACertFile, "ca-cert-file", "", "Optional file with X.509 certificate authority certificate in PEM format")
	flagset.StringVar(&app.TokenFile, "token-file", "", "File with access token")
	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
	kubeConfigFlags.AddFlags(flagset)
	if err := flagset.Parse(arguments); err != nil {
		return nil, err
	}
	app.K8sClientGetter = kubeConfigFlags
	return app, nil
}

func logger() (*zap.Logger, zap.AtomicLevel, error) {
	level, err := logz.LevelFromString(defaultLoggingLevel.String())
	if err != nil {
		return nil, zap.NewAtomicLevel(), err
	}
	atomicLevel := zap.NewAtomicLevelAt(level)
	return logz.LoggerWithLevel(atomicLevel), atomicLevel, nil
}
