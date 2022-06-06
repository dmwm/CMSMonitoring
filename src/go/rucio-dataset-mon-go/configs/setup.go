package configs

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

// ConnectDB mongo.Client which is MongoDB connection client
func ConnectDB() *mongo.Client {
	client, err := mongo.NewClient(options.Client().ApplyURI(EnvMongoURI()))
	if err != nil {
		log.Fatal(err)
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Duration(EnvConnTimeout())*time.Second)
	if err = client.Connect(ctx); err != nil {
		log.Fatal(err)
	}

	//ping the database
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to MongoDB")
	return client
}

// DB MongoDb Client instance(mongo.Client)
var DB = ConnectDB()

// GetCollection returns Mongo db collection with the given collection name
func GetCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	collection := client.Database(EnvMongoDB()).Collection(collectionName)
	return collection
}
