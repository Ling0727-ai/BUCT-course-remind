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
