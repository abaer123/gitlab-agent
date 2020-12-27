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
	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc"
	gitlab_access_rpc "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitlab_access/rpc"
	gitops_agent "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/agent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modshared"
	observability_agent "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/observability/agent"
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
	"k8s.io/klog/v2"
	"nhooyr.io/websocket"
)

const (
	defaultRefreshConfigurationRetryPeriod    = 10 * time.Second
	defaultGetObjectsToSynchronizeRetryPeriod = 10 * time.Second

	defaultLoggingLevel agentcfg.LoggingLevelEnum = 0 // whatever is 0 is the default value

	defaultMaxMessageSize = 10 * 1024 * 1024
	correlationClientName = "gitlab-agent"

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
	kasConn, err := a.constructKasConnection(ctx)
	if err != nil {
		return err
	}
	defer errz.SafeClose(kasConn, &retErr)
	modules, err := a.constructModules(kasConn)
	if err != nil {
		return err
	}
	refresher := configRefresher{
		Log:     a.Log,
		Modules: modules,
		ConfigurationWatcher: &rpc.ConfigurationWatcher{
			Log:         a.Log,
			AgentMeta:   a.AgentMeta,
			Client:      rpc.NewAgentConfigurationClient(kasConn),
			RetryPeriod: defaultRefreshConfigurationRetryPeriod,
		},
	}

	// Start things up.
	st := stager.New()

	// Start all modules.
	stage := st.NextStage()
	for _, module := range modules {
		stage.Go(module.Run)
	}

	// Configuration refresh stage. Starts after all modules and stops before all modules are stopped.
	stage = st.NextStage()
	stage.Go(refresher.Run)

	return st.Run(ctx)
}

func (a *App) constructModules(kasConn grpc.ClientConnInterface) ([]modagent.Module, error) {
	sv, err := grpctool.NewStreamVisitor(&gitlab_access_rpc.Response{})
	if err != nil {
		return nil, err
	}
	restConfig, err := a.K8sClientGetter.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("ToRESTConfig: %v", err)
	}
	factories := []modagent.Factory{
		//  Should be the first to configure logging ASAP
		&observability_agent.Factory{
			LogLevel: a.LogLevel,
		},
		&gitops_agent.Factory{
			EngineFactory: &gitops_agent.DefaultGitOpsEngineFactory{
				KubeClientConfig: restConfig,
			},
			GetObjectsToSynchronizeRetryPeriod: defaultGetObjectsToSynchronizeRetryPeriod,
		},
	}
	modules := make([]modagent.Module, 0, len(factories))
	for _, factory := range factories {
		module, err := factory.New(&modagent.Config{
			Log:       a.Log,
			AgentMeta: a.AgentMeta,
			Api: &agentAPI{
				ModuleName:      factory.Name(),
				Client:          gitlab_access_rpc.NewGitlabAccessClient(kasConn),
				ResponseVisitor: sv,
			},
			K8sClientGetter: a.K8sClientGetter,
			KasConn:         kasConn,
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
			grpccorrelation.StreamClientCorrelationInterceptor(grpccorrelation.WithClientName(correlationClientName)),
			grpctool.StreamClientValidatingInterceptor,
		),
		grpc.WithChainUnaryInterceptor(
			grpccorrelation.UnaryClientCorrelationInterceptor(grpccorrelation.WithClientName(correlationClientName)),
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

func NewFromFlags(flagset *pflag.FlagSet, arguments []string) (cmd.Runnable, error) {
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
