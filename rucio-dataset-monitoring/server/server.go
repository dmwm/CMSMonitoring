package server

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>
//
// Inspired by https://github.com/dmwm/dbs2go/blob/master/web/server.go

import (
	"fmt"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/controllers"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/mongo"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/utils"
	"github.com/gin-gonic/gin"
	timeout "github.com/s-wijaya/gin-timeout"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"time"
)

// GitVersion git version comes from Makefile
var GitVersion string

// ServiceInfo defines server info comes from Makefile
var ServiceInfo string

// middlewareReqHandler handles CORS and HTTP request settings for the context router
func middlewareReqHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		//c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// middlewareLogFormatter custom log formatter
var middlewareLogFormatter = func(param gin.LogFormatterParams) string {
	return fmt.Sprintf("[%s] - %s [%s %d %s %s %s %d] [%s] [%s]\n",
		param.TimeStamp.Format(time.RFC3339),
		param.ClientIP,
		param.Method,
		param.StatusCode,
		param.Path,
		param.Request.Proto,
		param.Latency,
		param.BodySize,
		param.Request.UserAgent(),
		param.ErrorMessage,
	)
}

// MainRouter main request router
func MainRouter(mongoColNames *MongoCollectionNames) http.Handler {
	responseBodyTimeout := gin.H{"code": http.StatusRequestTimeout, "message": "request timeout, response is sent from middleware"}

	gin.DisableConsoleColor()
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(timeout.TimeoutHandler(300*time.Second, http.StatusRequestTimeout, responseBodyTimeout))
	engine.Use(middlewareReqHandler())
	engine.Use(gin.LoggerWithFormatter(middlewareLogFormatter))
	engine.LoadHTMLGlob("static/templates/*.tmpl")

	// -------------------------------- Root ------------------------------------------------------
	engine.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title": "Main website",
		})
	})

	// ------------------------------- Config.BaseEndpoint group-----------------------------------
	e := engine.Group("/" + Config.BaseEndpoint)
	{
		e.StaticFS("/static", http.Dir("./static"))

		e.GET("/serverinfo", controllers.GetServiceInfo(GitVersion, ServiceInfo))

		// Page APIs
		e.POST("/api/main-datasets", controllers.GetMainDatasets(mongoColNames.MainDatasets))
		e.POST("/api/detailed-datasets", controllers.GetDetailedDatasets(mongoColNames.DetailedDatasets))
		e.POST("/api/main-dataset-details", controllers.GetMainDatasetDetails(mongoColNames.DetailedDatasets))

		e.POST("/api/short-url", controllers.GetShortUrlParam(mongoColNames.ShortUrl))

		// Main datasets page
		e.GET("/main", controllers.GetMainDatasetsPage(
			mongoColNames.DatasourceTimestamp,
			"../"+Config.BaseEndpoint+"/api/main-datasets",
			"../"+Config.BaseEndpoint+"/api/short-url",
			"../"+Config.BaseEndpoint+"/api/main-dataset-details",
			Config.BaseEndpoint,
		))

		// Detailed datasets page
		e.GET("/detailed", controllers.GetDetailedDatasetsPage(
			mongoColNames.DatasourceTimestamp,
			"../"+Config.BaseEndpoint+"/api/detailed-datasets",
			"../"+Config.BaseEndpoint+"/api/short-url",
			Config.BaseEndpoint,
		))

		// Short url result page
		e.GET("/short-url/:id", controllers.GetIndexPageFromShortUrlId(
			mongoColNames.ShortUrl,
			mongoColNames.DatasourceTimestamp,
			"../"+Config.BaseEndpoint+"/api/main-datasets",
			"../"+Config.BaseEndpoint+"/api/short-url",
			"../"+Config.BaseEndpoint+"/api/main-dataset-details",
			"../"+Config.BaseEndpoint+"/api/detailed-datasets",
			Config.BaseEndpoint,
		))
	}

	return engine
}

// Serve run service
func Serve(configFile string) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var g errgroup.Group
	err := ParseConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}
	mongo.InitializeMongo(Config.EnvFile, Config.MongoConnectionTimeout)

	utils.Verbose = Config.Verbose

	mongoCollectionNames := Config.CollectionNames
	mainServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", Config.Port),
		Handler:      MainRouter(&mongoCollectionNames),
		ReadTimeout:  time.Duration(Config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(Config.WriteTimeout) * time.Second,
	}

	utils.InfoLogV0("rucio dataset monitoring service is starting with base endpoint: %s", "/"+Config.BaseEndpoint)
	utils.InfoLogV0("rucio dataset monitoring service is starting with config: %s", Config.String())
	g.Go(func() error {
		return mainServer.ListenAndServe()
	})

	if err := g.Wait(); err != nil {
		log.Printf("[ERROR] server failed %s", err)
	}
}
