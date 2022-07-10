package routes

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
func MainRouter() http.Handler {

	responseBodyTimeout := gin.H{
		"code":    http.StatusRequestTimeout,
		"message": "request timeout, response is sent from middleware"}

	e := gin.New()
	e.Use(gin.Recovery())
	e.Use(timeout.TimeoutHandler(300*time.Second, http.StatusRequestTimeout, responseBodyTimeout))

	// REST
	e.Use(utils.MiddlewareReqHandler())
	e.POST("/api/datasets", controllers.GetDatasets())
	e.POST("/api/rse-details", controllers.GetDetailedDs())
	e.POST("/api/rse-detail", controllers.GetSingleDetailedDs())
	e.POST("/api/short-url", controllers.GetShortUrlParam())
	e.GET("/short-url/:id", controllers.GetIndexPageFromShortUrlId())
	e.GET("/serverinfo", controllers.GetServerInfo)

	// Static
	e.LoadHTMLGlob("static/*.html")
	e.Static("/static/img", "./static/img")
	e.GET("/", controllers.GetIndexPage)
	e.GET("/rse-details", controllers.GetDetailsPage)

	//
	return e
}
