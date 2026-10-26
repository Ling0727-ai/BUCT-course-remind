package scheduler

import (
	"sync"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/internal/scheduling"
	"github.com/Ling0727-ai/go-buct-course-backend/models/CourseData"
)

var refreshScheduleChanged = scheduling.RefreshChanged

// NotifyRefreshScheduleChanged wakes the scheduler after credentials are added or changed.
func NotifyRefreshScheduleChanged() {
	scheduling.NotifyRefresh()
}

const (
	// RefreshInterval 每个用户成功刷新后的自动刷新间隔。
	RefreshInterval = CourseData.AutoRefreshInterval

	RemindersCollection = "scheduled_reminders"
)

// ScheduledReminder 对应 Python scheduled_reminders 集合文档
type ScheduledReminder struct {
	ID            interface{} `bson:"_id,omitempty"`
	UserID        string      `bson:"user_id"`
	Type          string      `bson:"type"`      // "assignment" | "todo"
	TargetID      string      `bson:"target_id"` // task_id
	Email         string      `bson:"email"`
	Message       string      `bson:"message"`
	ScheduledTime time.Time   `bson:"scheduled_time"`
	Status        string      `bson:"status"` // "scheduled" | "processing" | "sent" | "failed" | "cancelled"
	AutoCreated   bool        `bson:"auto_created"`
	OffsetHours   int         `bson:"offset_hours"`
	DeadlineAt    *time.Time  `bson:"deadline_at,omitempty"`
	CancelReason  string      `bson:"cancel_reason,omitempty"`
	Error         string      `bson:"error,omitempty"`
	SentAt        *time.Time  `bson:"sent_at,omitempty"`
	CreatedAt     time.Time   `bson:"created_at"`
	UpdatedAt     time.Time   `bson:"updated_at"`
}

// CourseDataScheduler 课程数据定时刷新调度器，对应 Python CourseDataScheduler
type CourseDataScheduler struct {
	running bool
	stopCh  chan struct{}
	mu      sync.Mutex
}

// 全局单例，对应 Python _scheduler_instance
var (
	instance *CourseDataScheduler
	once     sync.Once
)

// GetScheduler 获取调度器单例，对应 Python get_scheduler()
func GetScheduler() *CourseDataScheduler {
	once.Do(func() {
		instance = &CourseDataScheduler{
			stopCh: make(chan struct{}),
		}
	})
	return instance
}

// Start 启动调度器，对应 Python start()
func (s *CourseDataScheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	go s.run(s.stopCh)
}

// Stop 停止调度器，对应 Python stop()
func (s *CourseDataScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}
	s.running = false
	close(s.stopCh)
}

// IsRunning 返回调度器是否正在运行
func (s *CourseDataScheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// TriggerRefresh 立即在后台触发一轮数据刷新（供管理员接口调用）
func (s *CourseDataScheduler) TriggerRefresh() {
	s.runRefreshCycle()
}
