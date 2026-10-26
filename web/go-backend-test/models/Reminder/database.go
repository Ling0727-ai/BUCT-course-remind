package Reminder

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/internal/scheduling"
	"github.com/Ling0727-ai/go-buct-course-backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collectionName = "scheduled_reminders"

// NewMongoReminderRepository 创建 Repository 并建立索引
func NewMongoReminderRepository() *MongoReminderRepository {
	client, err := models.ConnectToDB()
	if err != nil {
		panic(err)
	}
	col := client.Database("buct-course").Collection(collectionName)
	ensureIndexes(col)
	return &MongoReminderRepository{collection: col}
}

func ensureIndexes(col *mongo.Collection) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// (user_id, target_id, auto_created) 复合索引，用于去重查询
	if _, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "target_id", Value: 1},
			{Key: "auto_created", Value: 1},
			{Key: "status", Value: 1},
		},
	}); err != nil {
		log.Printf("[reminder] 创建去重查询索引失败: %v", err)
	}
	// status + scheduled_time，供调度器查询到期提醒
	if _, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "status", Value: 1},
			{Key: "scheduled_time", Value: 1},
		},
	}); err != nil {
		log.Printf("[reminder] 创建到期时间索引失败: %v", err)
	}
}

// Create 创建新提醒
func (r *MongoReminderRepository) Create(input *CreateReminderInput) (*Reminder, error) {
	defer scheduling.NotifyReminders()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	reminder := &Reminder{
		UserID:        input.UserID,
		Email:         input.Email,
		Type:          input.Type,
		TargetID:      input.TargetID,
		Message:       input.Message,
		ScheduledTime: input.ScheduledTime,
		Status:        StatusScheduled,
		AutoCreated:   false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	result, err := r.collection.InsertOne(ctx, reminder)
	if err != nil {
		return nil, err
	}
	reminder.ID = result.InsertedID
	return reminder, nil
}

// GetByID 按 ID 查询
func (r *MongoReminderRepository) GetByID(id string) (*Reminder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var key interface{} = id
	if oid, err := primitive.ObjectIDFromHex(id); err == nil {
		key = oid
	}

	var reminder Reminder
	err := r.collection.FindOne(ctx, bson.M{"_id": key}).Decode(&reminder)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &reminder, err
}

// GetByUserID 查询用户所有提醒，支持按 status 过滤
func (r *MongoReminderRepository) GetByUserID(userID string, status Status) ([]*Reminder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userID}
	if status != "" {
		filter["status"] = status
	}

	opts := options.Find().SetSort(bson.D{{Key: "scheduled_time", Value: 1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []*Reminder
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// Delete 删除提醒（只能删除非 auto_created 的）
func (r *MongoReminderRepository) Delete(id, userID string) error {
	defer scheduling.NotifyReminders()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("无效的 ID: %s", id)
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{
		"_id":          oid,
		"user_id":      userID,
		"auto_created": false,
	})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("提醒不存在或无权限删除")
	}
	return nil
}

// ExistsAutoReminder 检查某个 target_id 是否已有 scheduled/sent 状态的自动提醒
func (r *MongoReminderRepository) ExistsAutoReminder(userID, targetID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{
		"user_id":      userID,
		"target_id":    targetID,
		"auto_created": true,
		"status":       bson.M{"$in": bson.A{StatusScheduled, StatusSent}},
	})
	return count > 0, err
}
