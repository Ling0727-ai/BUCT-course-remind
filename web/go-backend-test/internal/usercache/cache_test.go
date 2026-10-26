package usercache

import (
	"context"
	"testing"
	"time"
)

func TestInvalidationDuringLoadPreventsStaleRefill(t *testing.T) {
	c := New[string](2, 128)
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		_, _ = c.Get(context.Background(), "user", func() (string, int, time.Duration, error) {
			close(started)
			<-release
			return "old", 8, time.Minute, nil
		})
	}()
	<-started
	c.Invalidate("user")
	close(release)
	<-done
	value, err := c.Get(context.Background(), "user", func() (string, int, time.Duration, error) { return "new", 8, time.Minute, nil })
	if err != nil || value != "new" {
		t.Fatalf("stale refill: %q %v", value, err)
	}
}

func TestCacheHonorsLRUAndByteBudget(t *testing.T) {
	c := New[string](2, 12)
	load := func(value string) func() (string, int, time.Duration, error) {
		return func() (string, int, time.Duration, error) { return value, 6, time.Minute, nil }
	}
	for _, key := range []string{"a", "b", "a", "c"} {
		if _, err := c.Get(context.Background(), key, load(key)); err != nil {
			t.Fatal(err)
		}
	}
	c.mu.Lock()
	if c.items["b"] != nil || len(c.items) != 2 || c.bytes != 12 {
		t.Fatal("cache did not evict its least recently used item")
	}
	c.mu.Unlock()
	Invalidate("a")
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items["a"] != nil {
		t.Fatal("model invalidation did not reach cache")
	}
}
