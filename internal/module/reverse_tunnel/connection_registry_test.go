package reverse_tunnel

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/info"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"k8s.io/apimachinery/pkg/util/wait"
)

// "slow" tests in this file are marked for concurrent execution with t.Parallel()

const (
	serviceName    = "gitlab.service1"
	methodName     = "DoSomething"
	fullMethodName = "/gitlab.service1/DoSomething"
)

var (
	_ TunnelConnectionHandler   = &ConnectionRegistry{}
	_ IncomingConnectionHandler = &ConnectionRegistry{}
)

func TestRunUnregistersAllConnections(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	tunnel := mock_reverse_tunnel.NewMockReverseTunnel_ConnectServer(ctrl)
	tunnel.EXPECT().
		Recv().
		Return(&rpc.ConnectRequest{
			Msg: &rpc.ConnectRequest_Descriptor_{
				Descriptor_: &rpc.Descriptor{},
			},
		}, nil)
	r, err := NewConnectionRegistry(zaptest.NewLogger(t))
	require.NoError(t, err)
	var wg wait.Group
	defer wg.Wait()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond) // will unblock Run()
	defer cancel()
	wg.Start(func() {
		assert.NoError(t, r.Run(ctx))
	})
	err = r.HandleTunnelConnection(context.Background(), testhelpers.AgentInfoObj(), tunnel)
	assert.EqualError(t, err, "rpc error: code = Canceled desc = context done")
}

func TestHandleTunnelConnectionIsUnblockedByContext(t *testing.T) {
	t.Parallel()
	ctxConn, cancelConn := context.WithTimeout(context.Background(), 50*time.Millisecond) // will unblock HandleTunnelConnection()
	defer cancelConn()

	ctrl := gomock.NewController(t)
	tunnel := mock_reverse_tunnel.NewMockReverseTunnel_ConnectServer(ctrl)
	tunnel.EXPECT().
		Recv().
		Return(&rpc.ConnectRequest{
			Msg: &rpc.ConnectRequest_Descriptor_{
				Descriptor_: &rpc.Descriptor{},
			},
		}, nil)
	r, err := NewConnectionRegistry(zaptest.NewLogger(t))
	require.NoError(t, err)
	var wg wait.Group
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg.Start(func() {
		assert.NoError(t, r.Run(ctx))
	})
	err = r.HandleTunnelConnection(ctxConn, testhelpers.AgentInfoObj(), tunnel)
	assert.EqualError(t, err, "rpc error: code = Canceled desc = context done")
}

func TestHandleTunnelConnectionReturnErrOnRecvErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	tunnel := mock_reverse_tunnel.NewMockReverseTunnel_ConnectServer(ctrl)
	tunnel.EXPECT().
		Recv().
		Return(nil, errors.New("expected err"))
	r, err := NewConnectionRegistry(zaptest.NewLogger(t))
	require.NoError(t, err)
	err = r.HandleTunnelConnection(context.Background(), testhelpers.AgentInfoObj(), tunnel)
	assert.EqualError(t, err, "rpc error: code = Unavailable desc = unavailable")
}

func TestHandleTunnelConnectionReturnErrOnInvalidMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	tunnel := mock_reverse_tunnel.NewMockReverseTunnel_ConnectServer(ctrl)
	tunnel.EXPECT().
		Recv().
		Return(&rpc.ConnectRequest{
			Msg: &rpc.ConnectRequest_Header{
				Header: &rpc.Header{},
			},
		}, nil)
	r, err := NewConnectionRegistry(zaptest.NewLogger(t))
	require.NoError(t, err)
	err = r.HandleTunnelConnection(context.Background(), testhelpers.AgentInfoObj(), tunnel)
	assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = Invalid oneof value type: *rpc.ConnectRequest_Header")
}

func TestHandleTunnelConnectionIsMatchedToIncomingConnection(t *testing.T) {
	t.Parallel()
	incomingStream, tunnel, r := setupStreams(t)
	agentInfo := testhelpers.AgentInfoObj()
	var wg wait.Group
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg.Start(func() {
		assert.NoError(t, r.Run(ctx))
	})
	wg.Start(func() {
		assert.NoError(t, r.HandleTunnelConnection(context.Background(), agentInfo, tunnel))
	})
	time.Sleep(50 * time.Millisecond)
	err := r.HandleIncomingConnection(agentInfo.Id, incomingStream)
	require.NoError(t, err)
}

func TestIncomingConnectionIsMatchedToHandleTunnelConnection(t *testing.T) {
	t.Parallel()
	incomingStream, tunnel, r := setupStreams(t)
	agentInfo := testhelpers.AgentInfoObj()
	var wg wait.Group
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg.Start(func() {
		assert.NoError(t, r.Run(ctx))
	})
	wg.Start(func() {
		assert.NoError(t, r.HandleIncomingConnection(agentInfo.Id, incomingStream))
	})
	time.Sleep(50 * time.Millisecond)
	err := r.HandleTunnelConnection(context.Background(), agentInfo, tunnel)
	require.NoError(t, err)
}

func setupStreams(t *testing.T) (*mock_rpc.MockServerStream, *mock_reverse_tunnel.MockReverseTunnel_ConnectServer, *ConnectionRegistry) {
	const metaKey = "Cba"
	meta := metadata.MD{}
	meta.Set(metaKey, "3", "4")
	ctrl := gomock.NewController(t)
	sts := mock_rpc.NewMockServerTransportStream(ctrl)

	incomingCtx := grpc.NewContextWithServerTransportStream(context.Background(), sts)
	incomingCtx = metadata.NewIncomingContext(incomingCtx, meta)
	incomingStream := mock_rpc.NewMockServerStream(ctrl)
	incomingStream.EXPECT().
		Context().
		Return(incomingCtx).
		MinTimes(1)

	tunnel := mock_reverse_tunnel.NewMockReverseTunnel_ConnectServer(ctrl)
	tunnel.EXPECT().
		Recv().
		Return(&rpc.ConnectRequest{
			Msg: &rpc.ConnectRequest_Descriptor_{
				Descriptor_: &rpc.Descriptor{
					AgentDescriptor: &info.AgentDescriptor{
						Services: []*info.Service{
							{
								Name: serviceName,
								Methods: []*info.Method{
									{
										Name: methodName,
									},
								},
							},
						},
					},
				},
			},
		}, nil)
	frame := grpctool.RawFrame{
		Data: []byte{1, 2, 3},
	}
	gomock.InOrder(
		sts.EXPECT().
			Method().
			Return(fullMethodName).
			MinTimes(1),
		tunnel.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ConnectResponse{
				Msg: &rpc.ConnectResponse_RequestInfo{
					RequestInfo: &rpc.RequestInfo{
						MethodName: fullMethodName,
						Meta: map[string]*rpc.Values{
							"cba": {Value: []string{"3", "4"}},
						},
					},
				},
			})),
		incomingStream.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(&frame)),
		tunnel.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ConnectResponse{
				Msg: &rpc.ConnectResponse_Message{
					Message: &rpc.Message{
						Data: frame.Data,
					},
				},
			})),
		incomingStream.EXPECT().
			RecvMsg(gomock.Any()).
			Return(io.EOF),
		tunnel.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ConnectResponse{
				Msg: &rpc.ConnectResponse_CloseSend{
					CloseSend: &rpc.CloseSend{},
				},
			})),
	)
	gomock.InOrder(
		tunnel.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(&rpc.ConnectRequest{
				Msg: &rpc.ConnectRequest_Header{
					Header: &rpc.Header{
						Meta: map[string]*rpc.Values{
							"resp": {Value: []string{"1", "2"}},
						},
					},
				},
			})),
		incomingStream.EXPECT().
			SetHeader(metadata.MD{"resp": []string{"1", "2"}}),
		tunnel.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(&rpc.ConnectRequest{
				Msg: &rpc.ConnectRequest_Message{
					Message: &rpc.Message{
						Data: []byte{5, 6, 7},
					},
				},
			})),
		incomingStream.EXPECT().
			SendMsg(&grpctool.RawFrame{
				Data: []byte{5, 6, 7},
			}),
		tunnel.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(&rpc.ConnectRequest{
				Msg: &rpc.ConnectRequest_Trailer{
					Trailer: &rpc.Trailer{
						Meta: map[string]*rpc.Values{
							"trailer": {Value: []string{"8", "9"}},
						},
					},
				},
			})),
		incomingStream.EXPECT().
			SetTrailer(metadata.MD{"trailer": []string{"8", "9"}}),
		tunnel.EXPECT().
			RecvMsg(gomock.Any()).
			Return(io.EOF),
	)

	r, err := NewConnectionRegistry(zaptest.NewLogger(t))
	require.NoError(t, err)
	return incomingStream, tunnel, r
}
