package server

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheReusesDeletesAndExpiresValues(t *testing.T) {
	cache := NewCache[int](20 * time.Millisecond)
	var loads atomic.Int32
	loader := func() (int, error) {
		return int(loads.Add(1)), nil
	}

	first, err := cache.Get("key", loader)
	if err != nil {
		t.Fatalf("first Get() error = %v", err)
	}
	second, err := cache.Get("key", loader)
	if err != nil {
		t.Fatalf("second Get() error = %v", err)
	}
	if first != 1 || second != 1 || loads.Load() != 1 {
		t.Fatalf("cached values = (%d, %d), loads = %d", first, second, loads.Load())
	}

	cache.Delete("key")
	afterDelete, err := cache.Get("key", loader)
	if err != nil {
		t.Fatalf("Get() after Delete error = %v", err)
	}
	if afterDelete != 2 {
		t.Fatalf("value after Delete = %d, want 2", afterDelete)
	}

	time.Sleep(40 * time.Millisecond)
	afterExpiry, err := cache.Get("key", loader)
	if err != nil {
		t.Fatalf("Get() after expiry error = %v", err)
	}
	if afterExpiry != 3 {
		t.Fatalf("value after expiry = %d, want 3", afterExpiry)
	}

	cache.Clear()
	afterClear, err := cache.Get("key", loader)
	if err != nil {
		t.Fatalf("Get() after Clear error = %v", err)
	}
	if afterClear != 4 {
		t.Fatalf("value after Clear = %d, want 4", afterClear)
	}
}

func TestCacheSingleflightLoadsConcurrentKeyOnce(t *testing.T) {
	cache := NewCache[int](time.Minute)
	var loads atomic.Int32
	entered := make(chan struct{})
	release := make(chan struct{})
	var enteredOnce sync.Once

	loader := func() (int, error) {
		loads.Add(1)
		enteredOnce.Do(func() { close(entered) })
		<-release
		return 42, nil
	}

	const callers = 16
	start := make(chan struct{})
	results := make(chan int, callers)
	errors := make(chan error, callers)
	var waitGroup sync.WaitGroup
	waitGroup.Add(callers)
	for range callers {
		go func() {
			defer waitGroup.Done()
			<-start
			value, err := cache.Get("shared", loader)
			if err != nil {
				errors <- err
				return
			}
			results <- value
		}()
	}

	close(start)
	<-entered
	close(release)
	waitGroup.Wait()
	close(results)
	close(errors)

	for err := range errors {
		t.Errorf("concurrent Get() error = %v", err)
	}
	for value := range results {
		if value != 42 {
			t.Errorf("concurrent value = %d, want 42", value)
		}
	}
	if loads.Load() != 1 {
		t.Fatalf("loader calls = %d, want 1", loads.Load())
	}
}
