package grpctools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
