package mongo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetAggQueryResults returns cursor of aggregate query results
func GetAggQueryResults(ctx context.Context, collection *mongo.Collection, match bson.M, sort bson.D, skip int64, length int64) (*mongo.Cursor, error) {
	pipeline := []bson.M{
		{"$match": match},
		{"$sort": sort},
		{"$skip": skip},
		{"$limit": length},
	}
	opts := options.AggregateOptions{}
	opts.SetBatchSize(int32(length))
	opts.SetAllowDiskUse(true)
	cursor, err := collection.Aggregate(ctx, pipeline, &opts)
	return cursor, err
}

// GetFindQueryResults returns cursor of find query results
func GetFindQueryResults(ctx context.Context, collection *mongo.Collection, match bson.M, sort bson.D, skip int64, length int64) (*mongo.Cursor, error) {
	//pipeline := []bson.M{
	//	{"$match": match},
	//}
	opts := options.FindOptions{}
	opts.SetBatchSize(int32(length))
	opts.SetSort(sort)
	opts.SetSkip(skip)
	opts.SetLimit(length)
	opts.SetAllowDiskUse(true)
	cursor, err := collection.Find(context.TODO(), match, &opts)
	return cursor, err
}

// GetCount returns count of query result
func GetCount(ctx context.Context, collection *mongo.Collection, match bson.M) (int64, error) {
	count, err := collection.CountDocuments(context.TODO(), match)
	return count, err
}
