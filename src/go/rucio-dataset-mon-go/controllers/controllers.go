package controllers

import (
	"context"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
	mymongo "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/short_url"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// All important variables in controllers are gathered here
var (
	// GitVersion git version comes from Makefile
	GitVersion string

	// ServerInfo defines server info comes from Makefile
	ServerInfo string

	// Verbose defines verbosity, user given parameter, default 0: INFO logs
	Verbose int

	// datasetsCollectionName collection name that stores the datasets
	datasetsCollectionName = "datasets"

	// datasetsCollectionName collection name that stores the detailed_datasets
	detailedDsCollectionName = "detailed_datasets"

	// sourceTimeCollectionName collection name that stores the time-stamp of data that used in Spark jobs
	sourceTimeCollectionName = "source_timestamp"

	// shortUrlCollectionName short_url Mongo collection name
	shortUrlCollectionName = "short_url"

	// prodAccounts production accounts that lock files
	prodAccounts = []string{"transfer_ops", "wma_prod", "wmcore_output", "wmcore_transferor", "crab_tape_recall", "sync"}
)

// GetServerInfo provides basic functionality of status response
func GetServerInfo(c *gin.Context) {
	verboseControllerInitLog(c)
	c.JSON(http.StatusOK,
		models.ServerInfoResp{
			ServiceVersion: GitVersion,
			Server:         ServerInfo,
		})
}

// GetIndexPage serves index.html page
func GetIndexPage(c *gin.Context) {
	start := time.Now()
	verboseControllerInitLog(c)
	ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
	defer cancel()

	// get source data creation time
	dataTimestamp := getDataSourceTimestamp(ctx, c, sourceTimeCollectionName)
	c.HTML(
		http.StatusOK,
		"index.html",
		gin.H{
			"title":          "Home Page",
			"is_short_url":   "false", // Should be set as string since it will be required by JS in index.html
			"data_timestamp": dataTimestamp.CreatedAt,
		},
	)
	verboseControllerOutLog(start, "GetIndexPage", nil, dataTimestamp)
	return
}

// GetDetailsPage serves details.html page
func GetDetailsPage(c *gin.Context) {
	verboseControllerInitLog(c)
	start := time.Now()
	c.HTML(
		http.StatusOK,
		"details.html",
		gin.H{
			"title": "Detailed Datasets Page",
		},
	)
	verboseControllerOutLog(start, "GetDetailsPage", nil, nil)
}

// GetDatasets datasets controller
func GetDatasets() gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.DataTableSearchBuilderRequest to the controller initializer and use same type in casting
		ctx, cancel, start, req := initializeController(c, models.DataTableSearchBuilderRequest{})
		defer cancel()
		detailedDatasetsResp := getDatasetResults(ctx, c, datasetsCollectionName, req.(models.DataTableSearchBuilderRequest))
		c.JSON(http.StatusOK,
			detailedDatasetsResp,
		)
		verboseControllerOutLog(start, "GetDetailedDs", req, detailedDatasetsResp)
		return
	}
}

// GetDetailedDs controller that returns datasets according to DataTable request json
func GetDetailedDs() gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.DataTableCustomRequest to the controller initializer and use same type in casting
		ctx, cancel, start, req := initializeController(c, models.DataTableCustomRequest{})
		defer cancel()
		detailedDatasetsResp := getDetailedDsResults(ctx, c, req.(models.DataTableCustomRequest))
		c.JSON(http.StatusOK,
			detailedDatasetsResp,
		)
		verboseControllerOutLog(start, "GetDetailedDs", req, detailedDatasetsResp)
		return
	}
}

// GetShortUrlParam controller that returns short url parameter which is md5 hash of the datatables request
func GetShortUrlParam() gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.ShortUrlRequest to the controller initializer and use same type in casting
		ctx, cancel, start, req := initializeController(c, models.ShortUrlRequest{})
		defer cancel()
		requestHash := short_url.GetShortUrl(ctx, c, shortUrlCollectionName, req.(models.ShortUrlRequest))
		c.JSON(http.StatusOK,
			requestHash,
		)
		verboseControllerOutLog(start, "GetShortUrlParam", req, requestHash)
		return
	}
}

// GetIndexPageFromShortUrlId controller that returns page from short url hash id
func GetIndexPageFromShortUrlId() gin.HandlerFunc {
	return func(c *gin.Context) {
		verboseControllerInitLog(c)
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()

		hashId := c.Param("id")
		utils.InfoLogV1("[INFO] Hash Id: %s", hashId)
		shortUrlObj := short_url.GetRequestFromShortUrl(ctx, c, shortUrlCollectionName, hashId)

		dataTimestamp := getDataSourceTimestamp(ctx, c, sourceTimeCollectionName)
		c.HTML(
			http.StatusOK,
			"index.html",
			gin.H{
				"title":                "Home Page",
				"is_short_url":         "true",
				"dt_request_short_url": shortUrlObj.Request,
				"dt_saved_state":       shortUrlObj.SavedState,
				"data_timestamp":       dataTimestamp.CreatedAt,
			},
		)
		verboseControllerOutLog(start, "GetIndexPageFromShortUrlId", shortUrlObj, hashId)
		return
	}
}

// GetSingleDetailedDs controller that returns detailed dataset in TAPE or DISK
func GetSingleDetailedDs() gin.HandlerFunc {
	return func(c *gin.Context) {
		verboseControllerInitLog(c)
		ctx, cancel, start, req := initializeController(c, models.SingleDetailedDatasetsRequest{})
		defer cancel()
		detailedRows := getSingleDatasetResults(ctx, c, req.(models.SingleDetailedDatasetsRequest))
		c.HTML(http.StatusOK,
			"rse_detail_table.html",
			gin.H{"data": detailedRows},
		)
		verboseControllerOutLog(start, "GetSingleDetailedDs", req, detailedRows)
		return
	}
}
