package agentkapp

import (
	"context"
	"flag"
	"fmt"
	"net/url"

	"gitlab.com/gitlab-org/cluster-integration/kubernetes-management-ng/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/kubernetes-management-ng/gitlab-agent/pkg/agentk"
	"gitlab.com/gitlab-org/cluster-integration/kubernetes-management-ng/gitlab-agent/pkg/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/kubernetes-management-ng/gitlab-agent/pkg/wstunnel"
	"google.golang.org/grpc"
	"nhooyr.io/websocket"
)

const (
	defaultMaxMessageSize = 10 * 1024 * 1024
)

type App struct {
	// AgentgAddress specifies the address of agentg.
	AgentgAddress string
	// Insecure disables transport security.
	Insecure bool
}

func (a *App) Run(ctx context.Context) error {
	conn, err := a.agentgConnection()
	if err != nil {
		return err
	}
	defer conn.Close()
	agent := agentk.Agent{
		Client: agentrpc.NewGitLabServiceClient(conn),
	}
	return agent.Run(ctx)
}

func (a *App) agentgConnection() (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	if a.Insecure {
		opts = append(opts, grpc.WithInsecure())
	}
	u, err := url.Parse(a.AgentgAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid Agentg address: %v", err)
	}
	if u.Scheme == "ws" || u.Scheme == "wss" {
		opts = append(opts, grpc.WithContextDialer(wstunnel.DialerForGRPC(defaultMaxMessageSize, &websocket.DialOptions{
			// TODO
		})))
	}
	conn, err := grpc.Dial(a.AgentgAddress, opts...)
	if err != nil {
		return nil, fmt.Errorf("gRPC.dial: %v", err)
	}
	return conn, nil
}

func NewFromFlags(flagset *flag.FlagSet, arguments []string) (cmd.Runnable, error) {
	app := &App{}
	flagset.StringVar(&app.AgentgAddress, "agentg-address", "", "Agentg address. Syntax is described at https://github.com/grpc/grpc/blob/master/doc/naming.md")
	flagset.BoolVar(&app.Insecure, "agentg-insecure", false, "Disable transport security for agentg connection")
	if err := flagset.Parse(arguments); err != nil {
		return nil, err
	}
	return app, nil
}
