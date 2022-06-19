package mongo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetResults returns cursor of aggregate query results
func GetResults(ctx context.Context, collection *mongo.Collection, match bson.M, sort bson.D, skip int64, limit int64) (*mongo.Cursor, error) {
	pipeline := []bson.M{
		{"$match": match},
		{"$sort": sort},
		{"$skip": skip},
		{"$limit": limit},
	}
	cursor, err := collection.Aggregate(ctx, pipeline)
	return cursor, err
}

// GetCount returns count of query result
func GetCount(ctx context.Context, collection *mongo.Collection, match bson.M) (int64, error) {
	count, err := collection.CountDocuments(ctx, match)
	return count, err
}
