package agentkapp

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/go-logr/zapr"
	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentk"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api/apiutil"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/gitops/gitops_agent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/observability/observability_agent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/tlstool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/wstunnel"
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
)

type App struct {
	Log      *zap.Logger
	LogLevel zap.AtomicLevel
	// KasAddress specifies the address of kas.
	KasAddress      string
	CACertFile      string
	TokenFile       string
	K8sClientGetter resource.RESTClientGetter
}

func (a *App) Run(ctx context.Context) error {
	defer a.Log.Sync() // nolint: errcheck
	// Kubernetes uses klog so here we pipe all logs from it to our logger via an adapter.
	klog.SetLogger(zapr.NewLogger(a.Log))
	restConfig, err := a.K8sClientGetter.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("ToRESTConfig: %v", err)
	}
	tokenData, err := ioutil.ReadFile(a.TokenFile)
	if err != nil {
		return fmt.Errorf("token file: %v", err)
	}
	tlsConfig, err := tlstool.DefaultClientTLSConfigWithCACert(a.CACertFile)
	if err != nil {
		return err
	}
	kasConn, err := kasConnection(ctx, a.KasAddress, string(tokenData), tlsConfig)
	if err != nil {
		return err
	}
	defer kasConn.Close() // nolint: errcheck
	agent := agentk.Agent{
		Log:             a.Log,
		KasConn:         kasConn,
		K8sClientGetter: a.K8sClientGetter,
		ConfigurationWatcher: &agentrpc.ConfigurationWatcher{
			Log:         a.Log,
			KasClient:   agentrpc.NewKasClient(kasConn),
			RetryPeriod: defaultRefreshConfigurationRetryPeriod,
		},
		ModuleFactories: []modagent.Factory{
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
		},
	}
	return agent.Run(ctx)
}

func kasConnection(ctx context.Context, kasAddress, token string, tlsConfig *tls.Config) (*grpc.ClientConn, error) {
	u, err := url.Parse(kasAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid gitlab-kas address: %v", err)
	}
	userAgent := fmt.Sprintf("agentk/%s/%s", cmd.Version, cmd.Commit)
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
		),
		grpc.WithChainUnaryInterceptor(
			grpccorrelation.UnaryClientCorrelationInterceptor(grpccorrelation.WithClientName(correlationClientName)),
		),
	}
	var addressToDial string
	// "grpcs" is the only scheme where encryption is done by gRPC.
	// "wss" is secure too but gRPC cannot know that, so we tell it it's not.
	secure := u.Scheme == "grpcs"
	switch u.Scheme {
	case "ws", "wss":
		addressToDial = kasAddress
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
	opts = append(opts, grpc.WithPerRPCCredentials(apiutil.NewTokenCredentials(token, !secure)))
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
