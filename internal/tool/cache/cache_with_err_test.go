package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetItem_HappyPath(t *testing.T) {
	c := NewWithError(time.Minute, time.Minute)
	item, err := c.GetItem(context.Background(), key, func() (interface{}, error) {
		return itemVal, nil
	})
	require.NoError(t, err)
	assert.Equal(t, itemVal, item)

	item, err = c.GetItem(context.Background(), key, func() (interface{}, error) {
		t.FailNow()
		return nil, nil
	})
	require.NoError(t, err)
	assert.Equal(t, itemVal, item)
}

func TestGetItem_Error(t *testing.T) {
	c := NewWithError(time.Minute, time.Minute)
	_, err := c.GetItem(context.Background(), key, func() (interface{}, error) {
		return nil, errors.New("boom")
	})
	assert.EqualError(t, err, "boom")

	_, err = c.GetItem(context.Background(), key, func() (interface{}, error) {
		t.FailNow()
		return nil, nil
	})
	assert.EqualError(t, err, "boom")
}

func TestGetItem_Context(t *testing.T) {
	c := NewWithError(time.Minute, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := make(chan struct{})
	go func() {
		<-start
		_, err := c.GetItem(ctx, key, func() (interface{}, error) {
			return "Stalemate. No-oh, too late, too late", nil
		})
		assert.Equal(t, context.Canceled, err)
	}()
	item, err := c.GetItem(context.Background(), key, func() (interface{}, error) {
		close(start)
		cancel()
		return itemVal, nil
	})
	require.NoError(t, err)
	assert.Equal(t, itemVal, item)
}
