package grpctool_test

import (
	"io"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool/test"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_rpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	firstNumber protoreflect.FieldNumber = 1
	dataNumber  protoreflect.FieldNumber = 2
	lastNumber  protoreflect.FieldNumber = 3
)

func TestStreamVisitorHappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	stream, calls := mockStream(ctrl, true,
		&test.Response{
			Message: &test.Response_First_{
				First: &test.Response_First{},
			},
		},
		&test.Response{
			Message: &test.Response_Data_{
				Data: &test.Response_Data{},
			},
		},
		&test.Response{
			Message: &test.Response_Data_{
				Data: &test.Response_Data{},
			},
		},
		&test.Response{
			Message: &test.Response_Last_{
				Last: &test.Response_Last{},
			},
		},
	)
	gomock.InOrder(calls...)

	var (
		eofCalled   int
		firstCalled int
		dataCalled  int
		lastCalled  int
	)
	v, err := grpctool.NewStreamVisitor(&test.Response{})
	require.NoError(t, err)
	err = v.Visit(stream,
		grpctool.WithEOFCallback(func() error {
			eofCalled++
			return nil
		}),
		grpctool.WithCallback(firstNumber, func(message *test.Response) error {
			firstCalled++
			return nil
		}),
		grpctool.WithCallback(dataNumber, func(message *test.Response) error {
			dataCalled++
			return nil
		}),
		grpctool.WithCallback(lastNumber, func(message *test.Response) error {
			lastCalled++
			return nil
		}),
	)
	require.NoError(t, err)
	assert.Equal(t, 1, firstCalled)
	assert.Equal(t, 2, dataCalled)
	assert.Equal(t, 1, lastCalled)
	assert.Equal(t, 1, eofCalled)
}

func TestStreamVisitorHappyPathNoEof(t *testing.T) {
	ctrl := gomock.NewController(t)
	stream, calls := mockStream(ctrl, true,
		&test.Response{
			Message: &test.Response_First_{
				First: &test.Response_First{},
			},
		},
		&test.Response{
			Message: &test.Response_Data_{
				Data: &test.Response_Data{},
			},
		},
		&test.Response{
			Message: &test.Response_Last_{
				Last: &test.Response_Last{},
			},
		},
	)
	gomock.InOrder(calls...)

	var (
		firstCalled int
		dataCalled  int
		lastCalled  int
	)
	v, err := grpctool.NewStreamVisitor(&test.Response{})
	require.NoError(t, err)
	err = v.Visit(stream,
		grpctool.WithCallback(firstNumber, func(message *test.Response) error {
			firstCalled++
			return nil
		}),
		grpctool.WithCallback(dataNumber, func(message *test.Response) error {
			dataCalled++
			return nil
		}),
		grpctool.WithCallback(lastNumber, func(message *test.Response) error {
			lastCalled++
			return nil
		}),
	)
	require.NoError(t, err)
	assert.Equal(t, 1, firstCalled)
	assert.Equal(t, 1, dataCalled)
	assert.Equal(t, 1, lastCalled)
}

func TestStreamVisitorMissingCallback(t *testing.T) {
	ctrl := gomock.NewController(t)
	stream, calls := mockStream(ctrl, false,
		&test.Response{
			Message: &test.Response_First_{
				First: &test.Response_First{},
			},
		},
	)
	gomock.InOrder(calls...)

	v, err := grpctool.NewStreamVisitor(&test.Response{})
	require.NoError(t, err)
	err = v.Visit(stream)
	require.EqualError(t, err, "no callback defined for field test.Response.first (1)")
}

func TestStreamVisitorNoOneofs(t *testing.T) {
	_, err := grpctool.NewStreamVisitor(&test.NoOneofs{})
	require.EqualError(t, err, "one oneof group is expected in test.NoOneofs, 0 defined")
}

func TestStreamVisitorTwoOneofs(t *testing.T) {
	_, err := grpctool.NewStreamVisitor(&test.TwoOneofs{})
	require.EqualError(t, err, "one oneof group is expected in test.TwoOneofs, 2 defined")
}

func TestStreamVisitorTwoValidOneofs(t *testing.T) {
	_, err := grpctool.NewStreamVisitor(&test.TwoValidOneofs{})
	require.EqualError(t, err, "one oneof group is expected in test.TwoValidOneofs, 2 defined")
}

func TestStreamVisitorNumberOutOfOneof(t *testing.T) {
	_, err := grpctool.NewStreamVisitor(&test.OutOfOneof{})
	require.EqualError(t, err, "field number 1 is not part of oneof test.OutOfOneof.message")
}

func TestStreamVisitorNotAllFieldsReachable(t *testing.T) {
	_, err := grpctool.NewStreamVisitor(&test.NotAllReachable{})
	require.EqualError(t, err, "not all oneof test.NotAllReachable.message fields are reachable")
}

// setMsg sets msg to value.
// msg must be a pointer. i.e. *blaProtoMsgType
// value must of the same type as msg.
func setMsg(msg, value interface{}) {
	reflect.ValueOf(msg).Elem().Set(reflect.ValueOf(value).Elem())
}

func mockStream(ctrl *gomock.Controller, eof bool, msgs ...proto.Message) (*mock_rpc.MockClientStream, []*gomock.Call) {
	stream := mock_rpc.NewMockClientStream(ctrl)
	var res []*gomock.Call
	for _, msg := range msgs {
		msg := msg // ensure the right message variable is captured by the closure below
		call := stream.EXPECT().
			RecvMsg(gomock.Any()).
			DoAndReturn(func(m interface{}) error {
				setMsg(m, msg)
				return nil
			})
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
