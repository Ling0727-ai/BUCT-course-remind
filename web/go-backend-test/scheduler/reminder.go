package scheduler

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/config"
	"github.com/Ling0727-ai/go-buct-course-backend/internal/deadline"
	"github.com/Ling0727-ai/go-buct-course-backend/internal/scheduling"
	"github.com/Ling0727-ai/go-buct-course-backend/models"
	"github.com/Ling0727-ai/go-buct-course-backend/models/AssignmentStatus"
	"github.com/Ling0727-ai/go-buct-course-backend/models/CourseData"
	"github.com/Ling0727-ai/go-buct-course-backend/models/Reminder"
	"github.com/Ling0727-ai/go-buct-course-backend/models/Todo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const reminderProcessingLease = 2 * time.Minute

// Delivery concurrency is bounded independently of scraping.
func processDueReminders() error {
	client, err := models.ConnectToDB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	col := client.Database("buct-course").Collection(RemindersCollection)
	if err := recoverStaleReminders(ctx, col, time.Now()); err != nil {
		return err
	}
	cursor, err := col.Find(ctx, bson.M{"status": "scheduled", "scheduled_time": bson.M{"$lte": time.Now()}},
		options.Find().SetSort(bson.D{{Key: "scheduled_time", Value: 1}}).SetLimit(50))
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	var reminders []ScheduledReminder
	if err := cursor.All(ctx, &reminders); err != nil {
		return err
	}
	jobs := make(chan ScheduledReminder)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for reminder := range jobs {
				if err := deliverReminder(col, reminder); err != nil {
					log.Printf("[reminder] 发送失败: %v", err)
				}
			}
		}()
	}
	for _, reminder := range reminders {
		jobs <- reminder
	}
	close(jobs)
	wg.Wait()
	return nil
}

func recoverStaleReminders(ctx context.Context, col *mongo.Collection, now time.Time) error {
	result, err := col.UpdateMany(ctx, bson.M{"status": "processing", "$or": bson.A{
		bson.M{"processing_until": bson.M{"$lte": now}},
		bson.M{"processing_until": bson.M{"$exists": false}, "updated_at": bson.M{"$lte": now.Add(-reminderProcessingLease)}},
	}}, bson.M{"$set": bson.M{"status": "scheduled", "updated_at": now},
		"$unset": bson.M{"processing_until": "", "claim_token": ""}})
	if err == nil && result.ModifiedCount > 0 {
		log.Printf("[reminder] 恢复 %d 条超时的发送任务", result.ModifiedCount)
	}
	return err
}

func deliverReminder(col *mongo.Collection, reminder ScheduledReminder) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	now := time.Now()
	claimToken := primitive.NewObjectID().Hex()
	claim, err := col.UpdateOne(ctx, bson.M{"_id": reminder.ID, "status": "scheduled", "scheduled_time": bson.M{"$lte": now}},
		bson.M{"$set": bson.M{"status": "processing", "updated_at": now,
			"processing_until": now.Add(reminderProcessingLease), "claim_token": claimToken}})
	if err != nil {
		return err
	}
	if claim.MatchedCount == 0 {
		return nil
	}
	status, reason := "sent", ""
	if reminder.Email == "" || reminder.Message == "" {
		status, reason = "failed", "missing email or message"
	} else {
		message, why, checkErr := reminderDeliveryContent(reminder, time.Now())
		if checkErr != nil {
			status, reason = "scheduled", checkErr.Error()
		} else if why != "" {
			status, reason = "cancelled", why
		} else if config.Mail == nil {
			status, reason = "failed", "mail is not configured"
		} else if err := config.Mail.SendMail(reminder.Email, "北化课程提醒", message); err != nil {
			status, reason = "failed", err.Error()
		}
	}
	set := bson.M{"status": status, "updated_at": time.Now()}
	if status == "sent" {
		set["sent_at"] = time.Now()
	}
	if reason != "" {
		set["error"] = reason
	}
	if status == "cancelled" {
		set["cancel_reason"] = reason
	}
	if status == "scheduled" {
		set["scheduled_time"] = time.Now().Add(time.Minute)
	}
	// SMTP may have outlived its batch context; persist the outcome with a fresh one.
	writeCtx, writeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer writeCancel()
	result, err := col.UpdateOne(writeCtx, bson.M{"_id": reminder.ID, "status": "processing", "claim_token": claimToken},
		bson.M{"$set": set, "$unset": bson.M{"processing_until": "", "claim_token": ""}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("提醒 %v 的处理租约已失效", reminder.ID)
	}
	if reminder.AutoCreated && status == "failed" {
		_, err = col.Database().Collection("course_data").UpdateOne(writeCtx,
			bson.M{"user_id": reminder.UserID, "task_id": reminder.TargetID, "deadline_at": bson.M{"$gt": time.Now()}},
			bson.M{"$set": bson.M{"auto_reminder_pending": true, "auto_reminder_at": time.Now().Add(time.Minute)}})
	}
	return err
}

func reminderDeliveryContent(r ScheduledReminder, now time.Time) (string, string, error) {
	if r.AutoCreated && r.DeadlineAt != nil && scheduling.AutoReminderStageExpired(r.OffsetHours, *r.DeadlineAt, now) {
		return "", "auto reminder stage superseded or deadline passed", nil
	}
	if skip, why, err := reminderSkipState(r); err != nil || skip {
		return "", why, err
	}
	if !r.AutoCreated || r.Type != "assignment" {
		return r.Message, "", nil
	}
	task, err := CourseData.Repository.GetByTaskID(r.UserID, r.TargetID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", "task removed upstream", nil
	}
	if err != nil {
		return "", "", err
	}
	at := task.DeadlineAt
	if at == nil {
		parsed, err := deadline.Parse(task.Deadline)
		if err != nil {
			return "", "task deadline unavailable", nil
		}
		at = &parsed
	}
	if r.DeadlineAt != nil && !r.DeadlineAt.Equal(*at) {
		return "", "task deadline changed", nil
	}
	if scheduling.AutoReminderStageExpired(r.OffsetHours, *at, now) {
		return "", "auto reminder stage superseded or deadline passed", nil
	}
	return buildAutoReminderMessage(task, *at, now, r.OffsetHours), "", nil
}

func reminderSkipState(r ScheduledReminder) (bool, string, error) {
	if r.UserID == "" || r.TargetID == "" {
		return false, "", nil
	}
	if r.Type == "assignment" {
		status, err := AssignmentStatus.Repository.GetStatus(r.UserID, r.TargetID)
		if err != nil {
			return false, "", err
		}
		switch status {
		case AssignmentStatus.StatusCompleted:
			return true, "assignment completed", nil
		case AssignmentStatus.StatusDeleted, AssignmentStatus.StatusPermanentDeleted:
			return true, "assignment deleted", nil
		}
	} else if r.Type == "todo" {
		todo, err := Todo.Repository.GetTodoByID(r.TargetID)
		if errors.Is(err, mongo.ErrNoDocuments) || (err == nil && todo == nil) {
			return true, "todo missing", nil
		}
		if err != nil {
			return false, "", err
		}
		if todo.UserID != r.UserID || todo.Completed || todo.IsDeleted || todo.Forever == 0 {
			return true, "todo completed or deleted", nil
		}
	}
	return false, "", nil
}

func autoReminderID(userID, taskID string, hours int) string {
	key := userID + "\x00" + taskID
	// Keep the original 24-hour ID so already-created reminders remain idempotent.
	if hours != 0 && hours != 24 {
		key += fmt.Sprintf("\x00%dh", hours)
	}
	return fmt.Sprintf("auto_%x", sha256.Sum256([]byte(key)))
}

func autoReminderKey(userID, taskID string, hours int) string {
	if hours == 0 {
		hours = 24
	}
	return fmt.Sprintf("%s\x00%s\x00%d", userID, taskID, hours)
}

// Query a bounded set of due tasks, then fetch users, states and legacy reminders in batches.
var deadlineMigrationReady atomic.Bool

func checkAndCreateAutoReminders() error {
	client, err := models.ConnectToDB()
	if err != nil {
		return err
	}
	if !deadlineMigrationReady.Load() {
		if err := CourseData.Repository.BackfillDeadlines(); err != nil {
			return err
		}
		if err := Reminder.Repository.BackfillSendAhead(); err != nil {
			return err
		}
		deadlineMigrationReady.Store(true)
	}
	return createAutoReminders(client.Database("buct-course"), time.Now())
}

func createAutoReminders(db *mongo.Database, now time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	col := db.Collection("course_data")
	// Remove expired pending entries with a separate deadline index before the due range scan.
	if _, err := col.UpdateMany(ctx, bson.M{"auto_reminder_pending": true, "deadline_at": bson.M{"$lt": now}},
		bson.M{"$set": bson.M{"auto_reminder_pending": false}}); err != nil {
		return err
	}
	cursor, err := col.Find(ctx, bson.M{"auto_reminder_pending": true, "auto_reminder_at": bson.M{"$lte": now}, "deadline_at": bson.M{"$gte": now}},
		options.Find().SetProjection(bson.M{"details": 0}).SetSort(bson.D{{Key: "auto_reminder_at", Value: 1}}).SetLimit(200))
	if err != nil {
		return err
	}
	var tasks []*CourseData.CourseData
	err = cursor.All(ctx, &tasks)
	cursor.Close(ctx)
	if err != nil || len(tasks) == 0 {
		return err
	}
	userKeys := make([]interface{}, 0)
	stateFilters, reminderFilters := make(bson.A, 0, len(tasks)), make(bson.A, 0, len(tasks))
	seen := make(map[string]bool)
	for _, task := range tasks {
		stateFilters = append(stateFilters, bson.M{"user_id": task.UserID, "assignment_id": task.TaskID})
		reminderFilters = append(reminderFilters, bson.M{"user_id": task.UserID, "target_id": task.TaskID})
		if !seen[task.UserID] {
			seen[task.UserID] = true
			userKeys = append(userKeys, task.UserID)
			if id, err := primitive.ObjectIDFromHex(task.UserID); err == nil {
				userKeys = append(userKeys, id)
			}
		}
	}
	users, err := db.Collection("users").Find(ctx, bson.M{"_id": bson.M{"$in": userKeys}}, options.Find().SetProjection(bson.M{"email": 1}))
	if err != nil {
		return err
	}
	emails := make(map[string]string)
	for users.Next(ctx) {
		var user struct {
			ID    interface{} `bson:"_id"`
			Email string      `bson:"email"`
		}
		if err := users.Decode(&user); err != nil {
			users.Close(ctx)
			return err
		}
		emails[objectIDToString(user.ID)] = user.Email
	}
	if err := users.Err(); err != nil {
		users.Close(ctx)
		return err
	}
	users.Close(ctx)
	states, err := db.Collection("assignment_status").Find(ctx, bson.M{"$or": stateFilters},
		options.Find().SetProjection(bson.M{"user_id": 1, "assignment_id": 1, "status": 1, "expires_at": 1}))
	if err != nil {
		return err
	}
	blocked := make(map[string]bool)
	for states.Next(ctx) {
		var row struct {
			UserID  string     `bson:"user_id"`
			Target  string     `bson:"assignment_id"`
			Status  string     `bson:"status"`
			Expires *time.Time `bson:"expires_at"`
		}
		if err := states.Decode(&row); err != nil {
			states.Close(ctx)
			return err
		}
		if row.Status != AssignmentStatus.StatusCompleted && row.Expires != nil && !row.Expires.After(now) {
			continue
		}
		blocked[row.UserID+"\x00"+row.Target] = row.Status == "completed" || row.Status == "deleted" || row.Status == "permanent_deleted"
	}
	if err := states.Err(); err != nil {
		states.Close(ctx)
		return err
	}
	states.Close(ctx)
	remCol := db.Collection(RemindersCollection)
	existing, err := remCol.Find(ctx, bson.M{"$or": reminderFilters, "auto_created": true, "type": "assignment"},
		options.Find().SetProjection(bson.M{"user_id": 1, "target_id": 1, "status": 1, "offset_hours": 1}))
	if err != nil {
		return err
	}
	known := make(map[string]bool)
	retries := make(map[string]interface{})
	for existing.Next(ctx) {
		var r struct {
			ID     interface{} `bson:"_id"`
			UserID string      `bson:"user_id"`
			Target string      `bson:"target_id"`
			Status string      `bson:"status"`
			Hours  int         `bson:"offset_hours"`
		}
		if err := existing.Decode(&r); err != nil {
			existing.Close(ctx)
			return err
		}
		key := autoReminderKey(r.UserID, r.Target, r.Hours)
		if r.Status == "scheduled" || r.Status == "sent" || r.Status == "processing" {
			known[key] = true
		} else {
			retries[key] = r.ID
		}
	}
	if err := existing.Err(); err != nil {
		existing.Close(ctx)
		return err
	}
	existing.Close(ctx)
	reminders, updates := make([]mongo.WriteModel, 0), make([]mongo.WriteModel, 0, len(tasks))
	for _, task := range tasks {
		pending := false
		if task.DeadlineAt == nil {
			continue
		}
		for _, stage := range scheduling.AutoReminderStages(*task.DeadlineAt, now) {
			key := autoReminderKey(task.UserID, task.TaskID, stage.Hours)
			if known[key] {
				continue
			}
			if blocked[task.UserID+"\x00"+task.TaskID] {
				continue
			}
			if emails[task.UserID] == "" {
				pending = true
				continue
			}
			if retryID := retries[key]; retryID != nil {
				reminders = append(reminders, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": retryID, "status": bson.M{"$in": bson.A{"failed", "cancelled"}}}).
					SetUpdate(bson.M{"$set": bson.M{"status": "scheduled", "scheduled_time": stage.At, "updated_at": now, "email": emails[task.UserID],
						"offset_hours": stage.Hours, "deadline_at": *task.DeadlineAt, "send_ahead_seconds": int(scheduling.AutoReminderSendAhead / time.Second), "message": buildAutoReminderMessage(task, *task.DeadlineAt, stage.At, stage.Hours)}, "$unset": bson.M{"error": "", "cancel_reason": ""}}))
			} else {
				reminders = append(reminders, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": autoReminderID(task.UserID, task.TaskID, stage.Hours)}).
					SetUpdate(bson.M{"$setOnInsert": bson.M{"user_id": task.UserID, "target_id": task.TaskID, "type": "assignment", "auto_created": true,
						"offset_hours": stage.Hours, "deadline_at": *task.DeadlineAt, "send_ahead_seconds": int(scheduling.AutoReminderSendAhead / time.Second), "email": emails[task.UserID], "message": buildAutoReminderMessage(task, *task.DeadlineAt, stage.At, stage.Hours), "status": "scheduled",
						"scheduled_time": stage.At, "created_at": now, "updated_at": now}}).SetUpsert(true))
			}
		}
		set := bson.M{"auto_reminder_pending": pending, "auto_reminder_version": scheduling.AutoReminderVersion}
		if pending {
			set["auto_reminder_at"] = now.Add(time.Hour)
		}
		updates = append(updates, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": task.ID, "deadline": task.Deadline}).SetUpdate(bson.M{"$set": set}))
	}
	if len(reminders) > 0 {
		if _, err := remCol.BulkWrite(ctx, reminders, options.BulkWrite().SetOrdered(false)); err != nil {
			return err
		}
	}
	if len(updates) > 0 {
		if _, err := col.BulkWrite(ctx, updates, options.BulkWrite().SetOrdered(false)); err != nil {
			return err
		}
	}
	scheduling.NotifyReminders()
	return nil
}

func buildAutoReminderMessage(task *CourseData.CourseData, at, now time.Time, hours int) string {
	if hours == 0 {
		hours = 24
	}
	return fmt.Sprintf("提醒节点：截止前 %d 小时\n\n%s", hours, buildReminderMessage(task, at, now))
}

func buildReminderMessage(task *CourseData.CourseData, at, now time.Time) string {
	kind := "作业"
	if task.Type == "test" {
		kind = "测试"
	}
	seconds := max(0, int(math.Ceil(at.Sub(now).Seconds())))
	hours := seconds / 3600
	left := fmt.Sprintf("%d小时%02d分%02d秒", hours, seconds/60%60, seconds%60)
	if hours >= 24 {
		left = fmt.Sprintf("%d天%d小时%02d分%02d秒", hours/24, hours%24, seconds/60%60, seconds%60)
	}
	return fmt.Sprintf("⏰ 自动提醒：%s即将截止\n\n科目: %s\n标题: %s\n截止时间: %s\n剩余时间: %s\n\n请及时完成！", kind, task.Subject, task.Title, at.In(deadline.Campus).Format("2006年01月02日 15:04:05"), left)
}

func objectIDToString(value interface{}) string {
	switch id := value.(type) {
	case primitive.ObjectID:
		return id.Hex()
	case string:
		return id
	}
	return fmt.Sprint(value)
}
