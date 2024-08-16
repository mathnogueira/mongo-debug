package mongodebug_test

import (
	"context"
	"math/rand/v2"
	"testing"

	"github.com/jaswdr/faker/v2"
	mongodebug "github.com/mathnogueira/mongo-debug"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestDebugFindWithoutIndex(t *testing.T) {
	ctx := context.Background()
	client, chosenUserName := setup(t, false)

	_, err := mongodebug.
		Debug(mongodebug.WithThreshold(10)).
		WithDatabase(client.Database("db")).
		WithCollection("collection").
		Find(ctx, bson.D{{"name", chosenUserName}})

	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "total docs examined surpassed threashold")
}

func TestDebugFindWithIndex(t *testing.T) {
	ctx := context.Background()
	client, chosenUserName := setup(t, true)

	cursor, err := mongodebug.
		Debug(mongodebug.WithThreshold(10)).
		WithDatabase(client.Database("db")).
		WithCollection("collection").
		Find(ctx, bson.D{{"name", chosenUserName}})

	require.Nil(t, err)
	require.NotNil(t, cursor)
}

func setup(t *testing.T, withIndex bool) (*mongo.Client, string) {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017/db"))
	require.NoError(t, err)

	err = client.Database("db").CreateCollection(ctx, "collection")
	require.NoError(t, err)

	if withIndex {
		_, err = client.Database("db").Collection("collection").Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.M{"name": 1},
		})

		require.NoError(t, err)
	}

	faker := faker.New()

	randomNumber := rand.IntN(10_000)
	var name string

	for i := 0; i < 10_000; i++ {
		userName := faker.Person().Name()
		if randomNumber == i {
			name = userName
		}

		_, err = client.Database("db").Collection("collection").InsertOne(ctx, bson.D{{"name", userName}})
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		err = client.Database("db").Collection("collection").Drop(ctx)
		require.NoError(t, err)
	})

	return client, name
}
