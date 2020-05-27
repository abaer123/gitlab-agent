package agentgapp

import (
	"context"
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/ash2k/stager"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentg"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/wstunnel"
	gitalyauth "gitlab.com/gitlab-org/gitaly/auth"
	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc"
)

const (
	defaultReloadConfigurationPeriod = 5 * time.Minute
	defaultMaxMessageSize            = 10 * 1024 * 1024
)

type App struct {
	ListenNetwork             string
	ListenAddress             string
	ListenWebSocket           bool
	GitalyAddress             string
	GitalyToken               string
	ReloadConfigurationPeriod time.Duration
}

func (a *App) Run(ctx context.Context) error {
	// Gitaly client
	var gitalyOpts []grpc.DialOption
	if a.GitalyToken != "" {
		gitalyOpts = append(gitalyOpts, grpc.WithPerRPCCredentials(gitalyauth.RPCCredentialsV2(a.GitalyToken)))
	}
	gitalyConn, err := client.DialContext(ctx, a.GitalyAddress, gitalyOpts)
	if err != nil {
		return fmt.Errorf("gRPC.dial Gitaly: %v", err)
	}
	defer gitalyConn.Close()

	// Main logic of Agentg
	srv := &agentg.Agent{
		ReloadConfigurationPeriod: a.ReloadConfigurationPeriod,
		CommitServiceClient:       gitalypb.NewCommitServiceClient(gitalyConn),
	}

	// gRPC server
	lis, err := net.Listen(a.ListenNetwork, a.ListenAddress)
	if err != nil {
		return fmt.Errorf("listen: %v", err)
	}

	if a.ListenWebSocket {
		wsWrapper := wstunnel.ListenerWrapper{
			// TODO set timeouts
			ReadLimit: defaultMaxMessageSize,
		}
		lis = wsWrapper.Wrap(lis)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	agentrpc.RegisterGitLabServiceServer(grpcServer, srv)

	// Start things up
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
	flagset.BoolVar(&app.ListenWebSocket, "listen-websocket", false, "Enable \"gRPC through WebSocket\" listening mode. Rather than expecting gRPC directly, expect a WebSocket connection, from which a gRPC stream is then unpacked")
	flagset.StringVar(&app.GitalyAddress, "gitaly-address", "", "Gitaly address")
	flagset.StringVar(&app.GitalyToken, "gitaly-token", "", "Gitaly authentication token")
	flagset.DurationVar(&app.ReloadConfigurationPeriod, "reload-configuration-period", defaultReloadConfigurationPeriod, "How often to reload agentk configuration")
	if err := flagset.Parse(arguments); err != nil {
		return nil, err
	}
	return app, nil
}
