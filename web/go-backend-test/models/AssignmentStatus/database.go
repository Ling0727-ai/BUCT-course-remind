package AssignmentStatus

import (
	"context"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/config"
	"github.com/Ling0727-ai/go-buct-course-backend/internal/scheduling"
	"github.com/Ling0727-ai/go-buct-course-backend/internal/usercache"
	"github.com/Ling0727-ai/go-buct-course-backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewMongoAssignmentStatusRepository 创建 Repository 并建立索引
// 对应 Python AssignmentStatus.__init__ 中的索引创建
func NewMongoAssignmentStatusRepository() *MongoAssignmentStatusRepository {
	client, err := models.ConnectToDB()
	if err != nil {
		panic(err)
	}

	col := client.Database("buct-course").Collection("assignment_status")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 对应 Python: create_index([("user_id", 1), ("assignment_id", 1)], unique=True)
	col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "assignment_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	// 对应 Python: create_index("expires_at", expireAfterSeconds=0)
	col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0).SetSparse(true),
	})
	// 对应 Python: create_index([("user_id", 1), ("status", 1)])
	col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "status", Value: 1}},
	})
	// 对应 Python: create_index("todo_expires_at", expireAfterSeconds=0)
	col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "todo_expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0).SetSparse(true),
	})

	return &MongoAssignmentStatusRepository{collection: col}
}

func timePtr(v time.Time) *time.Time { return &v }

// ──────────────────────────────────────────────
//  核心写操作
// ──────────────────────────────────────────────

// UpdateStatus 更新或插入作业状态（核心方法）
// 对应 Python: update_assignment_status
// status 为 "active" 时删除记录（恢复到默认状态）
func (r *MongoAssignmentStatusRepository) UpdateStatus(
	userID, assignmentID, title, subject, assignmentType, status string,
	autoExpireHours *int,
) error {
	defer usercache.Invalidate(userID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// status == "active" 表示恢复，直接删除记录
	if status == StatusActive {
		_, err := r.collection.DeleteOne(ctx, bson.M{
			"user_id":       userID,
			"assignment_id": assignmentID,
		})
		if err == nil {
			_, err = r.collection.Database().Collection("course_data").UpdateOne(ctx,
				bson.M{"user_id": userID, "task_id": assignmentID, "deadline_at": bson.M{"$gt": time.Now()}},
				bson.M{"$set": bson.M{"auto_reminder_pending": true, "auto_reminder_at": time.Now()}})
			if err == nil {
				scheduling.NotifyReminders()
			}
		}
		return err
	}

	now := time.Now()
	var expiresAt *time.Time
	var todoExpiresAt *time.Time

	if status == StatusCompleted {
		autoExpireHours = nil
	}
	if autoExpireHours != nil {
		t := now.Add(time.Duration(*autoExpireHours) * time.Hour)
		expiresAt = timePtr(t)
	}

	// 待办类型软删除：12h 后 TTL 自动删除
	if status == StatusDeleted && assignmentType == "todo" {
		t := now.Add(12 * time.Hour)
		todoExpiresAt = timePtr(t)
	}

	forever := 1
	if status == StatusPermanentDeleted {
		forever = 0
	}

	doc := bson.M{
		"user_id":            userID,
		"assignment_id":      assignmentID,
		"assignment_title":   title,
		"assignment_subject": subject,
		"assignment_type":    assignmentType,
		"status":             status,
		"status_time":        now,
		"forever":            forever,
		"updated_at":         now,
	}

	if expiresAt != nil {
		doc["expires_at"] = *expiresAt
	}
	if todoExpiresAt != nil {
		doc["todo_expires_at"] = *todoExpiresAt
	}

	// Clear expiry fields left by a previous completed status; otherwise MongoDB
	// can expire a deleted record and make the assignment reappear.
	unset := bson.M{}
	if expiresAt == nil {
		unset["expires_at"] = ""
	}
	if todoExpiresAt == nil {
		unset["todo_expires_at"] = ""
	}
	update := bson.M{"$set": doc}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	_, err := r.collection.UpdateOne(ctx,
		bson.M{"user_id": userID, "assignment_id": assignmentID},
		update,
		options.Update().SetUpsert(true),
	)
	return err
}

// MarkCompleted 标记已完成，完成状态由作业生命周期决定
// 对应 Python: mark_completed
func (r *MongoAssignmentStatusRepository) MarkCompleted(userID, assignmentID, title, subject string) error {
	return r.UpdateStatus(userID, assignmentID, title, subject, "", StatusCompleted, nil)
}

// MarkDeleted 软删除，待办类型 12h 后自动删除
// 对应 Python: mark_deleted
func (r *MongoAssignmentStatusRepository) MarkDeleted(userID, assignmentID, title, subject, assignmentType string) error {
	return r.UpdateStatus(userID, assignmentID, title, subject, assignmentType, StatusDeleted, nil)
}

// RestoreAssignment 恢复到活跃状态（删除记录）
// 对应 Python: restore_assignment
func (r *MongoAssignmentStatusRepository) RestoreAssignment(userID, assignmentID string) error {
	return r.UpdateStatus(userID, assignmentID, "", "", "", StatusActive, nil)
}

// PermanentDelete 永久删除（forever=0）
// 对应 Python: permanent_delete_assignment
func (r *MongoAssignmentStatusRepository) PermanentDelete(userID, assignmentID, title, subject string) error {
	return r.UpdateStatus(userID, assignmentID, title, subject, "", StatusPermanentDeleted, nil)
}

// ──────────────────────────────────────────────
//  查询操作
// ──────────────────────────────────────────────

// GetStatus 获取单条作业状态，不存在返回 "active"
// 对应 Python: get_assignment_status
func (r *MongoAssignmentStatusRepository) GetStatus(userID, assignmentID string) (Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var doc AssignmentStatus
	err := r.collection.FindOne(ctx, bson.M{
		"user_id":       userID,
		"assignment_id": assignmentID,
	}).Decode(&doc)

	if err == mongo.ErrNoDocuments {
		return StatusActive, nil
	}
	if err != nil {
		return StatusActive, err
	}
	return doc.Status, nil
}

// GetUserStatuses 获取用户状态列表
// 对应 Python: get_user_assignment_status
func (r *MongoAssignmentStatusRepository) GetUserStatuses(
	userID string, status Status, includePermanentDeleted bool,
) ([]*AssignmentStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userID}
	if status != "" {
		filter["status"] = status
	}
	if !includePermanentDeleted {
		filter["forever"] = bson.M{"$ne": 0}
	}

	opts := options.Find().SetSort(bson.D{{Key: "status_time", Value: -1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []*AssignmentStatus
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// GetCompletedIDs 获取用户已完成的 assignmentID 列表（排除永久删除）
// 对应 Python: get_completed_assignment_ids
func (r *MongoAssignmentStatusRepository) GetCompletedIDs(userID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx,
		bson.M{
			"user_id": userID,
			"status":  StatusCompleted,
			"forever": bson.M{"$ne": 0},
		},
		options.Find().SetProjection(bson.M{"assignment_id": 1}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rows []struct {
		AssignmentID string `bson:"assignment_id"`
	}
	if err = cursor.All(ctx, &rows); err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.AssignmentID)
	}
	return ids, nil
}

// GetDeletedIDs 获取用户已删除的 assignmentID 列表（排除永久删除）
// 对应 Python: get_deleted_assignment_ids
func (r *MongoAssignmentStatusRepository) GetDeletedIDs(userID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx,
		bson.M{
			"user_id": userID,
			"status":  bson.M{"$in": bson.A{StatusDeleted, StatusPermanentDeleted}},
		},
		options.Find().SetProjection(bson.M{"assignment_id": 1}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rows []struct {
		AssignmentID string `bson:"assignment_id"`
	}
	if err = cursor.All(ctx, &rows); err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.AssignmentID)
	}
	return ids, nil
}

// IsCompleted 检查是否已完成
// 对应 Python: is_completed
func (r *MongoAssignmentStatusRepository) IsCompleted(userID, assignmentID string) (bool, error) {
	s, err := r.GetStatus(userID, assignmentID)
	return s == StatusCompleted, err
}

// IsDeleted 检查是否已删除
// 对应 Python: is_deleted
func (r *MongoAssignmentStatusRepository) IsDeleted(userID, assignmentID string) (bool, error) {
	s, err := r.GetStatus(userID, assignmentID)
	return s == StatusDeleted, err
}

// ──────────────────────────────────────────────
//  批量清理
// ──────────────────────────────────────────────

// ClearUserStatus 清空用户状态记录（设 forever=0 而非物理删除）
// 对应 Python: clear_user_status
func (r *MongoAssignmentStatusRepository) ClearUserStatus(userID string, status Status) (int64, error) {
	defer usercache.Invalidate(userID)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"user_id": userID,
		"forever": bson.M{"$ne": 0},
	}
	if status != "" {
		filter["status"] = status
	}

	result, err := r.collection.UpdateMany(ctx, filter, bson.M{
		"$set": bson.M{
			"forever":    0,
			"status":     StatusPermanentDeleted,
			"updated_at": time.Now(),
		},
	})
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

// ClearUserStatusByIDs 根据 assignmentID 列表批量物理删除状态记录
// 对应 Python: clear_user_status_by_ids
func (r *MongoAssignmentStatusRepository) ClearUserStatusByIDs(userID string, assignmentIDs []string) (int64, error) {
	defer usercache.Invalidate(userID)
	if len(assignmentIDs) == 0 {
		return 0, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.collection.DeleteMany(ctx, bson.M{
		"user_id":       userID,
		"assignment_id": bson.M{"$in": assignmentIDs},
	})
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

// CleanupExpired 手动清理过期记录（TTL 索引备用）
// 对应 Python: cleanup_expired
func (r *MongoAssignmentStatusRepository) CleanupExpired() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.collection.DeleteMany(ctx, bson.M{
		"expires_at": bson.M{"$lt": time.Now()},
		"status":     bson.M{"$ne": StatusCompleted},
	})
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

// ──────────────────────────────────────────────
//  统计
// ──────────────────────────────────────────────

// GetStats 获取用户状态统计
// 对应 Python: get_status_stats
func (r *MongoAssignmentStatusRepository) GetStats(userID string) (*StatusStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := bson.A{
		bson.M{"$match": bson.M{"user_id": userID}},
		bson.M{"$group": bson.M{
			"_id":   "$status",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	stats := &StatusStats{}
	var rows []struct {
		ID    string `bson:"_id"`
		Count int    `bson:"count"`
	}
	if err = cursor.All(ctx, &rows); err != nil {
		return nil, err
	}

	for _, row := range rows {
		switch row.ID {
		case StatusCompleted:
			stats.Completed = row.Count
		case StatusDeleted:
			stats.Deleted = row.Count
		}
		stats.Total += row.Count
	}
	return stats, nil
}

// GenerateID 为新文档生成 Snowflake ID（供 upsert 插入时使用）
func generateID() string {
	return config.Snowflake.GenerateID()
}
