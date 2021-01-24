package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
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

func setupConnection(t *testing.T) (*mock_reverse_tunnel.MockReverseTunnelClient, *mock_rpc.MockClientConnInterface, *mock_reverse_tunnel.MockReverseTunnel_ConnectClient, *connection) {
	ctrl := gomock.NewController(t)
	client := mock_reverse_tunnel.NewMockReverseTunnelClient(ctrl)
	conn := mock_rpc.NewMockClientConnInterface(ctrl)
	tunnel := mock_reverse_tunnel.NewMockReverseTunnel_ConnectClient(ctrl)
	sv, err := grpctool.NewStreamVisitor(&rpc.ConnectResponse{})
	require.NoError(t, err)
	c := &connection{
		log:                zaptest.NewLogger(t),
		client:             client,
		internalServerConn: conn,
		streamVisitor:      sv,
	}
	return client, conn, tunnel, c
}
