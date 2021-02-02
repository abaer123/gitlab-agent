package gitaly

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	_ error = &Error{}
)

func TestErrorUnwrap(t *testing.T) {
	e := &Error{
		Code:    RpcError,
		Cause:   context.Canceled,
		Message: "bla",
	}
	assert.Equal(t, context.Canceled, e.Unwrap())
	assert.True(t, errors.Is(e, context.Canceled))
}

func TestErrorString(t *testing.T) {
	e := &Error{
		Code:    RpcError,
		Message: "bla",
	}
	assert.EqualError(t, e, "RpcError: bla")

	e = &Error{
		Code:    RpcError,
		Cause:   context.Canceled,
		Message: "bla",
	}
	assert.EqualError(t, e, "RpcError: bla: context canceled")

	e = &Error{
		Code:    RpcError,
		Cause:   context.Canceled,
		Message: "bla",
		Path:    "path",
	}
	assert.EqualError(t, e, "RpcError: bla: path: context canceled")

	e = &Error{
		Code:    RpcError,
		Message: "bla",
		Path:    "path",
	}
	assert.EqualError(t, e, "RpcError: bla: path")

	e = &Error{
		Code:    RpcError,
		Cause:   context.Canceled,
		Message: "bla",
		RpcName: "GetFoo",
		Path:    "path",
	}
	assert.EqualError(t, e, "RpcError: GetFoo: bla: path: context canceled")

	e = &Error{
		Code:    RpcError,
		Message: "bla",
		RpcName: "GetFoo",
		Path:    "path",
	}
	assert.EqualError(t, e, "RpcError: GetFoo: bla: path")

	e = &Error{
		Code:    RpcError,
		Message: "bla",
		Path:    "path",
	}
	assert.EqualError(t, e, "RpcError: bla: path")

}

func TestUnknownErrorCode(t *testing.T) {
	var e ErrorCode = -1
	assert.Equal(t, "invalid ErrorCode: -1", e.String())
}

func TestErrorCodeFromError(t *testing.T) {
	e := &Error{
		Code: RpcError,
	}
	assert.Equal(t, RpcError, ErrorCodeFromError(e))

	err := fmt.Errorf("%w", e)
	assert.Equal(t, RpcError, ErrorCodeFromError(err))

	err = errors.New("bla")
	assert.Equal(t, UnknownError, ErrorCodeFromError(err))
}
