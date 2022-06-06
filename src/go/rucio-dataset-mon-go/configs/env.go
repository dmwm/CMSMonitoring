package configs

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
)

var EnvSecretFilesPath = os.Getenv("GO_ENVS_SECRET_PATH")

// EnvMongoURI returns "MONGOURI" environment variable which is defined in ~/.env file
func EnvMongoURI() string {
	if err := godotenv.Load(EnvSecretFilesPath); err != nil {
		log.Fatal("Error loading $GO_ENVS_SECRET_PATH file")
	}
	return os.Getenv("MONGOURI")
}

// EnvMongoDB returns MongoDB connection uri "MONGO_DATABASE" environment variable which is defined in ~/.env file
func EnvMongoDB() string {
	if err := godotenv.Load(EnvSecretFilesPath); err != nil {
		log.Fatal("Error loading $GO_ENVS_SECRET_PATH file")
	}
	return os.Getenv("MONGO_DATABASE")
}

// EnvConnTimeout returns "MONGO_CONNECT_TIMEOUT" environment variable which is defined in ~/.env file
func EnvConnTimeout() int {
	if err := godotenv.Load(EnvSecretFilesPath); err != nil {
		log.Fatal("Error loading $GO_ENVS_SECRET_PATH file")
	}
	connTimeout, err := strconv.Atoi(os.Getenv("MONGO_CONNECT_TIMEOUT"))
	if err != nil {
		log.Fatal("Error getting MONGO_CONNECT_TIMEOUT env variable")
	}
	return connTimeout
}

// GetEnvVar generic environment variable getter
func GetEnvVar(envVar string) string {
	if err := godotenv.Load(EnvSecretFilesPath); err != nil {
		log.Fatal("Error loading $GO_ENVS_SECRET_PATH file")
	}
	return os.Getenv(envVar)
}
