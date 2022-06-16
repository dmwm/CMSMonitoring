package main

import (
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/configs"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/controllers"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/routes"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"time"
)

// Reference: https://github.com/gin-gonic/gin/issues/346

var (
	g errgroup.Group
)

func mainRouter() http.Handler {
	e := gin.New()
	e.Use(gin.Recovery())

	// REST
	e.Use(controllers.MiddlewareReqHandler())
	routes.DatasetRoute(e)

	// Static
	e.LoadHTMLGlob("static/*")
	e.GET("/", routes.IndexRoute)

	return e
}

func main() {
	// connect to database
	configs.ConnectDB()

	mainServer := &http.Server{
		Addr:         ":8080",
		Handler:      mainRouter(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	g.Go(func() error {
		return mainServer.ListenAndServe()
	})

	if err := g.Wait(); err != nil {
		log.Printf("[ERROR] SERVER FAILED %s", err)
	}
}
