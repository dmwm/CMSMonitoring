package mongo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MaxLimit = int64(10000)

// GetFindQueryResults returns cursor of find query results
func GetFindQueryResults(ctx context.Context, coll *mongo.Collection, match bson.M, sort bson.D, skip int64, length int64) (*mongo.Cursor, error) {
	opts := options.FindOptions{}
	if len(sort) > 0 {
		opts.SetSort(sort)
	}
	if skip > 0 {
		opts.SetSkip(skip)
	}
	if length <= 0 || length > MaxLimit {
		length = MaxLimit
	}
	opts.SetLimit(length)
	opts.SetAllowDiskUse(true)
	//opts.SetBatchSize(int32(length))
	cursor, err := coll.Find(ctx, match, &opts)
	return cursor, err
}

// GetFindOnlyMatchResults no sort, skip, limit, just match
func GetFindOnlyMatchResults(ctx context.Context, coll *mongo.Collection, match bson.M) (*mongo.Cursor, error) {
	opts := options.FindOptions{}
	opts.SetAllowDiskUse(true)
	cursor, err := coll.Find(ctx, match, &opts)
	return cursor, err
}

// GetCount returns count of query result
func GetCount(ctx context.Context, collection *mongo.Collection, match bson.M) (int64, error) {
	count, err := collection.CountDocuments(context.TODO(), match)
	return count, err
}

// Insert returns count of query result
func Insert(ctx context.Context, collection *mongo.Collection, data interface{}) error {
	_, err := collection.InsertOne(ctx, data)
	return err
}
