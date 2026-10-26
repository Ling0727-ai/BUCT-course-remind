package CourseData

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/config"
	"github.com/Ling0727-ai/go-buct-course-backend/internal/scheduling"
	"github.com/Ling0727-ai/go-buct-course-backend/internal/usercache"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewMongoCourseDataRepository 创建并初始化仓库，同时建立索引
func NewMongoCourseDataRepository(col *mongo.Collection) *MongoCourseDataRepository {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// 对应 Python: create_index([("user_id", 1), ("updated_at", -1)])
	col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "updated_at", Value: -1},
		},
	})
	// 对应 Python: create_index([("user_id", 1), ("task_id", 1)], unique=True)
	col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "task_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if _, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "auto_reminder_pending", Value: 1}, {Key: "auto_reminder_at", Value: 1}, {Key: "deadline_at", Value: 1}},
	}); err != nil {
		log.Printf("[course-data] 创建提醒到期索引失败: %v", err)
	}
	if _, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "auto_reminder_pending", Value: 1}, {Key: "deadline_at", Value: 1}}}); err != nil {
		log.Printf("[course-data] 创建截止时间索引失败: %v", err)
	}

	return &MongoCourseDataRepository{collection: col}
}

// generateTaskID 生成确定性任务ID，对应 Python 中的 MD5 哈希逻辑
func generateTaskID(taskType, subject, title, deadline, details string) string {
	// 校验并修正 taskType
	switch taskType {
	case "homework", "test", "todo":
		// 合法，保持不变
	default:
		if strings.TrimSpace(details) != "" {
			taskType = "homework"
		} else {
			taskType = "test"
		}
	}
	// content_str = f"{subject}|{title}|{deadline}|{details}"
	contentStr := fmt.Sprintf("%s|%s|%s|%s", subject, title, deadline, details)
	hash := md5.Sum([]byte(contentStr))
	// 取前 16 个十六进制字符（即前 8 字节）
	contentHash := fmt.Sprintf("%x", hash)[:16]
	return fmt.Sprintf("%s_%s", taskType, contentHash)
}

// CreateCourseData 插入单条课程数据
func (r *MongoCourseDataRepository) CreateCourseData(courseData *CourseData) error {
	defer usercache.Invalidate(courseData.UserID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	courseData.ID = config.Snowflake.GenerateID()
	courseData.CreatedAt = now
	courseData.UpdatedAt = now
	normalizeSchedule(courseData)

	_, err := r.collection.InsertOne(ctx, courseData)
	return err
}

// GetCourseDataByID 通过 Snowflake ID 查询单条数据
func (r *MongoCourseDataRepository) GetCourseDataByID(id string) (*CourseData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result CourseData
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCourseDataByUserID 获取某用户的全部课程数据，按 updated_at 降序
func (r *MongoCourseDataRepository) GetCourseDataByUserID(userID string) ([]*CourseData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []*CourseData
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// UpdateCourseData 更新一条课程数据（以 ID 为条件）
func (r *MongoCourseDataRepository) UpdateCourseData(courseData *CourseData) error {
	defer usercache.Invalidate(courseData.UserID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	courseData.UpdatedAt = time.Now()
	normalizeSchedule(courseData)
	_, err := r.collection.ReplaceOne(ctx,
		bson.M{"_id": courseData.ID},
		courseData,
	)
	return err
}

// DeleteCourseData 删除单条课程数据（by Snowflake ID）
func (r *MongoCourseDataRepository) DeleteCourseData(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// GetLastUpdateTimeByUserID 获取用户数据的最后更新时间
// 对应 Python: get_last_update_time
func (r *MongoCourseDataRepository) GetLastUpdateTimeByUserID(userID string) (time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var meta struct {
		Refreshed time.Time `bson:"course_refreshed_at"`
	}
	key := interface{}(userID)
	if id, err := primitive.ObjectIDFromHex(userID); err == nil {
		key = id
	}
	err := r.collection.Database().Collection("users").FindOne(ctx, bson.M{"_id": key},
		options.FindOne().SetProjection(bson.M{"course_refreshed_at": 1})).Decode(&meta)
	if err == nil && !meta.Refreshed.IsZero() {
		return meta.Refreshed, nil
	}
	if err != nil && err != mongo.ErrNoDocuments {
		return time.Time{}, err
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "updated_at", Value: -1}})
	var result CourseData
	err = r.collection.FindOne(ctx, bson.M{"user_id": userID}, opts).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return result.UpdatedAt, nil
}

// ClearCourseDataByUserID 清空用户的全部课程数据
// 对应 Python: clear_user_data
func (r *MongoCourseDataRepository) ClearCourseDataByUserID(userID string) error {
	defer usercache.Invalidate(userID)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.collection.DeleteMany(ctx, bson.M{"user_id": userID})
	return err
}

// SaveUserCourseData writes only changed tasks, then prunes tasks removed upstream.
// The count remains the number of unique tasks to preserve the API contract.
func (r *MongoCourseDataRepository) SaveUserCourseData(userID string, tasks []TaskInput) (int, error) {
	defer usercache.Invalidate(userID)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	now := time.Now()
	previous, err := r.GetCourseDataByUserID(userID)
	if err != nil {
		return 0, err
	}
	old := make(map[string]*CourseData, len(previous))
	for _, task := range previous {
		old[task.TaskID] = task
	}
	active := make(map[string]bool, len(tasks))
	writes := make([]mongo.WriteModel, 0)
	for _, task := range tasks {
		taskID := generateTaskID(task.Type, task.Subject, task.Title, task.Deadline, task.Details)
		if active[taskID] {
			continue
		}
		active[taskID] = true
		doc := &CourseData{UserID: userID, TaskID: taskID, Subject: task.Subject, Title: task.Title,
			Deadline: task.Deadline, Details: task.Details, Url: task.Url, Type: strings.Split(taskID, "_")[0]}
		if prior := old[taskID]; prior != nil && sameContent(prior, doc) {
			continue
		}
		normalizeSchedule(doc)
		set := bson.M{"subject": doc.Subject, "title": doc.Title, "deadline": doc.Deadline,
			"details": doc.Details, "url": doc.Url, "type": doc.Type, "updated_at": now,
			"deadline_at": doc.DeadlineAt, "deadline_normalized": true}
		// Preserve reminder state when only an unrelated field (e.g. URL) changes.
		if prior := old[taskID]; prior == nil || !prior.DeadlineNormalized || prior.Deadline != doc.Deadline {
			set["auto_reminder_at"], set["auto_reminder_pending"] = doc.AutoReminderAt, doc.AutoReminderPending
		}
		writes = append(writes, mongo.NewUpdateOneModel().SetFilter(bson.M{"user_id": userID, "task_id": taskID}).
			SetUpdate(bson.M{"$set": set, "$setOnInsert": bson.M{"_id": config.Snowflake.GenerateID(), "user_id": userID, "task_id": taskID, "created_at": now}}).SetUpsert(true))
	}
	if len(writes) > 0 {
		if _, err := r.collection.BulkWrite(ctx, writes, options.BulkWrite().SetOrdered(false)); err != nil {
			return 0, err
		}
	}
	removed := make([]string, 0)
	for id := range old {
		if !active[id] {
			removed = append(removed, id)
		}
	}
	if len(removed) > 0 {
		if _, err := r.collection.Database().Collection("scheduled_reminders").UpdateMany(ctx,
			bson.M{"user_id": userID, "type": "assignment", "target_id": bson.M{"$in": removed}, "status": "scheduled"},
			bson.M{"$set": bson.M{"status": "cancelled", "cancel_reason": "task removed upstream", "updated_at": now}}); err != nil {
			return 0, err
		}
		if _, err := r.collection.DeleteMany(ctx, bson.M{"user_id": userID, "task_id": bson.M{"$in": removed}}); err != nil {
			return 0, err
		}
	}
	// Record freshness separately so unchanged/empty results still have a timestamp.
	userKey := interface{}(userID)
	if objectID, err := primitive.ObjectIDFromHex(userID); err == nil {
		userKey = objectID
	}
	users := r.collection.Database().Collection("users")
	if _, err := users.UpdateOne(ctx, bson.M{"_id": userKey}, bson.M{"$set": bson.M{
		"next_refresh_at":     time.Now().Add(AutoRefreshInterval),
		"course_refreshed_at": time.Now(),
	}, "$unset": bson.M{"refresh_lease_until": ""}}); err != nil {
		return 0, err
	}
	scheduling.NotifyReminders()
	return len(active), nil
}
