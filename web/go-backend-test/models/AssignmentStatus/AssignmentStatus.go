package AssignmentStatus

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Status 作业状态枚举
type Status = string

const (
	StatusCompleted        Status = "completed"
	StatusDeleted          Status = "deleted"
	StatusPermanentDeleted Status = "permanent_deleted"
	StatusActive           Status = "active"
)

// AssignmentStatus 对应 Python assignment_status 集合
type AssignmentStatus struct {
	ID                string     `bson:"_id,omitempty"              json:"id,omitempty"`
	UserID            string     `bson:"user_id"                    json:"user_id"`
	AssignmentID      string     `bson:"assignment_id"              json:"assignment_id"`
	AssignmentTitle   string     `bson:"assignment_title"           json:"assignment_title"`
	AssignmentSubject string     `bson:"assignment_subject"         json:"assignment_subject"`
	AssignmentType    string     `bson:"assignment_type"            json:"assignment_type"`
	Status            Status     `bson:"status"                     json:"status"`
	StatusTime        time.Time  `bson:"status_time"                json:"status_time"`
	ExpiresAt         *time.Time `bson:"expires_at,omitempty"       json:"expires_at,omitempty"`
	TodoExpiresAt     *time.Time `bson:"todo_expires_at,omitempty"  json:"todo_expires_at,omitempty"`
	Forever           int        `bson:"forever"                    json:"forever"`
	UpdatedAt         time.Time  `bson:"updated_at"                 json:"updated_at"`
}

// StatusStats 状态统计
type StatusStats struct {
	Completed int `json:"completed"`
	Deleted   int `json:"deleted"`
	Total     int `json:"total"`
}

type AssignmentStatusRepository interface {
	// UpdateStatus 更新或插入作业状态（核心方法）
	// status 为 "active" 时删除记录（恢复到默认状态）
	UpdateStatus(userID, assignmentID, title, subject, assignmentType, status string, autoExpireHours *int) error

	// MarkCompleted 标记已完成，直到作业消失或用户撤销
	MarkCompleted(userID, assignmentID, title, subject string) error
	// MarkDeleted 软删除（待办类型 12h 后 TTL 自动删除）
	MarkDeleted(userID, assignmentID, title, subject, assignmentType string) error
	// RestoreAssignment 恢复到活跃状态（删除状态记录）
	RestoreAssignment(userID, assignmentID string) error
	// PermanentDelete 永久删除（forever=0）
	PermanentDelete(userID, assignmentID, title, subject string) error

	// GetStatus 获取单条作业状态，不存在返回 "active"
	GetStatus(userID, assignmentID string) (Status, error)
	// GetUserStatuses 获取用户状态列表，支持按 status 过滤，默认排除永久删除
	GetUserStatuses(userID string, status Status, includePermanentDeleted bool) ([]*AssignmentStatus, error)

	// GetCompletedIDs 获取用户已完成的 assignmentID 列表
	GetCompletedIDs(userID string) ([]string, error)
	// GetDeletedIDs 获取用户已删除的 assignmentID 列表
	GetDeletedIDs(userID string) ([]string, error)

	// IsCompleted 检查是否已完成
	IsCompleted(userID, assignmentID string) (bool, error)
	// IsDeleted 检查是否已删除
	IsDeleted(userID, assignmentID string) (bool, error)

	// ClearUserStatus 清空用户状态记录（设 forever=0，而非物理删除）
	ClearUserStatus(userID string, status Status) (int64, error)
	// ClearUserStatusByIDs 根据 assignmentID 列表批量物理删除状态记录
	ClearUserStatusByIDs(userID string, assignmentIDs []string) (int64, error)

	// CleanupExpired 手动清理过期记录（TTL 索引备用）
	CleanupExpired() (int64, error)

	// GetStats 获取用户状态统计
	GetStats(userID string) (*StatusStats, error)
}

type MongoAssignmentStatusRepository struct {
	collection *mongo.Collection
}

var Repository = &MongoAssignmentStatusRepository{}

// Init 在服务启动时绑定 MongoDB collection，必须在接收任何请求前调用
func Init(db *mongo.Database) error {
	Repository.collection = db.Collection("assignment_status")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err := Repository.collection.UpdateMany(ctx,
		bson.M{"status": StatusCompleted, "expires_at": bson.M{"$exists": true}},
		bson.M{"$unset": bson.M{"expires_at": ""}})
	return err
}
