package mongo

import (
	"context"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/configs"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

// Global variables

var DBClient *mongo.Client
var DB string
var URI string
var ConnectionTimeout = 100
var Timeout = time.Duration(ConnectionTimeout) * time.Second

// GetMongoClient returns mongo client
func GetMongoClient() *mongo.Client {
	var err error
	var client *mongo.Client
	//opts.SetMaxPoolSize(100)
	//opts.SetConnectTimeout(time.Duration(ConnectionTimeout) * time.Second)
	//opts.SetMaxConnIdleTime(time.Microsecond * 100000)
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	if client, err = mongo.Connect(ctx, options.Client().ApplyURI(URI).SetConnectTimeout(Timeout)); err != nil {
		log.Fatal(err)
	}
	if err := client.Ping(context.Background(), nil); err != nil {
		log.Fatal(err)
	}
	return client
}

// GetCollection returns Mongo db collection with the given collection name
func GetCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	collection := client.Database(DB).Collection(collectionName)
	return collection
}

// InitializeClient setup client connection in main function
func InitializeClient() {
	configs.InitialChecks()
	DB = configs.EnvMongoDB()
	URI = configs.EnvMongoURI()
	DBClient = GetMongoClient()
}
