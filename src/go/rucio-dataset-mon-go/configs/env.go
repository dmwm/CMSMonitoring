package configs

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

var EnvFile string

func InitialChecks() {
	err := godotenv.Load(EnvFile)
	if err != nil {
		log.Fatalf("Error loading %s env file", EnvFile)
	}
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
