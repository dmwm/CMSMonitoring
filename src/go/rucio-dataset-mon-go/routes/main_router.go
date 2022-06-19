package routes

import (
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/controllers"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

// MainRouter main request router
func MainRouter() http.Handler {
	e := gin.New()
	e.Use(gin.Recovery())

	// REST
	e.Use(utils.MiddlewareReqHandler())
	e.POST("/api/datasets", controllers.GetDatasets())
	e.Static("/static/img", "./static/img")

	// Static
	e.LoadHTMLGlob("static/index.html")
	e.GET("/", controllers.GetIndexPage)
	e.GET("/serverinfo", controllers.GetServerInfo)

	return e
}
