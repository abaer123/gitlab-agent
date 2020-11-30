package cache

import (
	"context"
	"sync"
	"time"
)

type Entry struct {
	// protects state in this object.
	betterMutex
	// Expires holds the time when this entry should be removed from the cache.
	Expires time.Time
	// Item is the cached item.
	Item interface{}
}

func (e *Entry) IsNeedRefreshLocked() bool {
	return e.IsEmptyLocked() || e.IsExpiredLocked(time.Now())
}

func (e *Entry) IsEmptyLocked() bool {
	return e.Item == nil
}

func (e *Entry) IsExpiredLocked(t time.Time) bool {
	return e.Expires.Before(t)
}

type Cache struct {
	lock                  sync.Mutex
	data                  map[interface{}]*Entry
	expirationCheckPeriod time.Duration
	nextExpirationCheck   time.Time
}

func New(expirationCheckPeriod time.Duration) *Cache {
	return &Cache{
		data:                  make(map[interface{}]*Entry),
		expirationCheckPeriod: expirationCheckPeriod,
	}
}

func (c *Cache) EvictExpiredEntries() {
	c.lock.Lock()
	defer c.lock.Unlock()
	now := time.Now()
	if now.Before(c.nextExpirationCheck) {
		return
	}
	c.nextExpirationCheck = now.Add(c.expirationCheckPeriod)
	for key, entry := range c.data {
		func() {
			if !entry.TryLock() {
				// entry is busy, skip
				return
			}
			defer entry.Unlock()
			if entry.IsExpiredLocked(now) {
				delete(c.data, key)
			}
		}()
	}
}

func (c *Cache) GetOrCreateCacheEntry(key interface{}) *Entry {
	c.lock.Lock()
	defer c.lock.Unlock()
	entry := c.data[key]
	if entry != nil {
		return entry
	}
	entry = &Entry{
		betterMutex: newBetterMutex(),
	}
	c.data[key] = entry
	return entry
}

// betterMutex is a non-reentrant (like sync.Mutex) mutex that (unlike sync.Mutex) allows to:
// - try to acquire the mutex in a non-blocking way i.e. returning immediately if it cannot be done.
// - try to acquire the mutex with a possibility to abort the attempt early if a context signals done.
//
// A buffered channel of size 1 is used as the mutex. Think of it as of a box - the party that has put something
// into it has acquired the mutex. To unlock it, remove the contents from the box, so that someone else can use it.
// An empty box is created in the newBetterMutex() constructor.
//
// TryLock, Lock, and Unlock provide memory access ordering guarantees by piggybacking on channel's "happens before"
// guarantees. See https://golang.org/ref/mem
type betterMutex struct {
	c chan struct{}
}

func newBetterMutex() betterMutex {
	return betterMutex{
		c: make(chan struct{}, 1), // create an empty box
	}
}

func (m betterMutex) TryLock() bool {
	select {
	case m.c <- struct{}{}: // try to put something into the box
		return true
	default: // cannot put immediately, abort
		return false
	}
}

func (m betterMutex) Lock(ctx context.Context) bool {
	select {
	case <-ctx.Done(): // abort if context signals done
		return false
	case m.c <- struct{}{}: // try to put something into the box
		return true
	}
}

func (m betterMutex) Unlock() {
	<-m.c // take something from the box
}
