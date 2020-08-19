package kasapp

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/ash2k/stager"
	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kas"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/tools/wstunnel"
	"gitlab.com/gitlab-org/gitaly/client"
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
	GitLabAddress             string
	GitLabSocket              string
	ReloadConfigurationPeriod time.Duration
}

func (a *App) Run(ctx context.Context) error {
	// Main logic of kas
	gitLabUrl, err := url.Parse(a.GitLabAddress)
	if err != nil {
		return err
	}
	// gRPC server
	lis, err := net.Listen(a.ListenNetwork, a.ListenAddress)
	if err != nil {
		return err
	}

	if a.ListenWebSocket {
		wsWrapper := wstunnel.ListenerWrapper{
			// TODO set timeouts
			ReadLimit: defaultMaxMessageSize,
		}
		lis = wsWrapper.Wrap(lis)
	}

	gitalyClientPool := client.NewPool()
	defer gitalyClientPool.Close() // nolint: errcheck
	srv := &kas.Server{
		ReloadConfigurationPeriod: a.ReloadConfigurationPeriod,
		GitalyPool: &gitaly.Pool{
			ClientPool: gitalyClientPool,
		},
		GitLabClient: gitlab.NewClient(gitLabUrl, a.GitLabSocket, fmt.Sprintf("kas/%s/%s", cmd.Version, cmd.Commit)),
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	agentrpc.RegisterKasServer(grpcServer, srv)

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

func NewFromFlags(flagset *pflag.FlagSet, arguments []string) (cmd.Runnable, error) {
	app := &App{}
	flagset.StringVar(&app.ListenNetwork, "listen-network", "tcp", "Network type to listen on. Supported values: tcp, tcp4, tcp6, unix")
	flagset.StringVar(&app.ListenAddress, "listen-address", "127.0.0.1:0", "Address to listen on")
	flagset.BoolVar(&app.ListenWebSocket, "listen-websocket", false, "Enable \"gRPC through WebSocket\" listening mode. Rather than expecting gRPC directly, expect a WebSocket connection, from which a gRPC stream is then unpacked")
	flagset.StringVar(&app.GitLabAddress, "gitlab-address", "http://localhost:8080", "GitLab address")
	flagset.StringVar(&app.GitLabSocket, "gitlab-socket", "", "Optional: Unix domain socket to dial GitLab at")
	flagset.DurationVar(&app.ReloadConfigurationPeriod, "reload-configuration-period", defaultReloadConfigurationPeriod, "How often to reload agentk configuration")
	if err := flagset.Parse(arguments); err != nil {
		return nil, err
	}
	return app, nil
}
