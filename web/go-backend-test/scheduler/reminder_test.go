package scheduler

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestAutoRemindersBatchQueriesKeepUserStatesSeparate(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("two users sharing a task id", func(mt *mtest.T) {
		now := time.Now()
		task := func(id, user string) bson.D {
			return bson.D{{Key: "_id", Value: id}, {Key: "user_id", Value: user}, {Key: "task_id", Value: "shared-task"},
				{Key: "deadline", Value: "2026-12-01 12:00:00"}, {Key: "deadline_at", Value: now.Add(time.Hour)}}
		}
		ns := mt.DB.Name() + "."
		mt.AddMockResponses(
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}),
			mtest.CreateCursorResponse(0, ns+"course_data", mtest.FirstBatch, task("a-task", "a"), task("b-task", "b")),
			mtest.CreateCursorResponse(0, ns+"users", mtest.FirstBatch, bson.D{{Key: "_id", Value: "a"}, {Key: "email", Value: "a@example.com"}}, bson.D{{Key: "_id", Value: "b"}, {Key: "email", Value: "b@example.com"}}),
			mtest.CreateCursorResponse(0, ns+"assignment_status", mtest.FirstBatch, bson.D{{Key: "user_id", Value: "a"}, {Key: "assignment_id", Value: "shared-task"}, {Key: "status", Value: "completed"}}),
			mtest.CreateCursorResponse(0, ns+"scheduled_reminders", mtest.FirstBatch),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 2}),
		)
		if err := createAutoReminders(mt.DB, now); err != nil {
			mt.Fatal(err)
		}
		reminderWrites := 0
		for _, event := range mt.GetAllStartedEvents() {
			if event.CommandName == "aggregate" {
				mt.Fatal("per-task count query returned")
			}
			if event.CommandName == "update" && event.Command.Lookup("update").StringValue() == "scheduled_reminders" {
				reminderWrites++
				updates, _ := event.Command.Lookup("updates").Array().Values()
				if len(updates) != 1 {
					mt.Fatal("expected one reminder for the active user")
				}
				id := updates[0].Document().Lookup("q").Document().Lookup("_id").StringValue()
				if id != autoReminderID("b", "shared-task") {
					mt.Fatal("one user's completed status leaked into another user")
				}
			}
		}
		if reminderWrites != 1 {
			mt.Fatalf("reminder writes=%d", reminderWrites)
		}
	})
}

func TestAutoReminderIDIsStableAndUserScoped(t *testing.T) {
	if autoReminderID("a", "task") != autoReminderID("a", "task") {
		t.Fatal("unstable id")
	}
	if autoReminderID("ab", "c") == autoReminderID("a", "bc") {
		t.Fatal("ambiguous key encoding")
	}
}
