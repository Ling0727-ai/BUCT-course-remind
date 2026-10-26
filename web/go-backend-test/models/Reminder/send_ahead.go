package Reminder

import (
	"context"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/internal/scheduling"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// BackfillSendAhead shifts existing scheduled auto reminders once, without touching sent mail.
func (r *MongoReminderRepository) BackfillSendAhead() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	markers := r.collection.Database().Collection("_migrations")
	const migrationID = "auto_reminder_send_ahead_5s_v1"
	if err := markers.FindOne(ctx, bson.M{"_id": migrationID}).Err(); err == nil {
		return nil
	} else if err != mongo.ErrNoDocuments {
		return err
	}
	filter := bson.M{"auto_created": true, "status": "scheduled", "send_ahead_seconds": bson.M{"$exists": false}, "scheduled_time": bson.M{"$type": "date"}}
	cursor, err := r.collection.Find(ctx, filter, options.Find().SetProjection(bson.M{"scheduled_time": 1}).SetBatchSize(128))
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
		var reminder struct {
			ID interface{} `bson:"_id"`
			At time.Time   `bson:"scheduled_time"`
		}
		if err := cursor.Decode(&reminder); err != nil {
			return err
		}
		batch = append(batch, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": reminder.ID, "auto_created": true, "status": "scheduled",
			"scheduled_time": reminder.At, "send_ahead_seconds": bson.M{"$exists": false}}).
			SetUpdate(bson.M{"$set": bson.M{"scheduled_time": reminder.At.Add(-scheduling.AutoReminderSendAhead),
				"send_ahead_seconds": int(scheduling.AutoReminderSendAhead / time.Second), "updated_at": time.Now()}}))
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
	_, err = markers.UpdateOne(ctx, bson.M{"_id": migrationID}, bson.M{"$set": bson.M{"completed_at": time.Now()}}, options.Update().SetUpsert(true))
	if err == nil {
		scheduling.NotifyReminders()
	}
	return err
}
