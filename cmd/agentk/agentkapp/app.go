package agentkapp

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"gitlab.com/ash2k/gitlab-agent/pkg/agentk"
	"gitlab.com/ash2k/gitlab-agent/pkg/agentrpc"
	"google.golang.org/grpc"
)

const (
	agentgAddressEnv         = "AGENTK_AGENTG_ADDRESS"
	agentgAddressInsecureEnv = "AGENTK_AGENTG_ADDRESS_INSECURE"
)

type App struct {
	// AgentgAddress specifies the address of agentg.
	// The target name syntax is defined in
	// https://github.com/grpc/grpc/blob/master/doc/naming.md.
	AgentgAddress string
	// Insecure disables transport security.
	Insecure bool
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
	agent := agentk.Agent{
		Client: agentrpc.NewGitLabServiceClient(conn),
	}
	return agent.Run(ctx)
}

func New() (*App, error) {
	insecure := false
	insecureStr := os.Getenv(agentgAddressInsecureEnv)
	if insecureStr != "" {
		var err error
		insecure, err = strconv.ParseBool(insecureStr)
		if err != nil {
			return nil, err
		}
	}
	app := &App{
		AgentgAddress: os.Getenv(agentgAddressEnv),
		Insecure:      insecure,
	}
	return app, nil
}
