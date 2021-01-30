package mock_rpc

import (
	"io"

	"github.com/golang/mock/gomock"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"google.golang.org/protobuf/proto"
)

func InitMockClientStream(ctrl *gomock.Controller, eof bool, msgs ...proto.Message) (*MockClientStream, []*gomock.Call) {
	stream := NewMockClientStream(ctrl)
	var res []*gomock.Call
	for _, msg := range msgs {
		call := stream.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(msg))
		res = append(res, call)
	}
	if eof {
		call := stream.EXPECT().
			RecvMsg(gomock.Any()).
			Return(io.EOF)
		res = append(res, call)
	}
	return stream, res
}
