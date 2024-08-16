package mongodebug

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type DebugSession interface {
	WithDatabase(database *mongo.Database) DebugSession
	WithCollection(collection string) DebugSession
	Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error)
}

type debugSession struct {
	database       *mongo.Database
	collection     *mongo.Collection
	collectionName string

	threshold int
}

func (d *debugSession) WithDatabase(database *mongo.Database) DebugSession {
	d.database = database
	return d
}

func (d *debugSession) WithCollection(name string) DebugSession {
	d.collection = d.database.Collection(name)
	d.collectionName = name
	return d
}

func (d *debugSession) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	err := d.runExplain(ctx, "find", filter)
	if err != nil {
		return nil, err
	}

	return d.collection.Find(ctx, filter, opts...)
}

func Debug(opts ...Option) DebugSession {
	session := &debugSession{threshold: 100}
	for _, opt := range opts {
		opt(session)
	}

	return session
}

type explain struct {
	ExecutionStats executionStats `json:"executionStats"`
}

type executionStats struct {
	TotalDocsExamined int `json:"totalDocsExamined"`
}

func (d *debugSession) runExplain(ctx context.Context, name string, filter interface{}) error {
	command := bson.D{{name, d.collectionName}, {"filter", filter}}
	explainCommand := bson.D{{"explain", command}}
	explainOpts := options.RunCmd().SetReadPreference(readpref.Primary())
	var result bson.D
	err := d.database.RunCommand(ctx, explainCommand, explainOpts).Decode(&result)
	if err != nil {
		return fmt.Errorf("could not run explain on query: %w", err)
	}

	bytes, err := bson.Marshal(result)
	if err != nil {
		return fmt.Errorf("could not marshal explain result: %w", err)
	}

	var explainResult explain
	err = bson.Unmarshal(bytes, &explainResult)
	if err != nil {
		return fmt.Errorf("could not unmarshal into explain result: %w", err)
	}

	if explainResult.ExecutionStats.TotalDocsExamined > d.threshold {
		filterString, _ := bson.Marshal(filter)
		return fmt.Errorf("query plan: total docs examined surpassed threashold: '%s' {%s}", name, string(filterString))
	}

	return nil
}
