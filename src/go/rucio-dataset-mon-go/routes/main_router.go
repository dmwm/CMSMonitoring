package routes

import (
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/controllers"
	"github.com/gin-gonic/gin"
	"net/http"
)

// MainRouter main request router
func MainRouter() http.Handler {
	e := gin.New()
	e.Use(gin.Recovery())

	// REST
	e.Use(controllers.MiddlewareReqHandler())
	e.POST("/api/datasets", controllers.GetDatasets())

	// Static
	e.LoadHTMLGlob("static/*")
	e.GET("/", controllers.GetIndexPage)
	e.GET("/serverinfo", controllers.GetServerInfo)

	return e
}
