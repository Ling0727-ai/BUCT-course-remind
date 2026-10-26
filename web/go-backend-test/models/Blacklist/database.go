package Blacklist

import (
	"context"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/config"
	"github.com/Ling0727-ai/go-buct-course-backend/internal/usercache"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewMongoBlacklistRepository 创建仓库并建立索引
func NewMongoBlacklistRepository(col *mongo.Collection) *MongoBlacklistRepository {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// (user_id, subject_id) 唯一索引，确保幂等插入
	col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "subject_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})

	return &MongoBlacklistRepository{collection: col}
}

// AddSubject 添加 subjectID 到黑名单（已存在则直接返回，幂等）
func (r *MongoBlacklistRepository) AddSubject(userID, subjectID string) error {
	defer usercache.Invalidate(userID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	entry := Blacklist{
		ID:        config.Snowflake.GenerateID(),
		UserID:    userID,
		SubjectID: subjectID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := r.collection.InsertOne(ctx, entry)
	// 唯一索引冲突 → 已存在，视为成功
	if mongo.IsDuplicateKeyError(err) {
		return nil
	}
	return err
}

// RemoveSubject 从黑名单移除 subjectID
func (r *MongoBlacklistRepository) RemoveSubject(userID, subjectID string) error {
	defer usercache.Invalidate(userID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{
		"user_id":    userID,
		"subject_id": subjectID,
	})
	return err
}

// GetBlacklist 获取用户全部黑名单记录
func (r *MongoBlacklistRepository) GetBlacklist(userID string) ([]*Blacklist, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []*Blacklist
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// GetBlacklistedIDs 返回用户黑名单中的所有 subjectID（用于 scraper 过滤）
func (r *MongoBlacklistRepository) GetBlacklistedIDs(userID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID}, options.Find().SetProjection(bson.M{"subject_id": 1, "_id": 0}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	ids := make([]string, 0)
	for cursor.Next(ctx) {
		var row struct {
			ID string `bson:"subject_id"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		ids = append(ids, row.ID)
	}
	return ids, cursor.Err()
}

// IsBlacklisted 判断 subjectID 是否在黑名单中
func (r *MongoBlacklistRepository) IsBlacklisted(userID, subjectID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{
		"user_id":    userID,
		"subject_id": subjectID,
	})
	return count > 0, err
}

// ClearBlacklist 清空用户的全部黑名单
func (r *MongoBlacklistRepository) ClearBlacklist(userID string) error {
	defer usercache.Invalidate(userID)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.collection.DeleteMany(ctx, bson.M{"user_id": userID})
	return err
}
