package mock_rpc

import (
	"io"
	"reflect"

	"github.com/golang/mock/gomock"
	"google.golang.org/protobuf/proto"
)

func InitMockClientStream(ctrl *gomock.Controller, eof bool, msgs ...proto.Message) (*MockClientStream, []*gomock.Call) {
	stream := NewMockClientStream(ctrl)
	var res []*gomock.Call
	for _, msg := range msgs {
		call := stream.EXPECT().
			RecvMsg(gomock.Any()).
			Do(RetMsg(msg))
		res = append(res, call)
	}
	if eof {
		call := stream.EXPECT().
			RecvMsg(gomock.Any()).
			DoAndReturn(func(msg interface{}) error {
				return io.EOF
			})
		res = append(res, call)
	}
	return stream, res
}

func RetMsg(value proto.Message) func(interface{}) {
	return func(msg interface{}) {
		SetMsg(msg, value)
	}
}

// SetMsg sets msg to value.
// msg must be a pointer. i.e. *blaProtoMsgType
// value must of the same type as msg.
func SetMsg(msg interface{}, value proto.Message) {
	reflect.ValueOf(msg).Elem().Set(reflect.ValueOf(value).Elem())
}
