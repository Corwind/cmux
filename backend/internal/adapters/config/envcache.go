package config

import (
	"log"
	"sync"
	"time"
)

// EnvCache resolves shell environment variables once at construction time
// and caches the result. After the TTL expires, the next call to Get returns
// the stale cached value and triggers a background refresh, so callers are
// never blocked by a shell spawn.
type EnvCache struct {
	mu          sync.RWMutex
	env         []string
	resolvedAt  time.Time
	ttl         time.Duration
	resolver    func() []string
	refreshing  bool // true while a background refresh is in flight
}

// NewEnvCache creates a cache and synchronously resolves the shell environment.
// This blocks until the first resolution completes, guaranteeing that Get
// never has to wait.
func NewEnvCache(resolver func() []string, ttl time.Duration) *EnvCache {
	c := &EnvCache{
		ttl:      ttl,
		resolver: resolver,
	}

	start := time.Now()
	c.env = resolver()
	c.resolvedAt = time.Now()
	log.Printf("shell env resolved in %s (%d vars)", time.Since(start), len(c.env))

	return c
}

// Get returns the cached environment. It never blocks on shell resolution.
// When the TTL has expired it returns the stale value and kicks off a
// background refresh.
func (c *EnvCache) Get() []string {
	c.mu.RLock()
	env := c.env
	expired := time.Since(c.resolvedAt) >= c.ttl
	refreshing := c.refreshing
	c.mu.RUnlock()

	if expired && !refreshing {
		c.mu.Lock()
		// Double-check: another goroutine may have started a refresh.
		if time.Since(c.resolvedAt) >= c.ttl && !c.refreshing {
			c.refreshing = true
			go c.backgroundRefresh()
		}
		c.mu.Unlock()
	}

	return env
}

func (c *EnvCache) backgroundRefresh() {
	start := time.Now()
	env := c.resolver()

	c.mu.Lock()
	c.env = env
	c.resolvedAt = time.Now()
	c.refreshing = false
	c.mu.Unlock()

	log.Printf("shell env refreshed in %s (%d vars)", time.Since(start), len(env))
}
