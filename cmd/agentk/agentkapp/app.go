package agentkapp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentk"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/api/apiutil"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/tools/wstunnel"
	"google.golang.org/grpc"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"nhooyr.io/websocket"
)

const (
	defaultMaxMessageSize = 10 * 1024 * 1024
)

type App struct {
	// KasAddress specifies the address of kas.
	KasAddress      string
	TokenFile       string
	K8sClientGetter resource.RESTClientGetter
}

func (a *App) Run(ctx context.Context) error {
	restConfig, err := a.K8sClientGetter.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("ToRESTConfig: %v", err)
	}
	tokenData, err := ioutil.ReadFile(a.TokenFile)
	if err != nil {
		return fmt.Errorf("token file: %v", err)
	}
	conn, err := a.kasConnection(ctx, string(tokenData))
	if err != nil {
		return err
	}
	defer conn.Close() // nolint: errcheck
	agent := agentk.New(agentrpc.NewKasClient(conn), &agentk.DefaultGitOpsEngineFactory{
		KubeClientConfig: restConfig,
	}, a.K8sClientGetter)
	return agent.Run(ctx)
}

func (a *App) kasConnection(ctx context.Context, token string) (*grpc.ClientConn, error) {
	u, err := url.Parse(a.KasAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid kas address: %v", err)
	}
	var opts []grpc.DialOption
	var addressToDial string
	// "grpcs" is the only scheme where encryption is done by gRPC.
	// "wss" is secure too but gRPC cannot know that, so we tell it it's not.
	secure := u.Scheme == "grpcs"
	switch u.Scheme {
	case "ws", "wss":
		addressToDial = a.KasAddress
		opts = append(opts, grpc.WithContextDialer(wstunnel.DialerForGRPC(defaultMaxMessageSize, &websocket.DialOptions{
			// TODO
		})))
	case "grpc":
		addressToDial = u.Host
	//case "grpcs":
	// TODO https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/7
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
	app := &App{}
	flagset.StringVar(&app.KasAddress, "kas-address", "", "GitLab Kubernetes Agent Server address")
	flagset.StringVar(&app.TokenFile, "token-file", "", "File with access token")
	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
	kubeConfigFlags.AddFlags(flagset)
	if err := flagset.Parse(arguments); err != nil {
		return nil, err
	}
	app.K8sClientGetter = kubeConfigFlags
	return app, nil
}
