package gitlab

import (
	"context"
	"sync"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
)

// CacheOptions holds cache behaviour configuration.
type CacheOptions struct {
	CacheTTL      time.Duration
	CacheErrorTTL time.Duration
}

// CachingClient wraps ClientInterface to add caching.
type CachingClient struct {
	client ClientInterface

	agentInfoCache        cache
	agentInfoCacheOptions CacheOptions

	projectInfoCache        cache
	projectInfoCacheOptions CacheOptions
}

type projectInfoCacheKey struct {
	agentToken api.AgentToken
	projectId  string
}

// agentInfoCacheItem holds cached information about an agent.
type agentInfoCacheItem struct {
	agentInfo *api.AgentInfo
	err       error
}

// projectInfoCacheItem holds cached information about a project.
type projectInfoCacheItem struct {
	projectInfo *api.ProjectInfo
	err         error
}

func NewCachingClient(client ClientInterface, agentInfoCacheOptions CacheOptions, projectInfoCacheOptions CacheOptions) *CachingClient {
	return &CachingClient{
		client:                  client,
		agentInfoCache:          newCache(),
		agentInfoCacheOptions:   agentInfoCacheOptions,
		projectInfoCache:        newCache(),
		projectInfoCacheOptions: projectInfoCacheOptions,
	}
}

func (c *CachingClient) Run(ctx context.Context) {
	agentInfoTimer := expirationTimer(c.agentInfoCacheOptions)
	defer agentInfoTimer.Stop()
	projectInfoTimer := expirationTimer(c.projectInfoCacheOptions)
	defer projectInfoTimer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-agentInfoTimer.C:
			c.agentInfoCache.EvictExpiredEntries()
		case <-projectInfoTimer.C:
			c.projectInfoCache.EvictExpiredEntries()
		}
	}
}

func (c *CachingClient) GetAgentInfo(ctx context.Context, agentMeta *api.AgentMeta) (*api.AgentInfo, error) {
	if c.agentInfoCacheOptions.CacheTTL == 0 {
		return c.client.GetAgentInfo(ctx, agentMeta)
	}
	entry := c.agentInfoCache.GetOrCreateCacheEntry(agentMeta.Token)
	if !entry.lock.Lock(ctx) { // a concurrent caller may be refreshing the entry. Block until exclusive access is available.
		return nil, ctx.Err()
	}
	defer entry.lock.Unlock()
	var item agentInfoCacheItem
	if entry.IsNeedRefreshLocked() {
		item.agentInfo, item.err = c.client.GetAgentInfo(ctx, agentMeta)
		var ttl time.Duration
		if item.err == nil {
			ttl = c.agentInfoCacheOptions.CacheTTL
		} else {
			ttl = c.agentInfoCacheOptions.CacheErrorTTL
		}
		entry.item = item
		entry.expires = time.Now().Add(ttl)
	} else {
		item = entry.item.(agentInfoCacheItem)
	}
	return item.agentInfo, item.err
}

func (c *CachingClient) GetProjectInfo(ctx context.Context, agentMeta *api.AgentMeta, projectId string) (*api.ProjectInfo, error) {
	if c.projectInfoCacheOptions.CacheTTL == 0 {
		return c.client.GetProjectInfo(ctx, agentMeta, projectId)
	}
	entry := c.projectInfoCache.GetOrCreateCacheEntry(projectInfoCacheKey{
		agentToken: agentMeta.Token,
		projectId:  projectId,
	})
	if !entry.lock.Lock(ctx) { // a concurrent caller may be refreshing the entry. Block until exclusive access is available.
		return nil, ctx.Err()
	}
	defer entry.lock.Unlock()
	var item projectInfoCacheItem
	if entry.IsNeedRefreshLocked() {
		item.projectInfo, item.err = c.client.GetProjectInfo(ctx, agentMeta, projectId)
		var ttl time.Duration
		if item.err == nil {
			ttl = c.projectInfoCacheOptions.CacheTTL
		} else {
			ttl = c.projectInfoCacheOptions.CacheErrorTTL
		}
		entry.item = item
		entry.expires = time.Now().Add(ttl)
	} else {
		item = entry.item.(projectInfoCacheItem)
	}
	return item.projectInfo, item.err
}

type cacheEntry struct {
	// lock protects state in this object
	lock betterMutex
	// expires holds the time when this entry should be removed from the cache.
	expires time.Time
	// item is the cached item
	item interface{}
}

func (e *cacheEntry) IsNeedRefreshLocked() bool {
	return e.IsEmptyLocked() || e.IsExpiredLocked(time.Now())
}

func (e *cacheEntry) IsEmptyLocked() bool {
	return e.item == nil
}

func (e *cacheEntry) IsExpiredLocked(t time.Time) bool {
	return e.expires.Before(t)
}

type cache struct {
	lock sync.Mutex
	data map[interface{}]*cacheEntry
}

func newCache() cache {
	return cache{
		data: make(map[interface{}]*cacheEntry),
	}
}

func (c *cache) EvictExpiredEntries() {
	c.lock.Lock()
	defer c.lock.Unlock()
	now := time.Now()
	for key, entry := range c.data {
		func() {
			if !entry.lock.TryLock() {
				// entry is busy, skip
				return
			}
			defer entry.lock.Unlock()
			if entry.IsExpiredLocked(now) {
				delete(c.data, key)
			}
		}()
	}
}

func (c *cache) GetOrCreateCacheEntry(key interface{}) *cacheEntry {
	c.lock.Lock()
	defer c.lock.Unlock()
	entry := c.data[key]
	if entry != nil {
		return entry
	}
	entry = &cacheEntry{
		lock: newBetterMutex(),
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

func expirationTimer(opts CacheOptions) *time.Timer {
	var minTTL time.Duration
	if opts.CacheTTL < opts.CacheErrorTTL {
		minTTL = opts.CacheTTL
	} else {
		minTTL = opts.CacheErrorTTL
	}
	return time.NewTimer(minTTL)
}
