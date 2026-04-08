package config

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEnvCacheServesFromCache(t *testing.T) {
	var callCount atomic.Int32
	resolver := func() []string {
		callCount.Add(1)
		return []string{"FOO=bar"}
	}

	cache := NewEnvCache(resolver, 5*time.Minute)

	env1 := cache.Get()
	env2 := cache.Get()
	env3 := cache.Get()

	if callCount.Load() != 1 {
		t.Errorf("expected resolver called once, got %d", callCount.Load())
	}
	for _, env := range [][]string{env1, env2, env3} {
		if len(env) != 1 || env[0] != "FOO=bar" {
			t.Errorf("unexpected env: %v", env)
		}
	}
}

func TestEnvCacheReturnsStaleAndRefreshesInBackground(t *testing.T) {
	var callCount atomic.Int32
	resolver := func() []string {
		n := callCount.Add(1)
		return []string{fmt.Sprintf("CALL=%d", n)}
	}

	cache := NewEnvCache(resolver, 50*time.Millisecond)

	env1 := cache.Get()
	if env1[0] != "CALL=1" {
		t.Errorf("expected CALL=1, got %s", env1[0])
	}

	time.Sleep(60 * time.Millisecond)

	// After TTL, Get returns stale value and triggers background refresh
	env2 := cache.Get()
	if env2[0] != "CALL=1" {
		t.Errorf("expected stale CALL=1, got %s", env2[0])
	}

	// Wait for background refresh to complete
	time.Sleep(50 * time.Millisecond)

	env3 := cache.Get()
	if env3[0] != "CALL=2" {
		t.Errorf("expected CALL=2 after background refresh, got %s", env3[0])
	}

	if callCount.Load() != 2 {
		t.Errorf("expected 2 calls, got %d", callCount.Load())
	}
}

func TestEnvCacheDoesNotRefreshBeforeTTL(t *testing.T) {
	var callCount atomic.Int32
	resolver := func() []string {
		callCount.Add(1)
		return []string{"OK=true"}
	}

	cache := NewEnvCache(resolver, 1*time.Hour)

	for i := 0; i < 10; i++ {
		cache.Get()
	}

	if callCount.Load() != 1 {
		t.Errorf("expected 1 call, got %d", callCount.Load())
	}
}

func TestEnvCacheConcurrentAccess(t *testing.T) {
	var callCount atomic.Int32
	resolver := func() []string {
		callCount.Add(1)
		time.Sleep(10 * time.Millisecond)
		return []string{"CONCURRENT=ok"}
	}

	cache := NewEnvCache(resolver, 5*time.Minute)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			env := cache.Get()
			if len(env) != 1 || env[0] != "CONCURRENT=ok" {
				t.Errorf("unexpected env: %v", env)
			}
		}()
	}
	wg.Wait()

	if callCount.Load() != 1 {
		t.Errorf("expected 1 call, got %d", callCount.Load())
	}
}

func TestEnvCacheConstructorBlocksUntilResolved(t *testing.T) {
	resolved := make(chan struct{})
	resolver := func() []string {
		defer close(resolved)
		time.Sleep(50 * time.Millisecond)
		return []string{"READY=true"}
	}

	cache := NewEnvCache(resolver, 5*time.Minute)

	// Constructor has returned, so resolver must have completed
	select {
	case <-resolved:
	default:
		t.Fatal("constructor returned before resolver completed")
	}

	env := cache.Get()
	if len(env) != 1 || env[0] != "READY=true" {
		t.Errorf("unexpected env: %v", env)
	}
}

func TestEnvCacheBackgroundRefreshOnlyOnce(t *testing.T) {
	var callCount atomic.Int32
	resolver := func() []string {
		n := callCount.Add(1)
		time.Sleep(20 * time.Millisecond) // simulate slow resolution
		return []string{fmt.Sprintf("N=%d", n)}
	}

	cache := NewEnvCache(resolver, 30*time.Millisecond)

	_ = cache.Get() // initial (already resolved in constructor)

	time.Sleep(40 * time.Millisecond) // expire TTL

	// Multiple concurrent Gets after TTL expiry should only trigger one refresh
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.Get()
		}()
	}
	wg.Wait()

	// Wait for the single background refresh to complete
	time.Sleep(50 * time.Millisecond)

	if callCount.Load() != 2 {
		t.Errorf("expected exactly 2 resolver calls (initial + 1 refresh), got %d", callCount.Load())
	}
}

func TestEnvCacheGetNeverBlocks(t *testing.T) {
	resolver := func() []string {
		time.Sleep(100 * time.Millisecond)
		return []string{"SLOW=true"}
	}

	cache := NewEnvCache(resolver, 10*time.Millisecond)

	time.Sleep(20 * time.Millisecond) // expire TTL

	// Get should return immediately with stale data, not block on refresh
	start := time.Now()
	env := cache.Get()
	elapsed := time.Since(start)

	if elapsed > 10*time.Millisecond {
		t.Errorf("Get() took %s, expected near-instant return", elapsed)
	}
	if len(env) != 1 || env[0] != "SLOW=true" {
		t.Errorf("unexpected env: %v", env)
	}
}
