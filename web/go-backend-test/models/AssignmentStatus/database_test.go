package AssignmentStatus

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestCompletedStatusHasNoExpiry(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("completed", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}))
		repo := &MongoAssignmentStatusRepository{collection: mt.Coll}
		if err := repo.MarkCompleted("user", "task", "title", "subject"); err != nil {
			mt.Fatal(err)
		}
		rows, _ := mt.GetAllStartedEvents()[0].Command.Lookup("updates").Array().Values()
		update := rows[0].Document().Lookup("u").Document()
		if update.Lookup("$set").Document().Lookup("status").StringValue() != StatusCompleted ||
			update.Lookup("$set").Document().Lookup("expires_at").Type != 0 ||
			update.Lookup("$unset").Document().Lookup("expires_at").Type == 0 {
			mt.Fatal("completed assignment would still expire")
		}
	})
}

func TestInitRemovesLegacyCompletedExpiry(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("migration", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}))
		if err := Init(mt.DB); err != nil {
			mt.Fatal(err)
		}
		rows, _ := mt.GetAllStartedEvents()[0].Command.Lookup("updates").Array().Values()
		query := rows[0].Document().Lookup("q").Document()
		if query.Lookup("status").StringValue() != StatusCompleted ||
			rows[0].Document().Lookup("u").Document().Lookup("$unset").Document().Lookup("expires_at").Type == 0 {
			mt.Fatal("legacy completed assignment expiry was not removed")
		}
	})
}
