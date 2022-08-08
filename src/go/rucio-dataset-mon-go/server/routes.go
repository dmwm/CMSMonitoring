package server

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/controllers"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
	timeout "github.com/s-wijaya/gin-timeout"
	"net/http"
	"time"
)

// MainRouter main request router
func MainRouter(mongoColNames *MongoCollectionNames) http.Handler {
	responseBodyTimeout := gin.H{
		"code":    http.StatusRequestTimeout,
		"message": "request timeout, response is sent from middleware"}

	e := gin.New()
	e.Use(gin.Recovery())
	e.Use(timeout.TimeoutHandler(300*time.Second, http.StatusRequestTimeout, responseBodyTimeout))

	// REST
	e.Use(utils.MiddlewareReqHandler())
	e.POST("/api/datasets", controllers.GetDatasets(mongoColNames.Datasets))
	e.POST("/api/rse-details", controllers.GetDetailedDs(mongoColNames.DetailedDatasets, &Config.ProdLockAccounts))
	e.POST("/api/rse-detail", controllers.GetSingleDetailedDs(mongoColNames.DetailedDatasets))
	e.POST("/api/short-url", controllers.GetShortUrlParam(mongoColNames.ShortUrl))
	e.GET("/short-url/:id", controllers.GetIndexPageFromShortUrlId(mongoColNames.ShortUrl, mongoColNames.DatasourceTimestamp))
	e.GET("/serverinfo", controllers.GetServiceInfo(GitVersion, ServiceInfo))

	// Static
	e.LoadHTMLGlob("static/templates/*.tmpl")
	e.Static("/static/img", "./static/img")
	e.GET("/", controllers.GetIndexPage(mongoColNames.DatasourceTimestamp))
	e.GET("/rse-details", controllers.GetDetailsPage)

	//
	return e
}
