package errz

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	_ error = CorrelationError{}
)

func Test_CorrelationError_Unwrap(t *testing.T) {
	e := CorrelationError{
		Err:           context.Canceled,
		CorrelationId: "123123",
	}
	assert.Equal(t, context.Canceled, e.Unwrap())
	assert.True(t, errors.Is(e, context.Canceled))
}

func Test_CorrelationError_String(t *testing.T) {
	t.Run("without id", func(t *testing.T) {
		e := CorrelationError{
			Err: context.Canceled,
		}
		assert.EqualError(t, e, context.Canceled.Error())
	})
	t.Run("with id", func(t *testing.T) {
		e := CorrelationError{
			Err:           context.Canceled,
			CorrelationId: "123123",
		}
		assert.EqualError(t, e, context.Canceled.Error()) // id is not added
	})
}
