package grpctool_test

import (
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

// Test *test.Response as callback parameter type
func TestStreamVisitorMessageHappyPath(t *testing.T) {
	stream := setupStream(t)

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

// Test field types as callback parameter type.
func TestStreamVisitorFieldHappyPath(t *testing.T) {
	stream := setupStream(t)

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
		grpctool.WithCallback(firstNumber, func(first *test.Response_First) error {
			firstCalled++
			return nil
		}),
		grpctool.WithCallback(dataNumber, func(data *test.Response_Data) error {
			dataCalled++
			return nil
		}),
		grpctool.WithCallback(lastNumber, func(last *test.Response_Last) error {
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

// Test mixed types as callback parameter type.
func TestStreamVisitorMixedHappyPath(t *testing.T) {
	stream := setupStream(t)

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
		grpctool.WithCallback(firstNumber, func(message proto.Message) error {
			firstCalled++
			return nil
		}),
		grpctool.WithCallback(dataNumber, func(data interface{ GetData() []byte }) error {
			dataCalled++
			return nil
		}),
		grpctool.WithCallback(lastNumber, func(last interface{}) error {
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

func setupStream(t *testing.T) *mock_rpc.MockClientStream {
	ctrl := gomock.NewController(t)
	stream, calls := mock_rpc.InitMockClientStream(ctrl, true,
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
	return stream
}

func TestStreamVisitorHappyPathNoEof(t *testing.T) {
	ctrl := gomock.NewController(t)
	stream, calls := mock_rpc.InitMockClientStream(ctrl, true,
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
	stream, calls := mock_rpc.InitMockClientStream(ctrl, false,
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
	require.EqualError(t, err, "unreachable fields in oneof test.NotAllReachable.message: [1 2]")
}

func TestStreamVisitorInvalidNumber(t *testing.T) {
	ctrl := gomock.NewController(t)
	stream := mock_rpc.NewMockClientStream(ctrl)
	v, err := grpctool.NewStreamVisitor(&test.Response{})
	require.NoError(t, err)
	cb := func(message *test.Response) error {
		return nil
	}
	err = v.Visit(stream,
		grpctool.WithCallback(firstNumber, cb),
		grpctool.WithCallback(dataNumber, cb),
		grpctool.WithCallback(lastNumber, cb),
		grpctool.WithCallback(20, cb),
	)
	require.EqualError(t, err, "oneof test.Response.message does not have a field 20")
}
