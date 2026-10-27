package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/internal/scheduling"
	"github.com/Ling0727-ai/go-buct-course-backend/models/CourseData"
	"github.com/Ling0727-ai/go-buct-course-backend/models/User"
)

var ErrRefreshQueueFull = errors.New("刷新队列繁忙，请稍后重试")

type refreshJob struct {
	userID string
	done   chan struct{}
	count  int
	err    error
}

type refreshQueue struct {
	mu      sync.Mutex
	pending map[string]*refreshJob
	jobs    chan *refreshJob
	run     func(string) (int, error)
}

func newRefreshQueue(workers, capacity int, run func(string) (int, error)) *refreshQueue {
	q := &refreshQueue{pending: make(map[string]*refreshJob), jobs: make(chan *refreshJob, capacity), run: run}
	for i := 0; i < workers; i++ {
		go q.worker()
	}
	return q
}

func (q *refreshQueue) submit(userID string) (*refreshJob, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if job := q.pending[userID]; job != nil {
		return job, nil
	}
	job := &refreshJob{userID: userID, done: make(chan struct{})}
	q.pending[userID] = job
	select {
	case q.jobs <- job:
		return job, nil
	default:
		delete(q.pending, userID)
		return nil, ErrRefreshQueueFull
	}
}

func (q *refreshQueue) worker() {
	for job := range q.jobs {
		job.count, job.err = q.execute(job.userID)
		q.mu.Lock()
		delete(q.pending, job.userID)
		close(job.done)
		q.mu.Unlock()
	}
}

func (q *refreshQueue) execute(userID string) (count int, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("刷新任务异常: %v", recovered)
			log.Print(err)
		}
	}()
	return q.run(userID)
}

var refreshQueueOnce sync.Once
var sharedRefreshQueue *refreshQueue

func getRefreshQueue() *refreshQueue {
	refreshQueueOnce.Do(func() {
		sharedRefreshQueue = newRefreshQueue(boundedEnvInt("REFRESH_WORKERS", 2, 8), boundedEnvInt("REFRESH_QUEUE_SIZE", 128, 1024), executeQueuedRefresh)
	})
	return sharedRefreshQueue
}

func boundedEnvInt(name string, fallback, maximum int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value < 1 {
		return fallback
	}
	return min(value, maximum)
}

func executeQueuedRefresh(userID string) (int, error) {
	count, err := refreshUserData(userID)
	if err != nil {
		log.Printf("[refresh] 用户 %s 刷新失败: %v", userID, err)
		retryAt := time.Now().Add(15 * time.Minute)
		if errors.Is(err, errRefreshSuperseded) {
			retryAt = time.Now()
		} else if errors.Is(err, CourseData.ErrEmptyRefreshUnconfirmed) {
			retryAt = time.Now().Add(time.Minute)
		}
		if retryErr := User.Repository.ScheduleRefresh(userID, retryAt); retryErr != nil {
			log.Printf("[refresh] 设置重试时间失败: %v", retryErr)
		}
	}
	scheduling.NotifyRefresh()
	scheduling.NotifyReminders()
	return count, err
}

// TryRefreshUserDataAsync enqueues without spawning a goroutine for every request.
func TryRefreshUserDataAsync(userID string) error {
	_, err := getRefreshQueue().submit(userID)
	return err
}

func RefreshUserDataContext(ctx context.Context, userID string) (int, error) {
	job, err := getRefreshQueue().submit(userID)
	if err != nil {
		return 0, err
	}
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-job.done:
		return job.count, job.err
	}
}
