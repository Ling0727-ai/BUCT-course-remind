package Todo

import (
	"context"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/config"
	"github.com/Ling0727-ai/go-buct-course-backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewMongoTodoRepository 创建 Repository 并建立 TTL 索引
func NewMongoTodoRepository() *MongoTodoRepository {
	client, err := models.ConnectToDB()
	if err != nil {
		panic(err)
	}

	col := client.Database("buct-course").Collection("todos")

	// 对应 Python: create_index("expires_at", expireAfterSeconds=0)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	})

	return &MongoTodoRepository{collection: col}
}

// CreateTodo 创建待办，由 database 层负责生成 ID 和时间戳
func (r *MongoTodoRepository) CreateTodo(todo *Todo) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	todo.ID = config.Snowflake.GenerateID()
	now := time.Now()
	todo.CreatedAt = now
	todo.UpdatedAt = now
	// 初始化默认值，对应 Python create_todo 的默认字段
	todo.Completed = false
	todo.IsDeleted = false
	todo.Forever = 1

	_, err := r.collection.InsertOne(ctx, todo)
	return err
}

// GetTodoByID 根据 ID 查询单条待办
func (r *MongoTodoRepository) GetTodoByID(id string) (*Todo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var todo Todo
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&todo)
	if err != nil {
		return nil, err
	}
	return &todo, nil
}

// GetTodosByUserID 获取用户正常待办（排除软删除 + 永久删除）
// 对应 Python: get_user_todos，is_deleted != true && forever != 0
func (r *MongoTodoRepository) GetTodosByUserID(userID string, includeCompleted bool) ([]*Todo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"user_id":    userID,
		"is_deleted": bson.M{"$ne": true},
		"forever":    bson.M{"$ne": 0},
	}
	if !includeCompleted {
		filter["completed"] = false
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var todos []*Todo
	if err = cursor.All(ctx, &todos); err != nil {
		return nil, err
	}
	return todos, nil
}

// GetDeletedTodosByUserID 获取软删除待办（排除永久删除）
// 对应 Python: get_deleted_todos，is_deleted==true && forever != 0
func (r *MongoTodoRepository) GetDeletedTodosByUserID(userID string) ([]*Todo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"user_id":    userID,
		"is_deleted": true,
		"forever":    bson.M{"$ne": 0},
	}
	opts := options.Find().SetSort(bson.D{{Key: "delete_time", Value: -1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var todos []*Todo
	if err = cursor.All(ctx, &todos); err != nil {
		return nil, err
	}
	return todos, nil
}

// UpdateTodo 整体更新一条待办（自动更新 updated_at）
func (r *MongoTodoRepository) UpdateTodo(todo *Todo) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	todo.UpdatedAt = time.Now()
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": todo.ID}, bson.M{"$set": todo})
	return err
}

// DeleteTodo 软删除：is_deleted=true，forever 保持 1
// 对应 Python: delete_todo
func (r *MongoTodoRepository) DeleteTodo(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"is_deleted":  true,
			"delete_time": now,
			"updated_at":  now,
		},
	})
	return err
}

// RestoreTodo 恢复软删除：is_deleted=false，forever=1，清空 delete_time
// 对应 Python: restore_todo
func (r *MongoTodoRepository) RestoreTodo(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"is_deleted":  false,
			"delete_time": nil,
			"forever":     1,
			"updated_at":  now,
		},
	})
	return err
}

// PermanentlyDeleteTodo 永久删除：forever=0，is_deleted=true
// 对应 Python: permanent_delete_todo（标记而非物理删除）
func (r *MongoTodoRepository) PermanentlyDeleteTodo(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"forever":     0,
			"is_deleted":  true,
			"delete_time": now,
			"updated_at":  now,
		},
	})
	return err
}

// MarkCompleted 标记完成，保留待办直到用户删除
// 对应 Python: mark_completed
func (r *MongoTodoRepository) MarkCompleted(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"completed":    true,
			"completed_at": now,
			"updated_at":   now,
		},
		"$unset": bson.M{"expires_at": ""},
	})
	return err
}

// MarkUncompleted 撤销完成状态，清除 expires_at
// 对应 Python: mark_uncompleted
func (r *MongoTodoRepository) MarkUncompleted(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"completed":    false,
			"completed_at": nil,
			"updated_at":   now,
		},
		"$unset": bson.M{"expires_at": ""},
	})
	return err
}

// ClearDeletedTodos 清空回收站：将所有软删除项设为 forever=0
// 对应 Python: clear_deleted_todos（update_many 而非物理删除）
func (r *MongoTodoRepository) ClearDeletedTodos(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	_, err := r.collection.UpdateMany(ctx,
		bson.M{
			"user_id":    userID,
			"is_deleted": true,
			"forever":    bson.M{"$ne": 0},
		},
		bson.M{
			"$set": bson.M{
				"forever":    0,
				"updated_at": now,
			},
		},
	)
	return err
}
