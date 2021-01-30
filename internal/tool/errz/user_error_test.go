package errz

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	_ error = &UserError{}
)

func TestErrorUnwrap(t *testing.T) {
	e := &UserError{
		Cause:   context.Canceled,
		Message: "bla",
	}
	assert.Equal(t, context.Canceled, e.Unwrap())
	assert.True(t, errors.Is(e, context.Canceled))
}

func TestErrorString(t *testing.T) {
	e := &UserError{
		Message: "bla",
	}
	assert.EqualError(t, e, "bla")

	e = &UserError{
		Cause:   context.Canceled,
		Message: "bla",
	}
	assert.EqualError(t, e, "bla: context canceled")
}
