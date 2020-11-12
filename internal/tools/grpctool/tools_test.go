package grpctool

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type customKey int

const (
	key customKey = iota
)

func TestJoinContexts(t *testing.T) {
	t.Run("inherits values", func(t *testing.T) {
		mainCtx := context.WithValue(context.Background(), key, 2)
		aux := context.Background()

		augmented, err := JoinContexts(aux)(mainCtx)
		require.NoError(t, err)
		assert.Equal(t, 2, augmented.Value(key))
	})
	t.Run("propagates cancel from main", func(t *testing.T) {
		mainCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		aux := context.Background()

		augmented, err := JoinContexts(aux)(mainCtx)
		require.NoError(t, err)

		cancel()
		<-augmented.Done() // does not block
		assert.Equal(t, context.Canceled, augmented.Err())
	})
	t.Run("propagates cancel from aux", func(t *testing.T) {
		mainCtx := context.Background()
		aux, cancel := context.WithCancel(context.Background())
		defer cancel()

		augmented, err := JoinContexts(aux)(mainCtx)
		require.NoError(t, err)

		cancel()
		<-augmented.Done() // does not block
		assert.Equal(t, context.Canceled, augmented.Err())
	})
}

func TestRequestCanceled(t *testing.T) {
	t.Run("context errors", func(t *testing.T) {
		assert.True(t, RequestCanceled(context.Canceled))
		assert.True(t, RequestCanceled(context.DeadlineExceeded))
		assert.False(t, RequestCanceled(io.EOF))
	})
	t.Run("wrapped context errors", func(t *testing.T) {
		assert.True(t, RequestCanceled(fmt.Errorf("bla: %w", context.Canceled)))
		assert.True(t, RequestCanceled(fmt.Errorf("bla: %w", context.DeadlineExceeded)))
		assert.False(t, RequestCanceled(fmt.Errorf("bla: %w", io.EOF)))
	})
	t.Run("gRPC errors", func(t *testing.T) {
		assert.True(t, RequestCanceled(status.Error(codes.Canceled, "bla")))
		assert.True(t, RequestCanceled(status.Error(codes.DeadlineExceeded, "bla")))
		assert.False(t, RequestCanceled(status.Error(codes.Unavailable, "bla")))
	})
	t.Run("wrapped gRPC errors", func(t *testing.T) {
		assert.True(t, RequestCanceled(fmt.Errorf("bla: %w", status.Error(codes.Canceled, "bla"))))
		assert.True(t, RequestCanceled(fmt.Errorf("bla: %w", status.Error(codes.DeadlineExceeded, "bla"))))
		assert.False(t, RequestCanceled(fmt.Errorf("bla: %w", status.Error(codes.Unavailable, "bla"))))
	})
}
