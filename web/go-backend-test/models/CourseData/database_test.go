package CourseData

import (
	"errors"
	"testing"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func existingTask(input TaskInput) bson.D {
	return existingTaskWithID(input, generateTaskID(input.Type, input.Subject, input.Title, input.Details))
}

func existingTaskWithID(input TaskInput, taskID string) bson.D {
	task := CourseData{ID: "stable-id", UserID: "user", TaskID: taskID,
		Subject: input.Subject, Title: input.Title, Deadline: input.Deadline, Details: input.Details, Url: input.Url, Type: input.Type,
		CreatedAt: time.Now().Add(-time.Hour), UpdatedAt: time.Now().Add(-time.Hour)}
	normalizeSchedule(&task)
	data, _ := bson.Marshal(task)
	var doc bson.D
	_ = bson.Unmarshal(data, &doc)
	return doc
}

func TestTaskIdentitySurvivesDeadlineChange(t *testing.T) {
	old := TaskInput{Type: "homework", Subject: "math", Title: "assignment", Details: "same content", Deadline: "2026-11-01 12:00:00"}
	changed := old
	changed.Deadline = "2026-11-02 12:00:00"
	prior := &CourseData{TaskID: legacyTaskID(old), Type: old.Type, Subject: old.Subject, Title: old.Title, Details: old.Details, Deadline: old.Deadline}
	identity := generateTaskID(old.Type, old.Subject, old.Title, old.Details)
	id, matched := resolveTaskID(changed, map[string][]*CourseData{identity: {prior}}, map[string]bool{})
	if id != prior.TaskID || matched != prior {
		t.Fatalf("legacy ID was not preserved: id=%q prior=%v", id, matched)
	}
}

func TestNewTaskIdentityExcludesDeadline(t *testing.T) {
	task := TaskInput{Type: "test", Subject: "math", Title: "quiz", Deadline: "2026-11-01 12:00:00"}
	id, prior := resolveTaskID(task, nil, map[string]bool{})
	copyID, _ := resolveTaskID(task, nil, map[string]bool{id: true})
	task.Deadline = "2026-11-02 12:00:00"
	changedID, changedPrior := resolveTaskID(task, nil, map[string]bool{})
	changedCopyID, _ := resolveTaskID(task, nil, map[string]bool{id: true})
	if prior != nil || changedPrior != nil || id != changedID || copyID != changedCopyID || copyID == id {
		t.Fatalf("deadline changed new task identity: before=%q after=%q copy=%q changedCopy=%q", id, changedID, copyID, changedCopyID)
	}
}

func TestDeadlineChangePreservesLegacyIDAndRearmsReminders(t *testing.T) {
	previousGenerator := config.Snowflake
	var err error
	config.Snowflake, err = config.NewSnowflakeGenerator()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { config.Snowflake = previousGenerator })
	old := TaskInput{Type: "homework", Subject: "math", Title: "assignment", Details: "same content", Deadline: "2026-11-01 12:00:00"}
	changed := old
	changed.Deadline = "2026-11-02 12:00:00"
	legacyID := legacyTaskID(old)
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("changed deadline", func(mt *mtest.T) {
		mt.AddMockResponses(
			mtest.CreateCursorResponse(0, mt.DB.Name()+"."+mt.Coll.Name(), mtest.FirstBatch, existingTaskWithID(old, legacyID)),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
		)
		repo := &MongoCourseDataRepository{collection: mt.Coll}
		count, err := repo.SaveUserCourseData("user", []TaskInput{changed})
		if err != nil || count != 1 {
			mt.Fatalf("count=%d err=%v", count, err)
		}
		courseUpdated, remindersRearmed := false, false
		for _, event := range mt.GetAllStartedEvents() {
			if event.CommandName == "delete" {
				mt.Fatal("deadline edit deleted the existing task")
			}
			if event.CommandName != "update" {
				continue
			}
			rows, _ := event.Command.Lookup("updates").Array().Values()
			filter := rows[0].Document().Lookup("q").Document()
			switch event.Command.Lookup("update").StringValue() {
			case mt.Coll.Name():
				courseUpdated = filter.Lookup("task_id").StringValue() == legacyID
			case "scheduled_reminders":
				remindersRearmed = filter.Lookup("target_id").StringValue() == legacyID
			}
		}
		if !courseUpdated || !remindersRearmed {
			mt.Fatalf("courseUpdated=%v remindersRearmed=%v", courseUpdated, remindersRearmed)
		}
	})
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
				rows, _ := event.Command.Lookup("updates").Array().Values()
				if rows[0].Document().Lookup("u").Document().Lookup("$unset").Document().Lookup("empty_refresh_seen_at").Type == 0 {
					mt.Fatal("a successful nonempty refresh kept the empty-result marker")
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

func TestFirstEmptyRefreshPreservesExistingTasks(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("first empty result", func(mt *mtest.T) {
		old := TaskInput{Type: "homework", Subject: "math", Title: "assignment"}
		mt.AddMockResponses(
			mtest.CreateCursorResponse(0, mt.DB.Name()+"."+mt.Coll.Name(), mtest.FirstBatch, existingTask(old)),
			mtest.CreateCursorResponse(0, mt.DB.Name()+".users", mtest.FirstBatch, bson.D{{Key: "_id", Value: "user"}}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
		)
		repo := &MongoCourseDataRepository{collection: mt.Coll}
		if _, err := repo.SaveUserCourseData("user", nil); !errors.Is(err, ErrEmptyRefreshUnconfirmed) {
			mt.Fatalf("expected confirmation request, got %v", err)
		}
		markerWritten := false
		for _, event := range mt.GetAllStartedEvents() {
			if event.CommandName == "delete" || (event.CommandName == "update" && event.Command.Lookup("update").StringValue() == mt.Coll.Name()) {
				mt.Fatal("first empty result changed existing course data")
			}
			if event.CommandName == "update" && event.Command.Lookup("update").StringValue() == "users" {
				rows, _ := event.Command.Lookup("updates").Array().Values()
				markerWritten = rows[0].Document().Lookup("u").Document().Lookup("$set").Document().Lookup("empty_refresh_seen_at").Type != 0
			}
		}
		if !markerWritten {
			mt.Fatal("empty result was not recorded for confirmation")
		}
	})
}

func TestConfirmedEmptyRefreshRemovesExistingTasks(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("second empty result", func(mt *mtest.T) {
		old := TaskInput{Type: "homework", Subject: "math", Title: "assignment"}
		mt.AddMockResponses(
			mtest.CreateCursorResponse(0, mt.DB.Name()+"."+mt.Coll.Name(), mtest.FirstBatch, existingTask(old)),
			mtest.CreateCursorResponse(0, mt.DB.Name()+".users", mtest.FirstBatch,
				bson.D{{Key: "_id", Value: "user"}, {Key: "empty_refresh_seen_at", Value: time.Now().Add(-time.Minute)}}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
		)
		repo := &MongoCourseDataRepository{collection: mt.Coll}
		if count, err := repo.SaveUserCourseData("user", nil); err != nil || count != 0 {
			mt.Fatalf("count=%d err=%v", count, err)
		}
		deleted, completedStatusCleared, markerCleared := false, false, false
		for _, event := range mt.GetAllStartedEvents() {
			if event.CommandName == "delete" && event.Command.Lookup("delete").StringValue() == mt.Coll.Name() {
				deleted = true
			}
			if event.CommandName == "delete" && event.Command.Lookup("delete").StringValue() == "assignment_status" {
				completedStatusCleared = true
			}
			if event.CommandName == "update" && event.Command.Lookup("update").StringValue() == "users" {
				rows, _ := event.Command.Lookup("updates").Array().Values()
				markerCleared = rows[0].Document().Lookup("u").Document().Lookup("$unset").Document().Lookup("empty_refresh_seen_at").Type != 0
			}
		}
		if !deleted || !completedStatusCleared || !markerCleared {
			mt.Fatalf("deleted=%v completedStatusCleared=%v markerCleared=%v", deleted, completedStatusCleared, markerCleared)
		}
	})
}

func TestImmediateSecondEmptyRefreshDoesNotConfirm(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("too soon", func(mt *mtest.T) {
		old := TaskInput{Type: "homework", Subject: "math", Title: "assignment"}
		mt.AddMockResponses(
			mtest.CreateCursorResponse(0, mt.DB.Name()+"."+mt.Coll.Name(), mtest.FirstBatch, existingTask(old)),
			mtest.CreateCursorResponse(0, mt.DB.Name()+".users", mtest.FirstBatch,
				bson.D{{Key: "_id", Value: "user"}, {Key: "empty_refresh_seen_at", Value: time.Now().Add(-5 * time.Second)}}),
		)
		repo := &MongoCourseDataRepository{collection: mt.Coll}
		if _, err := repo.SaveUserCourseData("user", nil); !errors.Is(err, ErrEmptyRefreshUnconfirmed) {
			mt.Fatalf("expected pending confirmation, got %v", err)
		}
		for _, event := range mt.GetAllStartedEvents() {
			if event.CommandName == "delete" {
				mt.Fatal("an immediate retry deleted course data")
			}
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
