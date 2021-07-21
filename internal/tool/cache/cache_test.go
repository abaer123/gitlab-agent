package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	key     = 1
	itemVal = 123
)

func TestEntry(t *testing.T) {
	c := New(time.Minute)
	expires := time.Now().Add(time.Hour)
	checkEntry := func() {
		entry := c.GetOrCreateCacheEntry(key)
		entry.Lock(context.Background())
		defer entry.Unlock()
		assert.False(t, entry.IsEmptyLocked())
		assert.False(t, entry.IsExpiredLocked(time.Now()))
		assert.False(t, entry.IsNeedRefreshLocked())
		assert.Equal(t, itemVal, entry.Item)
		assert.Equal(t, expires, entry.Expires)
	}
	entry := c.GetOrCreateCacheEntry(key)
	entry.Lock(context.Background())
	defer entry.Unlock()
	go checkEntry() // started while holding the lock
	assert.True(t, entry.IsEmptyLocked())
	assert.True(t, entry.IsExpiredLocked(time.Now()))
	assert.True(t, entry.IsNeedRefreshLocked())
	entry.Item = itemVal
	entry.Expires = expires
}

func TestEvictExpiredEntries_RemovesExpired(t *testing.T) {
	c := New(time.Minute)
	func() { // init entry
		entry := c.GetOrCreateCacheEntry(key)
		entry.Lock(context.Background())
		defer entry.Unlock()
		entry.Item = itemVal
		entry.Expires = time.Now().Add(-time.Second)
	}()
	c.EvictExpiredEntries()
	entry := c.GetOrCreateCacheEntry(key)
	entry.Lock(context.Background())
	defer entry.Unlock()
	assert.Zero(t, entry.Item)
	assert.Zero(t, entry.Expires)
}

func TestEvictExpiredEntries_IgnoresBusy(t *testing.T) {
	c := New(time.Minute)
	expires := time.Now().Add(-time.Second)
	func() {
		entry := c.GetOrCreateCacheEntry(key)
		entry.Lock(context.Background())
		defer entry.Unlock()
		entry.Expires = expires
		entry.Item = itemVal
		c.EvictExpiredEntries() // evict while locked
	}()
	entry := c.GetOrCreateCacheEntry(key)
	entry.Lock(context.Background())
	defer entry.Unlock()
	assert.Equal(t, itemVal, entry.Item)
	assert.Equal(t, expires, entry.Expires)
}
