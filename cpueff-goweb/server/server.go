package server

// Copyright (c) 2023 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>
//
// Inspired by https://github.com/dmwm/dbs2go/blob/master/web/server.go

import (
	"context"
	"fmt"
	"github.com/dmwm/CMSMonitoring/cpueff-goweb/controllers"
	"github.com/dmwm/CMSMonitoring/cpueff-goweb/models"
	"github.com/dmwm/CMSMonitoring/cpueff-goweb/mongo"
	"github.com/dmwm/CMSMonitoring/cpueff-goweb/utils"
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
	return fmt.Sprintf("[%s] [MLWR] - %s [%s %d %s %s %s %d] [%s] [%s]\n",
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
func MainRouter(mongoColNames *models.MongoCollectionNames) http.Handler {
	responseBodyTimeout := gin.H{"code": http.StatusRequestTimeout, "message": "request timeout, response is sent from middleware"}

	gin.DisableConsoleColor()
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(timeout.TimeoutHandler(300*time.Second, http.StatusRequestTimeout, responseBodyTimeout))
	engine.Use(middlewareReqHandler())
	engine.Use(gin.LoggerWithFormatter(middlewareLogFormatter))
	engine.LoadHTMLGlob("static/templates/*.tmpl")

	// ------------------------------- /Config.BaseEndpoint/ group -----------------------------------

	// Give root path as / and provide ingress/auth-poxy redirection same in Config.BaseEndpoint
	e := engine.Group("/")
	{
		e.GET("/", func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(Config.MongoConnectionTimeout)*time.Second)
			defer cancel()
			dataTimestamp := controllers.GetDataSourceTimestamp(ctx, c, Config.CollectionNames.DatasourceTimestamp)
			tierCpuEffs := controllers.GetTierEfficiencies(ctx, c, Config.CollectionNames.CondorTiers)
			c.HTML(http.StatusOK,
				"index.tmpl",
				gin.H{
					"title":                  "Main website",
					"govar_BASE_EP":          Config.BaseEndpoint,
					"data_tier_efficiencies": tierCpuEffs,
					"data_source_time":       dataTimestamp.StartDate + " - " + dataTimestamp.EndDate,
				},
			)
		})

		e.StaticFS("/static", http.Dir("./static"))
		e.GET("/serverinfo", controllers.ServiceCtrl(GitVersion, ServiceInfo))
		e.POST("/api/short-url", controllers.ShortUrlParamCtrl(mongoColNames.ShortUrl))

		// ---------------------------------------------------------- Condor
		// Page APIs
		e.POST("/api/condor-main", controllers.CondorMainCtrl(Config))
		e.POST("/api/condor-detailed", controllers.CondorDetailedCtrl(Config))
		e.POST("/api/condor-main-each-detailed", controllers.CondorMainEachDetailedRowCtrl(Config))
		// Condor Main cpueff page
		e.GET("/condor-main", controllers.GetCondorMainCpuEfficiencyPage(
			mongoColNames.DatasourceTimestamp,
			"/"+Config.BaseEndpoint+"/api/condor-main",
			"/"+Config.BaseEndpoint+"/api/condor-main-each-detailed",
			"/"+Config.BaseEndpoint+"/api/short-url",
			Config.BaseEndpoint,
		))
		// Condor Detailed cpueff page
		e.GET("/condor-detailed", controllers.GetCondorDetailedCpuEfficiencyPage(
			mongoColNames.DatasourceTimestamp,
			"/"+Config.BaseEndpoint+"/api/condor-detailed",
			"/"+Config.BaseEndpoint+"/api/short-url",
			Config.BaseEndpoint,
		))
		// ---------------------------------------------------------- Stepchain
		e.POST("/api/sc-task", controllers.ScTaskCtrl(Config))
		e.POST("/api/sc-task-cmsrun-jobtype", controllers.ScTaskCmsrunJobtypeCtrl(Config))
		e.POST("/api/sc-task-cmsrun-jobtype-site", controllers.ScTaskCmsrunJobtypeSiteCtrl(Config))

		e.POST("/api/sc-task-each-detailed", controllers.ScEachSiteDetailCtrl(Config))

		// Stepchain Main Task cpueff page
		e.GET("/sc-main", controllers.GetScTaskPage(
			mongoColNames.DatasourceTimestamp,
			"/"+Config.BaseEndpoint+"/api/sc-task",
			"/"+Config.BaseEndpoint+"/api/sc-task-each-detailed",
			"/"+Config.BaseEndpoint+"/api/short-url",
			Config.BaseEndpoint,
		))

		// Stepchain Detail TaskCmsrunJobtype cpueff page
		e.GET("/sc-detail-task-cmsrun-jobtype", controllers.GetScTaskCmsrunJobtypePage(
			mongoColNames.DatasourceTimestamp,
			"/"+Config.BaseEndpoint+"/api/sc-task-cmsrun-jobtype",
			"/"+Config.BaseEndpoint+"/api/sc-task-each-detailed",
			"/"+Config.BaseEndpoint+"/api/short-url",
			Config.BaseEndpoint,
		))

		// ---------------------------------------------------------- Short URL

		// Short url result page
		e.GET("/short-url/:id", controllers.GetIndexPageFromShortUrlId(
			mongoColNames.ShortUrl,
			mongoColNames.DatasourceTimestamp,
			"/"+Config.BaseEndpoint+"/api/condor-main",
			"/"+Config.BaseEndpoint+"/api/condor-detailed",
			"/"+Config.BaseEndpoint+"/api/condor-main-each-detailed",
			"/"+Config.BaseEndpoint+"/api/sc-task",
			"/"+Config.BaseEndpoint+"/api/sc-task-cmsrun-jobtype",
			"/"+Config.BaseEndpoint+"/api/sc-task-each-detailed",
			"/"+Config.BaseEndpoint+"/api/short-url",
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

	serveAddress := fmt.Sprintf(":%d", Config.Port)
	if Config.IsTest {
		// To skip MacOS port incoming request permission pop-up
		serveAddress = fmt.Sprintf("localhost:%d", Config.Port)
	}

	mainServer := &http.Server{
		Addr:         serveAddress,
		Handler:      MainRouter(&mongoCollectionNames),
		ReadTimeout:  time.Duration(Config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(Config.WriteTimeout) * time.Second,
	}

	utils.InfoLogV0("cpu efficiency monitoring service is starting with base endpoint: %s", "/"+Config.BaseEndpoint)
	utils.InfoLogV0("cpu efficiency monitoring service is starting with config: %s", Config.String())
	g.Go(func() error {
		return mainServer.ListenAndServe()
	})

	if err := g.Wait(); err != nil {
		log.Printf("[ERROR] server failed %s", err)
	}
}
