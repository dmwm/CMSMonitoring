package configs

import (
	"log"
	"os"
)

func InitialChecks() {
	CheckEnvVar("MONGOURI")
	CheckEnvVar("MONGO_DATABASE")
}

// CheckEnvVar helper function to check if required environment variable is set
func CheckEnvVar(env string) {
	_, present := os.LookupEnv(env)
	if present == false {
		log.Fatalf("Please set %s environment variable. Exiting...", env)
	}
}

// EnvMongoURI returns "MONGOURI" environment variable
func EnvMongoURI() string {
	mongouri := os.Getenv("MONGOURI")
	return mongouri
}

// EnvMongoDB returns MongoDB connection uri "MONGO_DATABASE" environment variable
func EnvMongoDB() string {
	return os.Getenv("MONGO_DATABASE")
}
