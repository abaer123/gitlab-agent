package reverse_tunnel_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/ash2k/stager"
	"github.com/golang/mock/gomock"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel"
	reverse_tunnel_server "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/server"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
)

func serverConstructComponents(t *testing.T) (func(context.Context) error, grpc.ClientConnInterface, grpc.ClientConnInterface, *grpc.Server, *mock_modserver.MockAPI) {
	log := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	serverApi := mock_modserver.NewMockAPI(ctrl)
	agentServer := serverConstructAgentServer(log)
	agentServerListener := grpctool.NewDialListener()

	internalListener := grpctool.NewDialListener()
	connRegistry, err := reverse_tunnel.NewConnectionRegistry(log)
	require.NoError(t, err)

	internalServer := serverConstructInternalServer(log)
	internalServerConn, err := serverConstructInternalServerConn(internalListener.DialContext)
	require.NoError(t, err)

	serverFactory := reverse_tunnel_server.Factory{
		TunnelConnectionHandler: connRegistry,
	}
	serverConfig := &modserver.Config{
		Log: log,
		Api: serverApi,
		Config: &kascfg.ConfigurationFile{
			Agent: &kascfg.AgentCF{
				Listen: &kascfg.ListenAgentCF{
					MaxConnectionAge: durationpb.New(time.Minute),
				},
			},
		},
		AgentServer:          agentServer,
		ReverseTunnelServer:  internalServer,
		ReverseTunnelClient:  internalServerConn,
		AgentStreamForwarder: connRegistry,
	}
	serverModule, err := serverFactory.New(serverConfig)
	require.NoError(t, err)

	kasConn, err := serverConstructKasConnection(testhelpers.AgentkToken, agentServerListener.DialContext)
	require.NoError(t, err)

	registerTestingServer(internalServer, &serverTestingServer{
		streamForwarder: connRegistry,
	})

	return func(ctx context.Context) error {
		return cmd.RunStages(ctx,
			// Start things that modules use.
			func(stage stager.Stage) {
				stage.Go(connRegistry.Run)
			},
			// Start modules.
			func(stage stager.Stage) {
				stage.Go(serverModule.Run)
			},
			// Start gRPC servers.
			func(stage stager.Stage) {
				serverStartAgentServer(stage, agentServer, agentServerListener)
				serverStartInternalServer(stage, internalServer, internalListener)
			},
		)
	}, kasConn, internalServerConn, internalServer, serverApi
}

func serverConstructInternalServer(log *zap.Logger) *grpc.Server {
	return grpc.NewServer(
		grpc.ChainStreamInterceptor(
			grpctool.StreamServerCtxAugmentingInterceptor(grpctool.LoggerInjector(log)),
		),
		grpc.ChainUnaryInterceptor(
			grpctool.UnaryServerCtxAugmentingInterceptor(grpctool.LoggerInjector(log)),
		),
		// TODO Stop using the deprecated API once https://github.com/grpc/grpc-go/issues/3694 is resolved
		grpc.CustomCodec(grpctool.RawCodec{}), // nolint: staticcheck
	)
}

func serverConstructInternalServerConn(dialContext func(ctx context.Context, addr string) (net.Conn, error)) (grpc.ClientConnInterface, error) {
	return grpc.DialContext(context.Background(), "pipe",
		grpc.WithContextDialer(dialContext),
		grpc.WithInsecure(),
	)
}

func serverConstructKasConnection(agentToken api.AgentToken, dialContext func(ctx context.Context, addr string) (net.Conn, error)) (*grpc.ClientConn, error) {
	return grpc.DialContext(context.Background(), "pipe",
		grpc.WithContextDialer(dialContext),
		grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(grpctool.NewTokenCredentials(agentToken, true)),
		grpc.WithChainStreamInterceptor(
			grpctool.StreamClientValidatingInterceptor,
		),
		grpc.WithChainUnaryInterceptor(
			grpctool.UnaryClientValidatingInterceptor,
		),
	)
}

func serverStartInternalServer(stage stager.Stage, internalServer *grpc.Server, internalListener net.Listener) {
	grpctool.StartServer(stage, internalServer, func() {}, func() (net.Listener, error) {
		return internalListener, nil
	})
}

func serverConstructAgentServer(log *zap.Logger) *grpc.Server {
	return grpc.NewServer(
		grpc.ChainStreamInterceptor(
			grpctool.StreamServerAgentMDInterceptor(),
			grpctool.StreamServerCtxAugmentingInterceptor(grpctool.LoggerInjector(log)),
			grpc_validator.StreamServerInterceptor(),
		),
		grpc.ChainUnaryInterceptor(
			grpctool.UnaryServerAgentMDInterceptor(),
			grpctool.UnaryServerCtxAugmentingInterceptor(grpctool.LoggerInjector(log)),
			grpc_validator.UnaryServerInterceptor(),
		),
	)
}

func serverStartAgentServer(stage stager.Stage, agentServer *grpc.Server, agentServerListener net.Listener) {
	grpctool.StartServer(stage, agentServer, func() {}, func() (net.Listener, error) {
		return agentServerListener, nil
	})
}
