package routes

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
	e.POST("/api/details", controllers.GetDetailedDs())
	e.Static("/static/img", "./static/img")

	// Static
	e.LoadHTMLGlob("static/*.html")
	e.GET("/", controllers.GetIndexPage)
	e.GET("/details", controllers.GetDetailsPage)
	e.GET("/serverinfo", controllers.GetServerInfo)

	return e
}
