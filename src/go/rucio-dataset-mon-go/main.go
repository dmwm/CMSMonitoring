package main

import (
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/configs"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/controllers"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/routes"
	"github.com/gin-gonic/gin"
	"log"
	"os"
)

// main
func main() {
	// Check GO_ENVS_SECRET_PATH is set
	envVal, present := os.LookupEnv("GO_ENVS_SECRET_PATH")
	if present == true {
		log.Fatal("Please set $GO_ENVS_SECRET_PATH environment variable. Exiting...")
	} else {
		log.Printf("GO_ENVS_SECRET_PATH is : %t\n", envVal)
	}

	router := gin.Default()

	// connect to database
	configs.ConnectDB()

	// router middleware HTTP settings for CORS
	router.Use(controllers.MiddlewareReqHandler())

	// routes
	routes.DatasetRoute(router)

	if err := router.Run("localhost:8080"); err != nil {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Fatal(err)
	}
}
