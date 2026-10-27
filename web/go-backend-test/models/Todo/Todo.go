package Todo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Todo struct {
	ID          string     `bson:"_id,omitempty" json:"_id,omitempty"`
	UserID      string     `bson:"user_id" json:"user_id"`
	Title       string     `bson:"title" json:"title"`
	Description string     `bson:"description,omitempty" json:"description,omitempty"`
	Priority    string     `bson:"priority" json:"priority"` // low, medium, high
	DueDate     *time.Time `bson:"due_date,omitempty" json:"due_date,omitempty"`
	// EstimatedHours 存用户输入的预计小时数（对应 Python estimated_hours 字段）
	EstimatedHours *float64 `bson:"estimated_hours,omitempty" json:"estimated_hours,omitempty"`

	Completed   bool       `bson:"completed" json:"completed"`
	CompletedAt *time.Time `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
	// ExpiresAt 兼容旧数据；完成态不再设置过期时间
	ExpiresAt *time.Time `bson:"expires_at,omitempty" json:"expires_at,omitempty"`

	IsDeleted  bool       `bson:"is_deleted" json:"is_deleted"`
	DeleteTime *time.Time `bson:"delete_time,omitempty" json:"delete_time,omitempty"`
	// Forever 永久删除标记：1=正常存在，0=永久删除
	Forever int `bson:"forever" json:"forever"`

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

type TodoRepository interface {
	CreateTodo(todo *Todo) error

	GetTodoByID(id string) (*Todo, error)
	// GetTodosByUserID 获取用户正常待办（排除软删除 + 永久删除）
	GetTodosByUserID(userID string, includeCompleted bool) ([]*Todo, error)
	// GetDeletedTodosByUserID 获取软删除待办（排除永久删除）
	GetDeletedTodosByUserID(userID string) ([]*Todo, error)

	UpdateTodo(todo *Todo) error
	// DeleteTodo 软删除（is_deleted=true，forever 保持 1）
	DeleteTodo(id string) error
	// RestoreTodo 恢复软删除（is_deleted=false，forever=1，清空 delete_time）
	RestoreTodo(id string) error
	// PermanentlyDeleteTodo 永久删除（forever=0，is_deleted=true）
	PermanentlyDeleteTodo(id string) error

	// MarkCompleted 标记完成，直到用户删除或撤销
	MarkCompleted(id string) error
	MarkUncompleted(id string) error

	// ClearDeletedTodos 清空用户回收站（将所有软删除项设为 forever=0）
	ClearDeletedTodos(userID string) error
}

type MongoTodoRepository struct {
	collection *mongo.Collection
}

var Repository = &MongoTodoRepository{}

// Init 在服务启动时绑定 MongoDB collection，必须在接收任何请求前调用
func Init(db *mongo.Database) error {
	Repository.collection = db.Collection("todos")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err := Repository.collection.UpdateMany(ctx,
		bson.M{"completed": true, "expires_at": bson.M{"$exists": true}},
		bson.M{"$unset": bson.M{"expires_at": ""}})
	return err
}
