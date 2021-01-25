package agent

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
)

var (
	_ modagent.Module  = &module{}
	_ modagent.Factory = &Factory{}
)

func TestPropagateUntilStop(t *testing.T) {
	ctxParent, cancelParent := context.WithCancel(context.Background())
	ctx, cancel, stop := propagateUntil(ctxParent)
	stop()
	cancelParent()
	select {
	case <-ctx.Done():
		require.FailNow(t, "Unexpected context cancellation")
	default:
	}
	cancel()
	<-ctx.Done()
}

func TestPropagateUntilNoStop(t *testing.T) {
	ctxParent, cancelParent := context.WithCancel(context.Background())
	cancelParent()
	ctx, cancel, _ := propagateUntil(ctxParent)
	defer cancel()
	<-ctx.Done()
}

func TestConnectUnblocksIfNotStartedStreaming(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, _, _, c := setupConnection(t)

	client.EXPECT().
		Connect(gomock.Any()).
		DoAndReturn(func(ctx context.Context, opts ...grpc.CallOption) (rpc.ReverseTunnel_ConnectClient, error) {
			cancel()
			<-ctx.Done()
			return nil, ctx.Err()
		})

	err := c.attempt(ctx)
	require.EqualError(t, err, "Connect(): context canceled")
}

// Visitor can get io.EOF after getting rpc.RequestInfo if client sent an rpc.Error, which was forwarded to the tunnel
// and then the tunnel closed the stream.
func TestNoErrorOnEofAfterRequestInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, conn, tunnel, c := setupConnection(t)
	clientStream := mock_rpc.NewMockClientStream(ctrl)

	gomock.InOrder(
		clientStream.EXPECT().
			Header().
			Return(nil, errors.New("header err")),
		tunnel.EXPECT().Send(gomock.Any()),
		tunnel.EXPECT().CloseSend(),
	)

	gomock.InOrder(
		client.EXPECT().
			Connect(gomock.Any()).
			Return(tunnel, nil),
		tunnel.EXPECT().
			Send(gomock.Any()),
		tunnel.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(&rpc.ConnectResponse{
				Msg: &rpc.ConnectResponse_RequestInfo{
					RequestInfo: &rpc.RequestInfo{},
				},
			})),
		conn.EXPECT().
			NewStream(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(clientStream, nil),
		tunnel.EXPECT().
			RecvMsg(gomock.Any()).
			Return(io.EOF),
	)

	err := c.attempt(ctx)
	require.NoError(t, err)
}

// Visitor can get io.EOF after getting rpc.Message if client sent an rpc.Error, which was forwarded to the tunnel
// and then the tunnel closed the stream.
func TestNoErrorOnEofAfterMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, conn, tunnel, c := setupConnection(t)
	clientStream := mock_rpc.NewMockClientStream(ctrl)

	gomock.InOrder(
		clientStream.EXPECT().
			Header().
			Return(nil, errors.New("header err")),
		tunnel.EXPECT().Send(gomock.Any()),
		tunnel.EXPECT().CloseSend(),
	)

	gomock.InOrder(
		client.EXPECT().
			Connect(gomock.Any()).
			Return(tunnel, nil),
		tunnel.EXPECT().
			Send(gomock.Any()),
		tunnel.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(&rpc.ConnectResponse{
				Msg: &rpc.ConnectResponse_RequestInfo{
					RequestInfo: &rpc.RequestInfo{},
				},
			})),
		conn.EXPECT().
			NewStream(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(clientStream, nil),
		tunnel.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(&rpc.ConnectResponse{
				Msg: &rpc.ConnectResponse_Message{
					Message: &rpc.Message{Data: []byte{1, 2, 3}},
				},
			})),
		clientStream.EXPECT().
			SendMsg(gomock.Any()),
		tunnel.EXPECT().
			RecvMsg(gomock.Any()).
			Return(io.EOF),
	)

	err := c.attempt(ctx)
	require.NoError(t, err)
}

func TestRecvMsgUnblocksIfNotStartedStreaming(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, _, tunnel, c := setupConnection(t)

	var connectCtx context.Context

	gomock.InOrder(
		client.EXPECT().
			Connect(gomock.Any()).
			DoAndReturn(func(ctx context.Context, opts ...grpc.CallOption) (rpc.ReverseTunnel_ConnectClient, error) {
				connectCtx = ctx
				return tunnel, nil
			}),
		tunnel.EXPECT().
			Send(gomock.Any()),
		tunnel.EXPECT().
			RecvMsg(gomock.Any()).
			DoAndReturn(func(m interface{}) error {
				cancel()
				<-connectCtx.Done()
				return connectCtx.Err()
			}),
	)

	err := c.attempt(ctx)
	require.EqualError(t, err, "context canceled")
}

func TestContextIgnoredIfStartedStreaming(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, conn, tunnel, c := setupConnection(t)
	clientStream := mock_rpc.NewMockClientStream(ctrl)

	gomock.InOrder(
		clientStream.EXPECT().
			Header().
			Return(nil, errors.New("header err")),
		tunnel.EXPECT().Send(gomock.Any()),
		tunnel.EXPECT().CloseSend(),
	)
	var connectCtx context.Context

	gomock.InOrder(
		client.EXPECT().
			Connect(gomock.Any()).
			DoAndReturn(func(ctx context.Context, opts ...grpc.CallOption) (rpc.ReverseTunnel_ConnectClient, error) {
				connectCtx = ctx
				return tunnel, nil
			}),
		tunnel.EXPECT().
			Send(gomock.Any()),
		tunnel.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(&rpc.ConnectResponse{
				Msg: &rpc.ConnectResponse_RequestInfo{
					RequestInfo: &rpc.RequestInfo{},
				},
			})),
		conn.EXPECT().
			NewStream(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(clientStream, nil),
		tunnel.EXPECT().
			RecvMsg(gomock.Any()).
			DoAndReturn(func(m interface{}) error {
				cancel()
				select {
				case <-connectCtx.Done():
					require.FailNow(t, "Unexpected context cancellation")
				default:
				}
				return errors.New("expected err")
			}),
	)

	err := c.attempt(ctx)
	require.EqualError(t, err, "expected err")
}

func TestAgentDescriptorIsSent(t *testing.T) {
	client, _, tunnel, c := setupConnection(t)
	gomock.InOrder(
		client.EXPECT().
			Connect(gomock.Any()).
			Return(tunnel, nil),
		tunnel.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ConnectRequest{
				Msg: &rpc.ConnectRequest_Descriptor_{
					Descriptor_: descriptor(),
				},
			})).
			Return(errors.New("expected err")),
	)
	err := c.attempt(context.Background())
	require.EqualError(t, err, "Send(descriptor): expected err")
}

func setupConnection(t *testing.T) (*mock_reverse_tunnel.MockReverseTunnelClient, *mock_rpc.MockClientConnInterface, *mock_reverse_tunnel.MockReverseTunnel_ConnectClient, *connection) {
	ctrl := gomock.NewController(t)
	client := mock_reverse_tunnel.NewMockReverseTunnelClient(ctrl)
	conn := mock_rpc.NewMockClientConnInterface(ctrl)
	tunnel := mock_reverse_tunnel.NewMockReverseTunnel_ConnectClient(ctrl)
	sv, err := grpctool.NewStreamVisitor(&rpc.ConnectResponse{})
	require.NoError(t, err)
	c := &connection{
		log:                zaptest.NewLogger(t),
		descriptor:         descriptor(),
		client:             client,
		internalServerConn: conn,
		streamVisitor:      sv,
	}
	return client, conn, tunnel, c
}

func descriptor() *rpc.AgentDescriptor {
	return &rpc.AgentDescriptor{
		Services: []*rpc.AgentService{
			{
				Name: "bla",
				Methods: []*rpc.ServiceMethod{
					{
						Name: "bab",
					},
				},
			},
		},
	}
}
