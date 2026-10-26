package Reminder

import (
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Status 提醒状态枚举
type Status = string

const (
	StatusScheduled  Status = "scheduled"
	StatusProcessing Status = "processing"
	StatusSent       Status = "sent"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
)

// ReminderType 提醒类型
type ReminderType = string

const (
	TypeAssignment ReminderType = "assignment"
	TypeTodo       ReminderType = "todo"
)

// Reminder 对应 scheduled_reminders 集合文档
// 统一管理用户手动创建和调度器自动创建的提醒
type Reminder struct {
	ID            interface{}  `bson:"_id,omitempty"        json:"id,omitempty"`
	UserID        string       `bson:"user_id"              json:"user_id"`
	Type          ReminderType `bson:"type"                json:"type"`       // "assignment" | "todo"
	TargetID      string       `bson:"target_id"            json:"target_id"` // task_id / todo_id
	Email         string       `bson:"email"                json:"email"`
	Message       string       `bson:"message"              json:"message"`
	ScheduledTime time.Time    `bson:"scheduled_time"       json:"scheduled_time"` // 计划发送时间
	Status        Status       `bson:"status"               json:"status"`
	AutoCreated   bool         `bson:"auto_created"         json:"auto_created"` // true=调度器自动创建
	OffsetHours   int          `bson:"offset_hours,omitempty" json:"offset_hours,omitempty"`
	DeadlineAt    *time.Time   `bson:"deadline_at,omitempty" json:"deadline_at,omitempty"`
	CancelReason  string       `bson:"cancel_reason,omitempty" json:"cancel_reason,omitempty"`
	Error         string       `bson:"error,omitempty"      json:"error,omitempty"`
	SentAt        *time.Time   `bson:"sent_at,omitempty"    json:"sent_at,omitempty"`
	CreatedAt     time.Time    `bson:"created_at"           json:"created_at"`
	UpdatedAt     time.Time    `bson:"updated_at"           json:"updated_at"`
}

// CreateReminderInput 创建提醒的输入参数
type CreateReminderInput struct {
	UserID        string
	Email         string
	Type          ReminderType
	TargetID      string
	Message       string
	ScheduledTime time.Time
}

// ReminderRepository 提醒数据库操作接口
type ReminderRepository interface {
	// Create 创建新提醒
	Create(input *CreateReminderInput) (*Reminder, error)
	// GetByID 按 ID 查询
	GetByID(id string) (*Reminder, error)
	// GetByUserID 查询用户所有提醒，按 scheduled_time 升序
	GetByUserID(userID string, status Status) ([]*Reminder, error)
	// Delete 删除提醒（仅限用户自己创建的，非 auto_created）
	Delete(id, userID string) error
	// ExistsAutoReminder 检查某个 target_id 是否已有 scheduled/sent 状态的自动提醒
	ExistsAutoReminder(userID, targetID string) (bool, error)
}

type MongoReminderRepository struct {
	collection *mongo.Collection
}

var Repository = &MongoReminderRepository{}

// Init 在服务启动时绑定 MongoDB collection，必须在接收任何请求前调用
func Init(db *mongo.Database) {
	Repository.collection = db.Collection("scheduled_reminders")
	ensureIndexes(Repository.collection)
}
