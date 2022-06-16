package configs

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"strings"
)

// checkEnvVar helper function to check if required environment variable is set
func checkEnvVar(env string) {
	_, present := os.LookupEnv(env)
	if present == false {
		log.Fatalf("Please set %s environment variable. Exiting...", env)
	}
}

var EnvSecretFilesPath = os.Getenv("GO_ENVS_SECRET_PATH")

// EnvMongoURI returns "MONGOURI" environment variable
func EnvMongoURI() string {
	// Check required environment variables are set (see configs/*.go files)
	checkEnvVar("GO_ENVS_SECRET_PATH")
	checkEnvVar("MONGO_ROOT_USERNAME")
	checkEnvVar("MONGO_ROOT_PASSWORD")

	if err := godotenv.Load(EnvSecretFilesPath); err != nil {
		log.Fatal("Error loading $GO_ENVS_SECRET_PATH file")
	}
	// replace  mongodb://[USERNAME]:[PASSWORD]@host:port/[DATABASE]?retryWrites=true&w=majority with credentials
	mongouri := os.Getenv("MONGOURI")
	mongouri = strings.Replace(mongouri, "[USERNAME]", os.Getenv("MONGO_ROOT_USERNAME"), -1)
	mongouri = strings.Replace(mongouri, "[PASSWORD]", os.Getenv("MONGO_ROOT_PASSWORD"), -1)
	mongouri = strings.Replace(mongouri, "[DATABASE]", os.Getenv("MONGO_ADMIN_DB"), -1)
	log.Println("Mongo uri")
	log.Println(mongouri)
	return mongouri
}

// EnvMongoDB returns MongoDB connection uri "MONGO_DATABASE" environment variable
func EnvMongoDB() string {
	if err := godotenv.Load(EnvSecretFilesPath); err != nil {
		log.Fatal("Error loading $GO_ENVS_SECRET_PATH file")
	}
	return os.Getenv("MONGO_DATABASE")
}

// EnvConnTimeout returns "MONGO_CONNECT_TIMEOUT" environment variable
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

// GetEnvVar generic environment variable getter from secret file
func GetEnvVar(envVar string) string {
	if err := godotenv.Load(EnvSecretFilesPath); err != nil {
		log.Fatal("Error loading $GO_ENVS_SECRET_PATH file")
	}
	return os.Getenv(envVar)
}
