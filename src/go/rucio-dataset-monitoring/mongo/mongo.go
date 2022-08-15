package mongo

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/utils"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
	"time"
)

// MaxPageDocumentsLimit max number of records that will be returned as a result of find query
const MaxPageDocumentsLimit = int64(10000)

// Timeout mongo connection timeout
var Timeout time.Duration

var mongoDatabase *mongo.Database

// GetCollection returns Mongo db collection with the given collection name
func GetCollection(collectionName string) *mongo.Collection {
	collection := mongoDatabase.Collection(collectionName)
	return collection
}

// GetFindQueryResults returns cursor of find query results
func GetFindQueryResults(ctx context.Context, coll *mongo.Collection, match bson.M, sort bson.D, skip int64, length int64) (*mongo.Cursor, error) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	opts := options.FindOptions{}
	if len(sort) > 0 {
		opts.SetSort(sort)
	}
	if skip > 0 {
		opts.SetSkip(skip)
	}
	if length <= 0 || length > MaxPageDocumentsLimit {
		length = MaxPageDocumentsLimit
	}
	opts.SetLimit(length)
	opts.SetAllowDiskUse(true)
	//opts.SetBatchSize(int32(length))
	cursor, err := coll.Find(ctx, match, &opts)
	return cursor, err
}

// GetFindOnlyMatchResults no sort, skip, limit, just match
func GetFindOnlyMatchResults(ctx context.Context, coll *mongo.Collection, match bson.M) (*mongo.Cursor, error) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	opts := options.FindOptions{}
	opts.SetAllowDiskUse(true)
	cursor, err := coll.Find(ctx, match, &opts)
	return cursor, err
}

// GetCount returns count of query result
func GetCount(ctx context.Context, collection *mongo.Collection, match bson.M) (int64, error) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	count, err := collection.CountDocuments(ctx, match)
	return count, err
}

// Insert returns count of query result
func Insert(ctx context.Context, collection *mongo.Collection, data interface{}) error {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	_, err := collection.InsertOne(ctx, data)
	return err
}

// ---------------- initialize ----------

// checkEnvVar helper function to check if required environment variable is set
func checkEnvVar(env string) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	_, present := os.LookupEnv(env)
	if present == false {
		log.Fatalf("Please set %s environment variable. Exiting...", env)
	}
}

// initializeMongoDatabase returns mongo client
func initializeMongoDatabase(mongoUri string, mongoDbName string, mongoConnTimeout int) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var err error
	var client *mongo.Client
	//opts.SetMaxPoolSize(100)
	//opts.SetConnectTimeout(time.Duration(ConnectionTimeout) * time.Second)
	//opts.SetMaxConnIdleTime(time.Microsecond * 100000)
	Timeout = time.Duration(mongoConnTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	if client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoUri).SetConnectTimeout(Timeout)); err != nil {
		log.Fatal(err)
	}
	if err := client.Ping(context.Background(), nil); err != nil {
		log.Fatal(err)
	}
	mongoDatabase = client.Database(mongoDbName)
}

// InitializeMongo setup MongoDB connection environments
func InitializeMongo(envFile string, mongoConnTimeout int) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	err := godotenv.Load(envFile)
	if err != nil {
		log.Fatalf("Error loading %s env file", envFile)
	}
	checkEnvVar("MONGOURI")
	checkEnvVar("MONGO_DATABASE")
	mongoUri := os.Getenv("MONGOURI")
	mongoDbName := os.Getenv("MONGO_DATABASE")

	// Mongo logs
	utils.InfoLogV1("MONGO_URI: %s", mongoUri)
	utils.InfoLogV1("MONGO_DATABASE: %s", mongoDbName)
	utils.InfoLogV1("MONGO CONNECTION TIMEOUT: %d", mongoConnTimeout)

	initializeMongoDatabase(mongoUri, mongoDbName, mongoConnTimeout)
}
