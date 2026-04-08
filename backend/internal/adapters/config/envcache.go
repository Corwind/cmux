package config

import (
	"log"
	"sync"
	"time"
)

// EnvCache resolves shell environment variables eagerly in the background
// at construction time and caches the result. Subsequent calls to Get
// return the cached value until the TTL expires, avoiding a shell spawn
// on every session creation.
type EnvCache struct {
	mu         sync.RWMutex
	env        []string
	resolvedAt time.Time
	ttl        time.Duration
	resolver   func() []string
	ready      chan struct{} // closed after first background resolution
}

// NewEnvCache creates a cache that immediately starts resolving the shell
// environment in a background goroutine. The result is cached for ttl.
func NewEnvCache(resolver func() []string, ttl time.Duration) *EnvCache {
	c := &EnvCache{
		ttl:      ttl,
		resolver: resolver,
		ready:    make(chan struct{}),
	}
	go c.warmUp()
	return c
}

func (c *EnvCache) warmUp() {
	start := time.Now()
	env := c.resolver()

	c.mu.Lock()
	c.env = env
	c.resolvedAt = time.Now()
	c.mu.Unlock()

	close(c.ready)
	log.Printf("shell env pre-warmed in %s (%d vars)", time.Since(start), len(env))
}

// Get returns the cached environment. It blocks until the initial background
// resolution completes. After the TTL expires, it resolves fresh inline.
func (c *EnvCache) Get() []string {
	<-c.ready

	c.mu.RLock()
	if time.Since(c.resolvedAt) < c.ttl {
		env := c.env
		c.mu.RUnlock()
		return env
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check: another goroutine may have refreshed while we waited.
	if time.Since(c.resolvedAt) < c.ttl {
		return c.env
	}

	start := time.Now()
	c.env = c.resolver()
	c.resolvedAt = time.Now()
	log.Printf("shell env refreshed in %s (%d vars)", time.Since(start), len(c.env))

	return c.env
}
