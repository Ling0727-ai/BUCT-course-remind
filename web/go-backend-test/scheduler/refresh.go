package scheduler

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/internal/scheduling"
	"github.com/Ling0727-ai/go-buct-course-backend/models"
	"github.com/Ling0727-ai/go-buct-course-backend/services"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func resetTimer(timer *time.Timer, delay time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(delay)
}

func (s *CourseDataScheduler) run(stop <-chan struct{}) {
	refreshTimer := time.NewTimer(0)
	reminderTimer := time.NewTimer(0)
	autoTimer := time.NewTimer(0)
	defer refreshTimer.Stop()
	defer reminderTimer.Stop()
	defer autoTimer.Stop()
	reminderSem, autoSem := make(chan struct{}, 1), make(chan struct{}, 1)
	launch := func(sem chan struct{}, task func() error) {
		select {
		case sem <- struct{}{}:
			go func() {
				defer func() { <-sem; scheduling.NotifyReminders() }()
				if err := task(); err != nil {
					log.Printf("[scheduler] 后台任务失败: %v", err)
				}
			}()
		default:
		}
	}
	launch(autoSem, checkAndCreateAutoReminders)
	for {
		select {
		case <-stop:
			return
		case <-refreshTimer.C:
			s.runRefreshCycle()
			refreshTimer.Reset(s.nextRefreshDelay())
		case <-refreshScheduleChanged:
			s.runRefreshCycle()
			resetTimer(refreshTimer, s.nextRefreshDelay())
		case <-reminderTimer.C:
			launch(reminderSem, processDueReminders)
			reminderTimer.Reset(30 * time.Second)
		case <-autoTimer.C:
			launch(autoSem, checkAndCreateAutoReminders)
			autoTimer.Reset(30 * time.Second)
		case <-scheduling.RemindersChanged:
			resetTimer(reminderTimer, nextReminderDelay())
			resetTimer(autoTimer, nextAutoReminderDelay())
		}
	}
}

func eligibleUsers() bson.M {
	return bson.M{
		"student_id": bson.M{"$type": "string", "$gt": ""},
		"s_password": bson.M{"$type": "string", "$gt": ""},
	}
}

// Claim a bounded batch before enqueueing, so scraping never runs in the timer loop.
func (s *CourseDataScheduler) runRefreshCycle() {
	client, err := models.ConnectToDB()
	if err != nil {
		log.Printf("[scheduler] 数据库连接失败: %v", err)
		return
	}
	col := client.Database("buct-course").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for i := 0; i < 100; i++ {
		now := time.Now()
		filter := eligibleUsers()
		filter["$and"] = bson.A{
			bson.M{"$or": bson.A{bson.M{"next_refresh_at": bson.M{"$lte": now}}, bson.M{"next_refresh_at": bson.M{"$exists": false}}}},
			bson.M{"$or": bson.A{bson.M{"refresh_lease_until": bson.M{"$lte": now}}, bson.M{"refresh_lease_until": bson.M{"$exists": false}}}},
		}
		var user struct {
			ID interface{} `bson:"_id"`
		}
		lease := now.Add(20 * time.Minute)
		err := col.FindOneAndUpdate(ctx, filter,
			bson.M{"$set": bson.M{"refresh_lease_until": lease}},
			options.FindOneAndUpdate().SetSort(bson.D{{Key: "next_refresh_at", Value: 1}}).SetProjection(bson.M{"_id": 1}),
		).Decode(&user)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return
		}
		if err != nil {
			log.Printf("[scheduler] 领取刷新任务失败: %v", err)
			return
		}
		if err := services.TryRefreshUserDataAsync(objectIDToString(user.ID)); err != nil {
			// A CAS prevents overwriting a successful concurrent manual refresh.
			_, _ = col.UpdateOne(ctx, bson.M{"_id": user.ID, "refresh_lease_until": lease},
				bson.M{"$set": bson.M{"next_refresh_at": now.Add(30 * time.Second)}, "$unset": bson.M{"refresh_lease_until": ""}})
			return
		}
	}
}

func delayUntil(at time.Time, ceiling time.Duration) time.Duration {
	delay := time.Until(at)
	if delay < time.Second {
		return time.Second
	}
	if delay > ceiling {
		return ceiling
	}
	return delay
}

func (s *CourseDataScheduler) nextRefreshDelay() time.Duration {
	client, err := models.ConnectToDB()
	if err != nil {
		return time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var next struct {
		At time.Time `bson:"next_refresh_at"`
	}
	filter := eligibleUsers()
	filter["$or"] = bson.A{bson.M{"refresh_lease_until": bson.M{"$lte": time.Now()}}, bson.M{"refresh_lease_until": bson.M{"$exists": false}}}
	err = client.Database("buct-course").Collection("users").FindOne(ctx, filter,
		options.FindOne().SetProjection(bson.M{"next_refresh_at": 1}).SetSort(bson.D{{Key: "next_refresh_at", Value: 1}}),
	).Decode(&next)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return 5 * time.Minute
	}
	if err != nil {
		return time.Minute
	}
	// Reconciliation discovers writes made by another backend instance.
	return delayUntil(next.At, 5*time.Minute)
}

func nextReminderDelay() time.Duration {
	return nextCollectionDelay(RemindersCollection, bson.M{"status": "scheduled"}, "scheduled_time")
}

func nextAutoReminderDelay() time.Duration {
	return nextCollectionDelay("course_data", bson.M{
		"auto_reminder_pending": true, "deadline_at": bson.M{"$gte": time.Now()},
	}, "auto_reminder_at")
}

func nextCollectionDelay(collection string, filter bson.M, field string) time.Duration {
	client, err := models.ConnectToDB()
	if err != nil {
		return time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var doc bson.M
	err = client.Database("buct-course").Collection(collection).FindOne(ctx, filter,
		options.FindOne().SetProjection(bson.M{field: 1}).SetSort(bson.D{{Key: field, Value: 1}}),
	).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return 5 * time.Minute
	}
	if err != nil {
		return time.Minute
	}
	if value, ok := doc[field].(interface{ Time() time.Time }); ok {
		return delayUntil(value.Time(), time.Minute)
	}
	return time.Minute
}
