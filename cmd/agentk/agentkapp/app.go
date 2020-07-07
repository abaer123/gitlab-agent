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
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/wstunnel"
	"google.golang.org/grpc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"nhooyr.io/websocket"
)

const (
	defaultMaxMessageSize = 10 * 1024 * 1024
	kubernetesFlagsPrefix = "kube-"
)

type App struct {
	// KgbAddress specifies the address of kgb.
	KgbAddress string
	// Insecure disables transport security.
	Insecure         bool
	TokenFile        string
	KubeClientConfig *rest.Config
}

func (a *App) Run(ctx context.Context) error {
	tokenData, err := ioutil.ReadFile(a.TokenFile)
	if err != nil {
		return fmt.Errorf("token file: %v", err)
	}
	conn, err := a.kgbConnection(ctx, string(tokenData))
	if err != nil {
		return err
	}
	defer conn.Close()
	agent := agentk.New(agentrpc.NewGitLabServiceClient(conn), &agentk.DefaultGitOpsEngineFactory{
		KubeClientConfig: a.KubeClientConfig,
	})
	return agent.Run(ctx)
}

func (a *App) kgbConnection(ctx context.Context, token string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(apiutil.NewTokenCredentials(token, a.Insecure)),
	}
	if a.Insecure {
		opts = append(opts, grpc.WithInsecure())
	}
	u, err := url.Parse(a.KgbAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid kgb address: %v", err)
	}
	addressToDial := u.Host
	if u.Scheme == "ws" || u.Scheme == "wss" {
		addressToDial = a.KgbAddress
		opts = append(opts, grpc.WithContextDialer(wstunnel.DialerForGRPC(defaultMaxMessageSize, &websocket.DialOptions{
			// TODO
		})))
	}
	conn, err := grpc.DialContext(ctx, addressToDial, opts...)
	if err != nil {
		return nil, fmt.Errorf("gRPC.dial: %v", err)
	}
	return conn, nil
}

func NewFromFlags(flagset *pflag.FlagSet, arguments []string) (cmd.Runnable, error) {
	app := &App{}
	flagset.StringVar(&app.KgbAddress, "kgb-address", "", "Kgb address")
	flagset.BoolVar(&app.Insecure, "kgb-insecure", false, "Disable transport security for kgb connection")
	flagset.StringVar(&app.TokenFile, "token-file", "", "File with access token")
	clientConf := addKubeFlags(flagset)
	if err := flagset.Parse(arguments); err != nil {
		return nil, err
	}
	var err error
	app.KubeClientConfig, err = clientConf.ClientConfig()
	if err != nil {
		return nil, err
	}
	return app, nil
}

// addKubeFlags adds kubectl-like flags to a flagset and returns the ClientConfig interface
// for retrieving the values.
func addKubeFlags(flagset *pflag.FlagSet) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	kflags := clientcmd.RecommendedConfigOverrideFlags(kubernetesFlagsPrefix)
	flagset.StringVar(&loadingRules.ExplicitPath, kubernetesFlagsPrefix+"config", "", "Path to a Kubernetes config. Only required if out-of-cluster")
	clientcmd.BindOverrideFlags(overrides, flagset, kflags)
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
}
