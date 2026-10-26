package CourseData

import (
	"context"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/internal/deadline"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func normalizeSchedule(task *CourseData) {
	task.DeadlineNormalized = true
	task.DeadlineAt, task.AutoReminderAt = nil, nil
	task.AutoReminderPending = false
	if at, err := deadline.Parse(task.Deadline); err == nil {
		auto := at.Add(-24 * time.Hour)
		task.DeadlineAt, task.AutoReminderAt = &at, &auto
		task.AutoReminderPending = true
	}
}

func sameContent(a, b *CourseData) bool {
	return a.Subject == b.Subject && a.Title == b.Title && a.Deadline == b.Deadline &&
		a.Details == b.Details && a.Url == b.Url && a.Type == b.Type && a.DeadlineNormalized
}

// BackfillDeadlines visits legacy records once, without loading all tasks into memory.
func (r *MongoCourseDataRepository) BackfillDeadlines() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	marker := r.collection.Database().Collection("_migrations")
	if err := marker.FindOne(ctx, bson.M{"_id": "course_deadline_v1"}).Err(); err == nil {
		return nil
	} else if err != mongo.ErrNoDocuments {
		return err
	}
	cursor, err := r.collection.Find(ctx, bson.M{"deadline_normalized": bson.M{"$ne": true}},
		options.Find().SetProjection(bson.M{"deadline": 1}).SetBatchSize(128))
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	batch := make([]mongo.WriteModel, 0, 128)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		_, err := r.collection.BulkWrite(ctx, batch, options.BulkWrite().SetOrdered(false))
		batch = batch[:0]
		return err
	}
	for cursor.Next(ctx) {
		var doc struct {
			ID       interface{} `bson:"_id"`
			Deadline string      `bson:"deadline"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return err
		}
		task := &CourseData{Deadline: doc.Deadline}
		normalizeSchedule(task)
		batch = append(batch, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": doc.ID, "deadline": doc.Deadline, "deadline_normalized": bson.M{"$ne": true}}).
			SetUpdate(bson.M{"$set": bson.M{"deadline_normalized": true, "deadline_at": task.DeadlineAt,
				"auto_reminder_at": task.AutoReminderAt, "auto_reminder_pending": task.AutoReminderPending}}))
		if len(batch) == 128 {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := cursor.Err(); err != nil {
		return err
	}
	if err := flush(); err != nil {
		return err
	}
	_, err = marker.UpdateOne(ctx, bson.M{"_id": "course_deadline_v1"}, bson.M{"$set": bson.M{"completed_at": time.Now()}}, options.Update().SetUpsert(true))
	return err
}

// GetUserSummaries excludes potentially large descriptions from cached list responses.
func (r *MongoCourseDataRepository) GetUserSummaries(userID string) ([]*CourseData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID}, options.Find().
		SetProjection(bson.M{"details": 0}).SetSort(bson.D{{Key: "updated_at", Value: -1}, {Key: "task_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var tasks []*CourseData
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *MongoCourseDataRepository) GetByTaskID(userID, taskID string) (*CourseData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var task CourseData
	err := r.collection.FindOne(ctx, bson.M{"user_id": userID, "task_id": taskID}).Decode(&task)
	return &task, err
}
