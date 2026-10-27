package CourseData

import (
	"context"
	"crypto/md5"
	"errors"
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

var ErrEmptyRefreshUnconfirmed = errors.New("抓取结果为空，正在复核，原有作业已保留")

const (
	emptyRefreshConfirmAfter  = 30 * time.Second
	emptyRefreshConfirmWithin = 30 * time.Minute
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

// generateTaskID keeps the identity independent of a deadline that teachers may edit.
func generateTaskID(taskType, subject, title, details string) string {
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
	contentStr := fmt.Sprintf("%s|%s|%s", subject, title, details)
	hash := md5.Sum([]byte(contentStr))
	// 取前 16 个十六进制字符（即前 8 字节）
	contentHash := fmt.Sprintf("%x", hash)[:16]
	return fmt.Sprintf("%s_%s", taskType, contentHash)
}

func legacyTaskID(task TaskInput) string {
	stableID := generateTaskID(task.Type, task.Subject, task.Title, task.Details)
	content := fmt.Sprintf("%s|%s|%s|%s", task.Subject, task.Title, task.Deadline, task.Details)
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%s_%x", strings.Split(stableID, "_")[0], hash[:8])
}

// resolveTaskID retains IDs created by the old deadline-based hash, so existing
// completion states and reminders keep pointing at the same assignment.
func resolveTaskID(task TaskInput, previous map[string][]*CourseData, active map[string]bool) (string, *CourseData) {
	stableID := generateTaskID(task.Type, task.Subject, task.Title, task.Details)
	candidates := previous[stableID]
	for _, prior := range candidates {
		if !active[prior.TaskID] && prior.Deadline == task.Deadline {
			return prior.TaskID, prior
		}
	}
	var only *CourseData
	for _, prior := range candidates {
		if active[prior.TaskID] {
			continue
		}
		if only != nil {
			only = nil
			break
		}
		only = prior
	}
	if only != nil {
		return only.TaskID, only
	}
	for _, prior := range candidates {
		if prior.TaskID == stableID && !active[stableID] {
			return stableID, prior
		}
	}
	if !active[stableID] {
		return stableID, nil
	}
	// Identical content can occur twice in one course. The upstream scraper has
	// no stable assignment ID yet, so give the next copy a distinct suffix.
	for copy := 2; ; copy++ {
		id := fmt.Sprintf("%s_%d", stableID, copy)
		if !active[id] {
			return id, nil
		}
	}
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
	if len(previous) > 0 && len(tasks) == 0 {
		if err := r.confirmEmptyRefresh(ctx, userID, now); err != nil {
			return 0, err
		}
	}
	old := make(map[string]*CourseData, len(previous))
	oldByIdentity := make(map[string][]*CourseData, len(previous))
	for _, task := range previous {
		old[task.TaskID] = task
		identity := generateTaskID(task.Type, task.Subject, task.Title, task.Details)
		oldByIdentity[identity] = append(oldByIdentity[identity], task)
	}
	// Reserve exact matches first. If two tasks share the same content, a
	// changed deadline must not consume the other task's unchanged ID.
	exactMatches := make([]*CourseData, len(tasks))
	occupied := make(map[string]bool, len(tasks))
	seenExact := make(map[string]bool, len(tasks))
	for i, task := range tasks {
		key := legacyTaskID(task)
		if seenExact[key] {
			continue
		}
		seenExact[key] = true
		identity := generateTaskID(task.Type, task.Subject, task.Title, task.Details)
		for _, prior := range oldByIdentity[identity] {
			if !occupied[prior.TaskID] && prior.Deadline == task.Deadline {
				exactMatches[i] = prior
				occupied[prior.TaskID] = true
				break
			}
		}
	}
	active := make(map[string]bool, len(tasks))
	seenInput := make(map[string]bool, len(tasks))
	writes := make([]mongo.WriteModel, 0)
	changedDeadlines := make(map[string]*time.Time)
	for i, task := range tasks {
		inputKey := legacyTaskID(task)
		if seenInput[inputKey] {
			continue
		}
		seenInput[inputKey] = true
		prior := exactMatches[i]
		taskID := ""
		if prior != nil {
			taskID = prior.TaskID
		} else {
			taskID, prior = resolveTaskID(task, oldByIdentity, occupied)
		}
		if active[taskID] {
			continue
		}
		active[taskID] = true
		occupied[taskID] = true
		doc := &CourseData{UserID: userID, TaskID: taskID, Subject: task.Subject, Title: task.Title,
			Deadline: task.Deadline, Details: task.Details, Url: task.Url, Type: strings.Split(taskID, "_")[0]}
		if prior != nil && sameContent(prior, doc) {
			continue
		}
		normalizeSchedule(doc)
		if prior != nil && prior.Deadline != doc.Deadline {
			changedDeadlines[taskID] = doc.DeadlineAt
		}
		set := bson.M{"subject": doc.Subject, "title": doc.Title, "deadline": doc.Deadline,
			"details": doc.Details, "url": doc.Url, "type": doc.Type, "updated_at": now,
			"deadline_at": doc.DeadlineAt, "deadline_normalized": true, "auto_reminder_version": scheduling.AutoReminderVersion}
		// Preserve reminder state when only an unrelated field (e.g. URL) changes.
		if prior := old[taskID]; prior == nil || !prior.DeadlineNormalized || prior.Deadline != doc.Deadline || prior.AutoReminderVersion < scheduling.AutoReminderVersion {
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
	// Previously scheduled or sent automatic stages refer to the old deadline.
	// Re-arm them so the reminder scheduler can plan against the new one.
	for taskID, deadlineAt := range changedDeadlines {
		filter := bson.M{"user_id": userID, "target_id": taskID, "auto_created": true,
			"status": bson.M{"$in": bson.A{"scheduled", "sent", "failed", "cancelled"}}}
		if deadlineAt != nil {
			filter["deadline_at"] = bson.M{"$ne": *deadlineAt}
		}
		if _, err := r.collection.Database().Collection("scheduled_reminders").UpdateMany(ctx,
			filter,
			bson.M{"$set": bson.M{"status": "cancelled", "cancel_reason": "task deadline changed", "updated_at": now}}); err != nil {
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
		if _, err := r.collection.Database().Collection("assignment_status").DeleteMany(ctx,
			bson.M{"user_id": userID, "status": "completed", "assignment_id": bson.M{"$in": removed}}); err != nil {
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
	}, "$unset": bson.M{"refresh_lease_until": "", "empty_refresh_seen_at": ""}}); err != nil {
		return 0, err
	}
	scheduling.NotifyReminders()
	return len(active), nil
}

// A single empty scrape may be an upstream glitch. Keep existing tasks until
// another independent refresh confirms the empty result shortly afterward.
func (r *MongoCourseDataRepository) confirmEmptyRefresh(ctx context.Context, userID string, now time.Time) error {
	userKey := interface{}(userID)
	if objectID, err := primitive.ObjectIDFromHex(userID); err == nil {
		userKey = objectID
	}
	users := r.collection.Database().Collection("users")
	var marker struct {
		SeenAt *time.Time `bson:"empty_refresh_seen_at"`
	}
	if err := users.FindOne(ctx, bson.M{"_id": userKey}, options.FindOne().SetProjection(bson.M{"empty_refresh_seen_at": 1})).Decode(&marker); err != nil {
		return err
	}
	if marker.SeenAt != nil {
		elapsed := now.Sub(*marker.SeenAt)
		if elapsed >= emptyRefreshConfirmAfter && elapsed <= emptyRefreshConfirmWithin {
			return nil
		}
		if elapsed >= 0 && elapsed < emptyRefreshConfirmAfter {
			return ErrEmptyRefreshUnconfirmed
		}
	}
	if _, err := users.UpdateOne(ctx, bson.M{"_id": userKey}, bson.M{"$set": bson.M{"empty_refresh_seen_at": now}}); err != nil {
		return err
	}
	return ErrEmptyRefreshUnconfirmed
}
