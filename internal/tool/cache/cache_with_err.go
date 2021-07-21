package cache

import (
	"context"
	"time"
)

type GetItemDirectly func() (interface{}, error)

type EntryWithErr struct {
	// Item is the cached item.
	Item interface{}
	Err  error
}

type CacheWithErr struct {
	cache  *Cache
	ttl    time.Duration
	errTtl time.Duration
}

func NewWithError(ttl, errTtl time.Duration) *CacheWithErr {
	return &CacheWithErr{
		cache:  New(minDuration(ttl, errTtl)),
		ttl:    ttl,
		errTtl: errTtl,
	}
}

func (c *CacheWithErr) GetItem(ctx context.Context, key interface{}, f GetItemDirectly) (interface{}, error) {
	if c.ttl == 0 {
		return f()
	}
	c.cache.EvictExpiredEntries()
	entry := c.cache.GetOrCreateCacheEntry(key)
	if !entry.Lock(ctx) { // a concurrent caller may be refreshing the entry. Block until exclusive access is available.
		return nil, ctx.Err()
	}
	defer entry.Unlock()
	var entryWithErr EntryWithErr
	if entry.IsNeedRefreshLocked() {
		entryWithErr.Item, entryWithErr.Err = f()
		var ttl time.Duration
		if entryWithErr.Err == nil {
			ttl = c.ttl
		} else {
			ttl = c.errTtl
		}
		entry.Item = entryWithErr
		entry.Expires = time.Now().Add(ttl)
	} else {
		entryWithErr = entry.Item.(EntryWithErr)
	}
	return entryWithErr.Item, entryWithErr.Err
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}

	return b
}
