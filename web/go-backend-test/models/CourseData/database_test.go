package CourseData

import (
	"testing"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func existingTask(input TaskInput) bson.D {
	task := CourseData{ID: "stable-id", UserID: "user", TaskID: generateTaskID(input.Type, input.Subject, input.Title, input.Deadline, input.Details),
		Subject: input.Subject, Title: input.Title, Deadline: input.Deadline, Details: input.Details, Url: input.Url, Type: input.Type,
		CreatedAt: time.Now().Add(-time.Hour), UpdatedAt: time.Now().Add(-time.Hour)}
	normalizeSchedule(&task)
	data, _ := bson.Marshal(task)
	var doc bson.D
	_ = bson.Unmarshal(data, &doc)
	return doc
}

func TestUnchangedRefreshOnlyWritesFreshness(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("unchanged", func(mt *mtest.T) {
		input := TaskInput{Subject: "math", Title: "task", Type: "homework", Deadline: "2026-12-01 12:00:00", Details: "large description"}
		mt.AddMockResponses(
			mtest.CreateCursorResponse(0, mt.DB.Name()+"."+mt.Coll.Name(), mtest.FirstBatch, existingTask(input)),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
		)
		repo := &MongoCourseDataRepository{collection: mt.Coll}
		count, err := repo.SaveUserCourseData("user", []TaskInput{input, input})
		if err != nil || count != 1 {
			mt.Fatalf("count=%d err=%v", count, err)
		}
		writes := 0
		for _, event := range mt.GetAllStartedEvents() {
			if event.CommandName == "delete" {
				mt.Fatal("unchanged tasks were deleted")
			}
			if event.CommandName == "update" {
				writes++
				if event.Command.Lookup("update").StringValue() != "users" {
					mt.Fatal("unchanged task was rewritten")
				}
			}
		}
		if writes != 1 {
			mt.Fatalf("expected one metadata write, got %d", writes)
		}
	})
}

func TestFailedUpsertDoesNotPruneExistingTasks(t *testing.T) {
	previous := config.Snowflake
	var err error
	config.Snowflake, err = config.NewSnowflakeGenerator()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { config.Snowflake = previous })
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("failed write", func(mt *mtest.T) {
		old := TaskInput{Subject: "math", Title: "old", Type: "homework"}
		mt.AddMockResponses(
			mtest.CreateCursorResponse(0, mt.DB.Name()+"."+mt.Coll.Name(), mtest.FirstBatch, existingTask(old)),
			mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "write failed"}),
		)
		repo := &MongoCourseDataRepository{collection: mt.Coll}
		if _, err := repo.SaveUserCourseData("user", []TaskInput{{Subject: "math", Title: "new", Type: "homework"}}); err == nil {
			mt.Fatal("expected failed upsert")
		}
		for _, event := range mt.GetAllStartedEvents() {
			if event.CommandName == "delete" {
				mt.Fatal("old tasks were pruned after a failed upsert")
			}
		}
	})
}

func TestEmptySuccessfulRefreshAdvancesFreshness(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("empty", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+"."+mt.Coll.Name(), mtest.FirstBatch), mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}))
		repo := &MongoCourseDataRepository{collection: mt.Coll}
		if count, err := repo.SaveUserCourseData("user", nil); err != nil || count != 0 {
			mt.Fatalf("count=%d err=%v", count, err)
		}
		events := mt.GetAllStartedEvents()
		if len(events) != 2 || events[1].Command.Lookup("update").StringValue() != "users" {
			mt.Fatal("empty refresh did not record freshness")
		}
	})
}

func TestLegacyReminderMigrationRearmsAlreadyNormalizedTasks(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("v1 to three stages", func(mt *mtest.T) {
		ns := mt.DB.Name() + "."
		mt.AddMockResponses(
			mtest.CreateCursorResponse(0, ns+"_migrations", mtest.FirstBatch),
			mtest.CreateCursorResponse(0, ns+mt.Coll.Name(), mtest.FirstBatch, bson.D{{Key: "_id", Value: "legacy-task"}, {Key: "deadline", Value: "2026-12-01 12:00:00"}}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
		)
		repo := &MongoCourseDataRepository{collection: mt.Coll}
		if err := repo.BackfillDeadlines(); err != nil {
			mt.Fatal(err)
		}
		migrated := false
		for _, event := range mt.GetAllStartedEvents() {
			if event.CommandName == "find" && event.Command.Lookup("find").StringValue() == "_migrations" {
				if event.Command.Lookup("filter").Document().Lookup("_id").StringValue() != "course_auto_reminders_v3" {
					mt.Fatal("v1 migration marker would suppress the three-stage upgrade")
				}
			}
			if event.CommandName == "update" && event.Command.Lookup("update").StringValue() == mt.Coll.Name() {
				writes, _ := event.Command.Lookup("updates").Array().Values()
				set := writes[0].Document().Lookup("u").Document().Lookup("$set").Document()
				if !set.Lookup("auto_reminder_pending").Boolean() || set.Lookup("auto_reminder_version").Int32() != 3 {
					mt.Fatal("legacy task did not re-enter reminder scheduling")
				}
				if set.Lookup("auto_reminder_at").DateTime() != time.Date(2026, 11, 30, 3, 59, 25, 0, time.UTC).UnixMilli() {
					mt.Fatal("task preparation did not start thirty seconds before its early-send timestamp")
				}
				migrated = true
			}
		}
		if !migrated {
			mt.Fatal("legacy task was not migrated")
		}
	})
}
