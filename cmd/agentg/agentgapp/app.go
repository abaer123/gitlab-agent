package agentgapp

import (
	"context"
	"flag"
	"fmt"
	"net"

	"github.com/ash2k/stager"
	"gitlab.com/ash2k/gitlab-agent/cmd"
	"gitlab.com/ash2k/gitlab-agent/pkg/agentg"
	"gitlab.com/ash2k/gitlab-agent/pkg/agentrpc"
	"google.golang.org/grpc"
)

type App struct {
	ListenNetwork string
	ListenAddress string
}

func (a *App) Run(ctx context.Context) error {
	lis, err := net.Listen(a.ListenNetwork, a.ListenAddress)
	if err != nil {
		return fmt.Errorf("listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	srv := &agentg.Agent{
		// Configuration
	}
	agentrpc.RegisterGitLabServiceServer(grpcServer, srv)
	st := stager.New()
	defer st.Shutdown()
	stage := st.NextStageWithContext(ctx)
	stage.StartWithContext(func(ctx context.Context) {
		<-ctx.Done() // can be cancelled because Server() failed or because main ctx was cancelled
		grpcServer.GracefulStop()
	})
	return grpcServer.Serve(lis)
}

func NewFromFlags(flagset *flag.FlagSet, arguments []string) (cmd.Runnable, error) {
	app := &App{}
	flagset.StringVar(&app.ListenNetwork, "listen-network", "", "Network type to listen on")
	flagset.StringVar(&app.ListenAddress, "listen-address", "", "Address to listen on")
	if err := flagset.Parse(arguments); err != nil {
		return nil, err
	}
	return app, nil
}
