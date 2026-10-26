package scheduler

import (
	"strings"
	"testing"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/models/CourseData"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestAutoRemindersBatchQueriesKeepUserStatesSeparate(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("two users sharing a task id", func(mt *mtest.T) {
		now := time.Now()
		task := func(id, user string) bson.D {
			return bson.D{{Key: "_id", Value: id}, {Key: "user_id", Value: user}, {Key: "task_id", Value: "shared-task"},
				{Key: "deadline", Value: "2026-12-01 12:00:00"}, {Key: "deadline_at", Value: now.Add(24 * time.Hour)}}
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
				if len(updates) != 3 {
					mt.Fatal("expected three reminders for the active user")
				}
				for _, update := range updates {
					doc := update.Document()
					set := doc.Lookup("u").Document().Lookup("$setOnInsert").Document()
					hours := int(set.Lookup("offset_hours").Int32())
					id := doc.Lookup("q").Document().Lookup("_id").StringValue()
					if id != autoReminderID("b", "shared-task", hours) {
						mt.Fatal("one user's completed status leaked into another user")
					}
					planned := set.Lookup("scheduled_time").DateTime()
					if set.Lookup("send_ahead_seconds").Int32() != 5 {
						mt.Fatal("new reminders were not marked as already advanced")
					}
					if planned != now.Add(time.Duration(24-hours)*time.Hour-5*time.Second).UnixMilli() {
						mt.Fatal("stage did not use its exact planned timestamp")
					}
				}
			}
		}
		if reminderWrites != 1 {
			mt.Fatalf("reminder writes=%d", reminderWrites)
		}
	})
}

func TestAutoReminderIDIsStableAndUserScoped(t *testing.T) {
	if autoReminderID("a", "task", 24) != autoReminderID("a", "task", 24) {
		t.Fatal("unstable id")
	}
	if autoReminderID("ab", "c", 24) == autoReminderID("a", "bc", 24) {
		t.Fatal("ambiguous key encoding")
	}
	if autoReminderID("a", "task", 24) == autoReminderID("a", "task", 12) || autoReminderID("a", "task", 12) == autoReminderID("a", "task", 1) {
		t.Fatal("reminder stages share the same ID")
	}
	if autoReminderID("a", "task", 0) != autoReminderID("a", "task", 24) || autoReminderKey("a", "task", 0) != autoReminderKey("a", "task", 24) {
		t.Fatal("legacy reminder was not mapped to the 24-hour stage")
	}
}

func TestLegacy24HourReminderDoesNotSuppressLaterStages(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("legacy sent", func(mt *mtest.T) {
		now := time.Now()
		ns := mt.DB.Name() + "."
		mt.AddMockResponses(
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}),
			mtest.CreateCursorResponse(0, ns+"course_data", mtest.FirstBatch, bson.D{{Key: "_id", Value: "task-row"}, {Key: "user_id", Value: "user"}, {Key: "task_id", Value: "task"}, {Key: "deadline", Value: "deadline"}, {Key: "deadline_at", Value: now.Add(24 * time.Hour)}}),
			mtest.CreateCursorResponse(0, ns+"users", mtest.FirstBatch, bson.D{{Key: "_id", Value: "user"}, {Key: "email", Value: "user@example.com"}}),
			mtest.CreateCursorResponse(0, ns+"assignment_status", mtest.FirstBatch),
			mtest.CreateCursorResponse(0, ns+"scheduled_reminders", mtest.FirstBatch, bson.D{{Key: "_id", Value: "legacy"}, {Key: "user_id", Value: "user"}, {Key: "target_id", Value: "task"}, {Key: "status", Value: "sent"}}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 2}), mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
		)
		if err := createAutoReminders(mt.DB, now); err != nil {
			mt.Fatal(err)
		}
		for _, event := range mt.GetAllStartedEvents() {
			if event.CommandName != "update" || event.Command.Lookup("update").StringValue() != "scheduled_reminders" {
				continue
			}
			updates, _ := event.Command.Lookup("updates").Array().Values()
			if len(updates) != 2 {
				mt.Fatalf("expected 12h and 1h stages, got %d", len(updates))
			}
			for _, update := range updates {
				hours := update.Document().Lookup("u").Document().Lookup("$setOnInsert").Document().Lookup("offset_hours").Int32()
				if hours != 12 && hours != 1 {
					mt.Fatal("legacy 24-hour reminder would be repeated")
				}
			}
		}
	})
}

func TestExpiredAutoReminderIsSkippedBeforeDatabaseReads(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	at := now.Add(12 * time.Hour)
	message, reason, err := reminderDeliveryContent(ScheduledReminder{AutoCreated: true, OffsetHours: 24, DeadlineAt: &at}, now)
	if err != nil || message != "" || reason == "" {
		t.Fatal("superseded reminders must be skipped without sending")
	}
}

func TestReminderMessageUsesCurrentRemainingTime(t *testing.T) {
	at := time.Date(2026, 10, 30, 18, 37, 42, 0, time.UTC)
	message := buildAutoReminderMessage(&CourseData.CourseData{Title: "作业"}, at, at.Add(-11*time.Hour-59*time.Minute-58*time.Second), 12)
	if !strings.Contains(message, "截止前 12 小时") || !strings.Contains(message, "11小时59分58秒") {
		t.Fatalf("message has stale remaining time: %s", message)
	}
}
