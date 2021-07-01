package agentkapp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/ash2k/stager"
	"github.com/go-logr/zapr"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_configuration/rpc"
	cilium_agent "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/cilium_alert/agent"
	gitlab_access_rpc "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitlab_access/rpc"
	gitops_agent "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops/agent"
	kubernetes_api_agent "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/kubernetes_api/agent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modshared"
	observability_agent "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/observability/agent"
	reverse_tunnel_agent "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel/agent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/tlstool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/wstunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	grpccorrelation "gitlab.com/gitlab-org/labkit/correlation/grpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	"nhooyr.io/websocket"
)

const (
	defaultLoggingLevel agentcfg.LoggingLevelEnum = 0 // whatever is 0 is the default value

	defaultMaxMessageSize = 10 * 1024 * 1024
	agentName             = "gitlab-agent"

	envVarPodNamespace = "POD_NAMESPACE"
	envVarPodName      = "POD_NAME"

	getConfigurationInitBackoff   = 10 * time.Second
	getConfigurationMaxBackoff    = 5 * time.Minute
	getConfigurationResetDuration = 10 * time.Minute
	getConfigurationBackoffFactor = 2.0
	getConfigurationJitter        = 1.0
)

type App struct {
	Log       *zap.Logger
	LogLevel  zap.AtomicLevel
	AgentMeta *modshared.AgentMeta
	// KasAddress specifies the address of kas.
	KasAddress      string
	CACertFile      string
	TokenFile       string
	K8sClientGetter genericclioptions.RESTClientGetter
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
	runner := a.newModuleRunner(modules, kasConn)

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

func (a *App) newModuleRunner(modules []modagent.Module, kasConn *grpc.ClientConn) *moduleRunner {
	return newModuleRunner(a.Log, modules, &rpc.ConfigurationWatcher{
		Log:       a.Log,
		AgentMeta: a.AgentMeta,
		Client:    rpc.NewAgentConfigurationClient(kasConn),
		Backoff: retry.NewExponentialBackoffFactory(
			getConfigurationInitBackoff,
			getConfigurationMaxBackoff,
			getConfigurationResetDuration,
			getConfigurationBackoffFactor,
			getConfigurationJitter,
		),
	})
}

func (a *App) constructModules(internalServer *grpc.Server, kasConn, internalServerConn grpc.ClientConnInterface) ([]modagent.Module, error) {
	sv, err := grpctool.NewStreamVisitor(&grpctool.HttpResponse{})
	if err != nil {
		return nil, err
	}
	k8sFactory := util.NewFactory(a.K8sClientGetter)
	fTracker := newFeatureTracker(a.Log)
	accessClient := gitlab_access_rpc.NewGitlabAccessClient(kasConn)
	factories := []modagent.Factory{
		//  Should be the first to configure logging ASAP
		&observability_agent.Factory{
			LogLevel: a.LogLevel,
		},
		&gitops_agent.Factory{},
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
				moduleName:      moduleName,
				client:          accessClient,
				responseVisitor: sv,
				featureTracker:  fTracker,
			},
			K8sUtilFactory: k8sFactory,
			KasConn:        kasConn,
			Server:         internalServer,
			AgentName:      agentName,
		})
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}
	return modules, nil
}

func (a *App) constructKasConnection(ctx context.Context) (*grpc.ClientConn, error) {
	tokenData, err := os.ReadFile(a.TokenFile)
	if err != nil {
		return nil, fmt.Errorf("token file: %w", err)
	}
	tlsConfig, err := tlstool.DefaultClientTLSConfigWithCACert(a.CACertFile)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(a.KasAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid gitlab-kas address: %w", err)
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
		return nil, fmt.Errorf("gRPC.dial: %w", err)
	}
	return conn, nil
}

func (a *App) constructInternalServer(interceptorsCtx context.Context) *grpc.Server {
	return grpc.NewServer(
		grpc.ChainStreamInterceptor(
			grpc_prometheus.StreamServerInterceptor,              // 1. measure all invocations
			grpccorrelation.StreamServerCorrelationInterceptor(), // 2. add correlation id
			grpctool.StreamServerLoggerInterceptor(a.Log),        // 3. inject logger with correlation id
			grpc_validator.StreamServerInterceptor(),
			grpctool.StreamServerCtxAugmentingInterceptor(grpctool.JoinContexts(interceptorsCtx)), // Last because it starts an extra goroutine
		),
		grpc.ChainUnaryInterceptor(
			grpc_prometheus.UnaryServerInterceptor,              // 1. measure all invocations
			grpccorrelation.UnaryServerCorrelationInterceptor(), // 2. add correlation id
			grpctool.UnaryServerLoggerInterceptor(a.Log),        // 3. inject logger with correlation id
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

func NewCommand() *cobra.Command {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
	a := App{
		AgentMeta: &modshared.AgentMeta{
			Version:      cmd.Version,
			CommitId:     cmd.Commit,
			PodNamespace: os.Getenv(envVarPodNamespace),
			PodName:      os.Getenv(envVarPodName),
		},
		K8sClientGetter: kubeConfigFlags,
	}
	c := &cobra.Command{
		Use:   "agentk",
		Short: "GitLab Kubernetes Agent",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			a.Log, a.LogLevel, err = logger()
			if err != nil {
				return err
			}
			return a.Run(cmd.Context())
		},
	}
	f := c.Flags()
	f.StringVar(&a.KasAddress, "kas-address", "", "GitLab Kubernetes Agent Server address")
	f.StringVar(&a.CACertFile, "ca-cert-file", "", "Optional file with X.509 certificate authority certificate in PEM format")
	f.StringVar(&a.TokenFile, "token-file", "", "File with access token")
	kubeConfigFlags.AddFlags(f)
	cobra.CheckErr(c.MarkFlagRequired("kas-address"))
	cobra.CheckErr(c.MarkFlagRequired("token-file"))
	return c
}

func logger() (*zap.Logger, zap.AtomicLevel, error) {
	level, err := logz.LevelFromString(defaultLoggingLevel.String())
	if err != nil {
		return nil, zap.NewAtomicLevel(), err
	}
	atomicLevel := zap.NewAtomicLevelAt(level)
	return logz.LoggerWithLevel(atomicLevel), atomicLevel, nil
}
