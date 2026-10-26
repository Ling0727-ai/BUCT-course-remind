package services

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func waitJob(t *testing.T, job *refreshJob) {
	t.Helper()
	select {
	case <-job.done:
	case <-time.After(3 * time.Second):
		t.Fatal("refresh job did not complete")
	}
}

func TestRefreshQueueCoalescesSameUser(t *testing.T) {
	var calls atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	q := newRefreshQueue(2, 4, func(string) (int, error) { calls.Add(1); close(started); <-release; return 7, nil })
	job, err := q.submit("user")
	if err != nil {
		t.Fatal(err)
	}
	<-started
	for i := 0; i < 100; i++ {
		duplicate, err := q.submit("user")
		if err != nil || duplicate != job {
			t.Fatal("same-user requests were not coalesced")
		}
	}
	close(release)
	waitJob(t, job)
	close(q.jobs)
	if calls.Load() != 1 || job.count != 7 {
		t.Fatalf("calls=%d count=%d", calls.Load(), job.count)
	}
}

func TestRefreshQueueBoundsWorkAndRecoversPanics(t *testing.T) {
	started, release := make(chan struct{}, 2), make(chan struct{})
	q := newRefreshQueue(2, 2, func(user string) (int, error) {
		started <- struct{}{}
		<-release
		if user == "panic" {
			panic("upstream failure")
		}
		return 1, nil
	})
	first, _ := q.submit("panic")
	second, _ := q.submit("b")
	<-started
	<-started
	third, _ := q.submit("c")
	fourth, _ := q.submit("d")
	if _, err := q.submit("e"); !errors.Is(err, ErrRefreshQueueFull) {
		t.Fatalf("expected backpressure, got %v", err)
	}
	if duplicate, err := q.submit("c"); err != nil || duplicate != third {
		t.Fatal("queued duplicates should not consume capacity")
	}
	close(release)
	for _, job := range []*refreshJob{first, second, third, fourth} {
		waitJob(t, job)
	}
	close(q.jobs)
	if first.err == nil || second.err != nil || third.err != nil {
		t.Fatal("panic must fail one job without killing its worker")
	}
}

func TestRefreshSettingsAreBounded(t *testing.T) {
	t.Setenv("TEST_REFRESH_WORKERS", "100000")
	if got := boundedEnvInt("TEST_REFRESH_WORKERS", 2, 8); got != 8 {
		t.Fatalf("unbounded worker count: %d", got)
	}
	t.Setenv("TEST_REFRESH_WORKERS", "0")
	if got := boundedEnvInt("TEST_REFRESH_WORKERS", 2, 8); got != 2 {
		t.Fatalf("invalid worker count did not use default: %d", got)
	}
}
