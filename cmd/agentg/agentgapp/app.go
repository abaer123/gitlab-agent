package agentgapp

import (
	"context"
	"flag"
	"fmt"
	"net"

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
	serveDone := make(chan struct{})
	defer close(serveDone)
	go func() {
		select {
		case <-ctx.Done():
			grpcServer.GracefulStop()
		case <-serveDone:
			// grpcServer.Serve returned earlier than ctx was done.
			// Unblock this goroutine.
		}
	}()
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
