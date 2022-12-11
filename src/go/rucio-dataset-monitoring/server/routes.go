package server

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"fmt"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/controllers"
	"github.com/gin-gonic/gin"
	timeout "github.com/s-wijaya/gin-timeout"
	"net/http"
	"time"
)

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
	return fmt.Sprintf("[%s] - %s [%s %s %s %d %s %d] [%s] [%s]\n",
		param.TimeStamp.Format(time.RFC3339),
		param.ClientIP,
		param.Method,
		param.Path,
		param.Request.Proto,
		param.StatusCode,
		param.Latency,
		param.BodySize,
		param.Request.UserAgent(),
		param.ErrorMessage,
	)
}

// MainRouter main request router
func MainRouter(mongoColNames *MongoCollectionNames) http.Handler {
	responseBodyTimeout := gin.H{
		"code":    http.StatusRequestTimeout,
		"message": "request timeout, response is sent from middleware"}

	gin.DisableConsoleColor()
	e := gin.New()
	e.Use(gin.Recovery())
	e.Use(timeout.TimeoutHandler(300*time.Second, http.StatusRequestTimeout, responseBodyTimeout))
	e.Use(middlewareReqHandler())
	e.Use(gin.LoggerWithFormatter(middlewareLogFormatter))

	// Static
	e.LoadHTMLGlob("static/templates/*.tmpl")
	e.Static("/static/img", "./static/img")

	// REST
	e.POST("/api/datasets", controllers.GetDatasets(mongoColNames.Datasets))
	e.POST("/api/rse-details", controllers.GetDetailedDs(mongoColNames.DetailedDatasets, &Config.ProdLockAccounts))
	e.POST("/api/rse-detail", controllers.GetSingleDetailedDs(mongoColNames.DetailedDatasets))
	e.POST("/api/short-url", controllers.GetShortUrlParam(mongoColNames.ShortUrl))
	e.GET("/short-url/:id", controllers.GetIndexPageFromShortUrlId(mongoColNames.ShortUrl, mongoColNames.DatasourceTimestamp))
	e.GET("/serverinfo", controllers.GetServiceInfo(GitVersion, ServiceInfo))

	// Pages
	e.GET("/", controllers.GetIndexPage(mongoColNames.DatasourceTimestamp))
	e.GET("/rse-details", controllers.GetDetailsPage)

	//
	return e
}
