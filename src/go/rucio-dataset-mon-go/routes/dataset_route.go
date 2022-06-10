package routes

import (
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/controllers"
	"github.com/gin-gonic/gin"
)

// DatasetRoute route for "/datasets"
func DatasetRoute(router *gin.Engine) {
	router.POST("/datasets", controllers.GetDatasets())
}
