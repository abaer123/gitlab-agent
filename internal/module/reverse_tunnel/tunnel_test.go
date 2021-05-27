package reverse_tunnel

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_reverse_tunnel_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/testhelpers"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	_ Tunnel = &tunnel{}
)

func TestVisitorErrorIsReturnedOnErrorMessageAndReadError(t *testing.T) {
	ctrl := gomock.NewController(t)
	tunnelRetErr := make(chan error)
	tunnelStreamVisitor, err := grpctool.NewStreamVisitor(&rpc.ConnectRequest{})
	require.NoError(t, err)
	connectServer := mock_reverse_tunnel_rpc.NewMockReverseTunnel_ConnectServer(ctrl)
	incomingStream := mock_rpc.NewMockServerStream(ctrl)
	sts := mock_rpc.NewMockServerTransportStream(ctrl)
	incomingCtx := grpc.NewContextWithServerTransportStream(context.Background(), sts)
	cb := NewMockTunnelDataCallback(ctrl)
	gomock.InOrder(
		incomingStream.EXPECT().
			Context().
			Return(incomingCtx).
			MinTimes(1),
		sts.EXPECT().
			Method().
			Return("some method"),
	)
	gomock.InOrder(
		connectServer.EXPECT().
			Send(gomock.Any()),
		incomingStream.EXPECT().
			RecvMsg(gomock.Any()).
			DoAndReturn(func(x interface{}) error {
				<-tunnelRetErr // wait until the other goroutine finished
				return errors.New("failed read")
			}),
	)

	stat := &statuspb.Status{
		Code:    int32(codes.DataLoss),
		Message: "expected data loss",
	}
	gomock.InOrder(
		connectServer.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(&rpc.ConnectRequest{
				Msg: &rpc.ConnectRequest_Error{
					Error: &rpc.Error{
						Status: stat,
					},
				},
			})),
		cb.EXPECT().
			Error(matcher.ProtoEq(t, stat)),
		connectServer.EXPECT().
			RecvMsg(gomock.Any()).
			Return(errors.New("correct error")),
	)
	c := tunnel{
		tunnel:              connectServer,
		tunnelStreamVisitor: tunnelStreamVisitor,
		tunnelRetErr:        tunnelRetErr,
	}
	err = c.ForwardStream(incomingStream, cb)
	assert.EqualError(t, err, "correct error")
}
