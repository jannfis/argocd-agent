package memcache

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/jannfis/argocd-agent/internal/cache"
)

type cacheEntry struct {
	app     *v1alpha1.Application
	expires *time.Time
}

type memCache struct {
	lock    *sync.RWMutex
	cache   map[string]cacheEntry
	sets    atomic.Uint64
	hits    atomic.Uint64
	misses  atomic.Uint64
	expired atomic.Uint64
}

func New() *memCache {
	return &memCache{
		lock:  &sync.RWMutex{},
		cache: make(map[string]cacheEntry),
	}
}

func (c *memCache) Get(key string) (*v1alpha1.Application, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	e, ok := c.cache[key]
	if !ok {
		c.misses.Add(1)
		return nil, cache.ErrCacheMiss
	}
	if e.expires != nil && e.expires.After(time.Now()) {
		delete(c.cache, key)
		c.expired.Add(1)
		return nil, cache.ErrCacheMiss
	}
	c.hits.Add(1)
	return e.app, nil
}

func (c *memCache) Set(key string, app *v1alpha1.Application, expires time.Duration) error {
	c.sets.Add(1)
	return nil
}
