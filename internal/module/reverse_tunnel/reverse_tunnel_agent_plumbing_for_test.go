package reverse_tunnel_test

import (
	"context"
	"net"
	"testing"

	"github.com/ash2k/stager"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	reverse_tunnel_agent "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/agent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modagent"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
)

func agentConstructComponents(t *testing.T, kasConn grpc.ClientConnInterface, agentApi *mock_modagent.MockAPI) (func(context.Context) error, *grpc.Server) {
	log := zaptest.NewLogger(t)
	internalListener := grpctool.NewDialListener()
	internalServer := agentConstructInternalServer(log)
	internalServerConn, err := agentConstructInternalServerConn(internalListener.DialContext)
	require.NoError(t, err)

	f := reverse_tunnel_agent.Factory{
		InternalServerConn: internalServerConn,
		NumConnections:     1,
	}
	config := &modagent.Config{
		Log:     log,
		Api:     agentApi,
		KasConn: kasConn,
		Server:  internalServer,
	}
	m, err := f.New(config)
	require.NoError(t, err)
	return func(ctx context.Context) error {
		return cmd.RunStages(ctx,
			// Start modules.
			func(stage stager.Stage) {
				stage.Go(func(ctx context.Context) error {
					return m.Run(ctx, nil)
				})
			},
			func(stage stager.Stage) {
				// Start internal gRPC server.
				agentStartInternalServer(stage, internalServer, internalListener)
			},
		)
	}, internalServer
}

func agentConstructInternalServer(log *zap.Logger) *grpc.Server {
	return grpc.NewServer(
		grpc.ChainStreamInterceptor(
			grpctool.StreamServerLoggerInterceptor(log),
			grpc_validator.StreamServerInterceptor(),
		),
		grpc.ChainUnaryInterceptor(
			grpctool.UnaryServerLoggerInterceptor(log),
			grpc_validator.UnaryServerInterceptor(),
		),
	)
}

func agentStartInternalServer(stage stager.Stage, internalServer *grpc.Server, internalListener net.Listener) {
	grpctool.StartServer(stage, internalServer, func() {}, func() (net.Listener, error) {
		return internalListener, nil
	})
}

func agentConstructInternalServerConn(dialContext func(ctx context.Context, addr string) (net.Conn, error)) (grpc.ClientConnInterface, error) {
	return grpc.DialContext(context.Background(), "pipe",
		grpc.WithContextDialer(dialContext),
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(grpctool.RawCodec{})),
	)
}
