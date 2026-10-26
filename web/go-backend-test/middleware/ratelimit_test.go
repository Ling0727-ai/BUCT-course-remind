package middleware

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimiterConcurrentLimit(t *testing.T) {
	r := &rateLimiter{limit: 10, window: time.Minute}
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _ := r.allow("shared-ip"); ok {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 10 {
		t.Fatalf("accepted %d requests", accepted.Load())
	}
}

func TestRateLimiterCapsEntriesWithoutResettingActiveWindows(t *testing.T) {
	r := &rateLimiter{limit: 1, window: time.Minute}
	for i := 0; i < 70000; i++ {
		r.allow(fmt.Sprint(i))
	}
	for i := range r.shards {
		if len(r.shards[i].entries) > 2048 {
			t.Fatal("shard exceeded memory bound")
		}
	}
	if ok, _ := r.allow("0"); ok {
		t.Fatal("capacity pressure reset an active window")
	}
}
