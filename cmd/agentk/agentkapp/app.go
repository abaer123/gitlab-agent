package agentkapp

import (
	"context"
	"fmt"

	"gitlab.com/ash2k/gitlab-agent/pkg/agentk"
	"gitlab.com/ash2k/gitlab-agent/pkg/agentrpc"
	"google.golang.org/grpc"
)

type App struct {
	AgentgAddress string
	Insecure      bool
}

func (a *App) Run(ctx context.Context) error {
	var opts []grpc.DialOption
	if a.Insecure {
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(a.AgentgAddress, opts...)
	if err != nil {
		return fmt.Errorf("gRPC.dial: %v", err)
	}
	defer conn.Close()
	client := agentrpc.NewReverseProxyServiceClient(conn)
	agent := agentk.Agent{
		Client: client,
	}
	return agent.Run(ctx)
}
