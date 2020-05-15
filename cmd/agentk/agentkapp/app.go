package agentkapp

import (
	"context"
	"flag"
	"fmt"

	"gitlab.com/ash2k/gitlab-agent/cmd"
	"gitlab.com/ash2k/gitlab-agent/pkg/agentk"
	"gitlab.com/ash2k/gitlab-agent/pkg/agentrpc"
	"google.golang.org/grpc"
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

func NewFromFlags(flagset *flag.FlagSet, arguments []string) (cmd.Runnable, error) {
	app := &App{}
	flagset.BoolVar(&app.Insecure, "agentg-insecure", false, "Disable transport security for agentg connection")
	flagset.StringVar(&app.AgentgAddress, "agentg-address", "", "Agentg address. Syntax is described at https://github.com/grpc/grpc/blob/master/doc/naming.md")
	if err := flagset.Parse(arguments); err != nil {
		return nil, err
	}
	return app, nil
}
