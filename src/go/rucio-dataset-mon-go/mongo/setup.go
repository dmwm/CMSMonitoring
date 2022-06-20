package mongo

import (
	"context"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/configs"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

// DBClient MongoDb Client instance(mongo.Client)
var DBClient = GetMongoClient()

// GetMongoClient returns mongo client
func GetMongoClient() *mongo.Client {
	var err error
	var client *mongo.Client
	opts := options.Client()
	opts.ApplyURI(configs.EnvMongoURI())
	opts.SetMaxPoolSize(100)
	opts.SetConnectTimeout(time.Duration(300) * time.Second)
	opts.SetMaxConnIdleTime(time.Microsecond * 100000)
	if client, err = mongo.Connect(context.Background(), opts); err != nil {
		log.Fatal(err)
	}
	if err := client.Ping(context.Background(), nil); err != nil {
		log.Fatal(err)
	}
	return client
}

// GetCollection returns Mongo db collection with the given collection name
func GetCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	collection := client.Database(configs.EnvMongoDB()).Collection(collectionName)
	return collection
}
