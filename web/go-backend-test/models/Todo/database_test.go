package Todo

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestCompletedTodoHasNoExpiry(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("completed", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}))
		repo := &MongoTodoRepository{collection: mt.Coll}
		if err := repo.MarkCompleted("todo"); err != nil {
			mt.Fatal(err)
		}
		rows, _ := mt.GetAllStartedEvents()[0].Command.Lookup("updates").Array().Values()
		update := rows[0].Document().Lookup("u").Document()
		if !update.Lookup("$set").Document().Lookup("completed").Boolean() ||
			update.Lookup("$set").Document().Lookup("expires_at").Type != 0 ||
			update.Lookup("$unset").Document().Lookup("expires_at").Type == 0 {
			mt.Fatal("completed todo would still expire")
		}
	})
}

func TestInitRemovesLegacyCompletedTodoExpiry(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("migration", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}))
		if err := Init(mt.DB); err != nil {
			mt.Fatal(err)
		}
		rows, _ := mt.GetAllStartedEvents()[0].Command.Lookup("updates").Array().Values()
		query := rows[0].Document().Lookup("q").Document()
		if !query.Lookup("completed").Boolean() ||
			rows[0].Document().Lookup("u").Document().Lookup("$unset").Document().Lookup("expires_at").Type == 0 {
			mt.Fatal("legacy completed todo expiry was not removed")
		}
	})
}
