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
	if len(env1) != 1 || env1[0] != "FOO=bar" {
		t.Errorf("unexpected env: %v", env1)
	}
	if len(env2) != 1 || env2[0] != "FOO=bar" {
		t.Errorf("unexpected env: %v", env2)
	}
	if len(env3) != 1 || env3[0] != "FOO=bar" {
		t.Errorf("unexpected env: %v", env3)
	}
}

func TestEnvCacheRefreshesAfterTTL(t *testing.T) {
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

	env2 := cache.Get()
	if env2[0] != "CALL=2" {
		t.Errorf("expected CALL=2, got %s", env2[0])
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

func TestEnvCacheBlocksUntilFirstResolution(t *testing.T) {
	resolving := make(chan struct{})
	resolver := func() []string {
		<-resolving
		return []string{"READY=true"}
	}

	cache := NewEnvCache(resolver, 5*time.Minute)

	done := make(chan []string, 1)
	go func() {
		done <- cache.Get()
	}()

	// Get() should be blocked
	select {
	case <-done:
		t.Fatal("Get() returned before resolver completed")
	case <-time.After(20 * time.Millisecond):
	}

	// Unblock the resolver
	close(resolving)

	select {
	case env := <-done:
		if len(env) != 1 || env[0] != "READY=true" {
			t.Errorf("unexpected env: %v", env)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Get() did not return after resolver completed")
	}
}

func TestEnvCacheTTLRefreshOnlyOnce(t *testing.T) {
	var callCount atomic.Int32
	resolver := func() []string {
		n := callCount.Add(1)
		time.Sleep(20 * time.Millisecond) // simulate slow resolution
		return []string{fmt.Sprintf("N=%d", n)}
	}

	cache := NewEnvCache(resolver, 30*time.Millisecond)

	_ = cache.Get() // initial

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

	if callCount.Load() != 2 {
		t.Errorf("expected exactly 2 resolver calls (initial + 1 refresh), got %d", callCount.Load())
	}
}
