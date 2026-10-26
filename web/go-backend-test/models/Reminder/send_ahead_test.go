package Reminder

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestExistingScheduledAutoReminderMovesEarlierOnce(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("shift legacy scheduled job", func(mt *mtest.T) {
		at := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
		ns := mt.DB.Name() + "."
		mt.AddMockResponses(
			mtest.CreateCursorResponse(0, ns+"_migrations", mtest.FirstBatch),
			mtest.CreateCursorResponse(0, ns+mt.Coll.Name(), mtest.FirstBatch, bson.D{{Key: "_id", Value: "legacy"}, {Key: "scheduled_time", Value: at}}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
		)
		repo := &MongoReminderRepository{collection: mt.Coll}
		if err := repo.BackfillSendAhead(); err != nil {
			mt.Fatal(err)
		}
		shifted := false
		for _, event := range mt.GetAllStartedEvents() {
			if event.CommandName != "update" || event.Command.Lookup("update").StringValue() != mt.Coll.Name() {
				continue
			}
			writes, _ := event.Command.Lookup("updates").Array().Values()
			query := writes[0].Document().Lookup("q").Document()
			if query.Lookup("status").StringValue() != "scheduled" || !query.Lookup("auto_created").Boolean() {
				mt.Fatal("migration could modify sent or manual reminders")
			}
			if query.Lookup("scheduled_time").DateTime() != at.UnixMilli() || query.Lookup("send_ahead_seconds").Document().Lookup("$exists").Boolean() {
				mt.Fatal("migration lost its compare-and-set guard")
			}
			set := writes[0].Document().Lookup("u").Document().Lookup("$set").Document()
			if set.Lookup("scheduled_time").DateTime() != at.Add(-5*time.Second).UnixMilli() || set.Lookup("send_ahead_seconds").Int32() != 5 {
				mt.Fatal("existing reminder was not advanced exactly five seconds")
			}
			shifted = true
		}
		if !shifted {
			mt.Fatal("scheduled reminder was not migrated")
		}
	})
}

func TestSendAheadMigrationDoesNotRunAgainAfterCompletion(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("already migrated", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+"._migrations", mtest.FirstBatch, bson.D{{Key: "_id", Value: "auto_reminder_send_ahead_5s_v1"}}))
		repo := &MongoReminderRepository{collection: mt.Coll}
		if err := repo.BackfillSendAhead(); err != nil {
			mt.Fatal(err)
		}
		if len(mt.GetAllStartedEvents()) != 1 {
			mt.Fatal("completed migration attempted another shift")
		}
	})
}
