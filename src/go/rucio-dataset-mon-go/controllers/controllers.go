package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	"encoding/json"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/data_timestamp"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/datasets"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/detailed_datasets"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/responses"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/short_url"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"log"
	"net/http"
	"time"
)

var (
	// ServerInfo defines server info comes from Makefile
	ServerInfo string
	// GitVersion git version comes from Makefile
	GitVersion string
	Verbose    int
)

func verboseControllerInitLog(c *gin.Context) {
	if Verbose > 0 {
		log.Printf("[INFO] Incoming request to: %s", c.FullPath())
	}
}

func controllerInitialize(c *gin.Context) (context.Context, context.CancelFunc, time.Time, models.DataTableCustomRequest) {
	verboseControllerInitLog(c)
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), mongo.Timeout)

	// Get request json with validation
	r := models.DataTableCustomRequest{}
	if err := c.ShouldBindBodyWith(&r, binding.JSON); err != nil {
		tempReqBody, exists := c.Get(gin.BodyBytesKey)
		if !exists {
			utils.ErrorResponse(c, "Bad request", err, "Request body does not exist in Gin context")
		} else {
			utils.ErrorResponse(c, "Bad request", err, string(tempReqBody.([]byte)))
		}
	}
	return ctx, cancel, start, r
}

func verboseControllerOutLog(start time.Time, name string, req any, data any) {
	if Verbose > 0 {
		elapsed := time.Since(start)
		req, err := json.Marshal(req)
		if err != nil {
			log.Printf("[ERROR] ------ Cannot marshall request, err:%s", err)
		} else {
			r := string(req)
			if Verbose >= 2 {
				// Response returns all query results, its verbosity should be at least 2
				data, err1 := json.Marshal(data)
				if err1 != nil {
					log.Printf("[ERROR] ------ Cannot marshall additional verbose log data, err:%s", err1)
				} else {
					d := string(data)
					log.Printf("[DEBUG] ------\n -Query time [%s] : %s\n\n -Request body: %s\n\n -Response: %s\n\n", name, elapsed, r, d)
				}
			} else {
				log.Printf("[INFO] ------\n -Query time [%s] : %s\n\n -Request body: %s\n\n -Response: %#v\n\n", name, elapsed, req, nil)
			}
		}
	}
}

// GetServerInfo provides basic functionality of status response
func GetServerInfo(c *gin.Context) {
	verboseControllerInitLog(c)
	c.JSON(http.StatusOK,
		responses.ServerInfo{
			ServiceVersion: GitVersion,
			Server:         ServerInfo,
		})
}

// GetDatasets controller that returns datasets according to DataTable request json
func GetDatasets() gin.HandlerFunc {
	return func(c *gin.Context) {
		verboseControllerInitLog(c)
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), mongo.Timeout)
		defer cancel()

		// Get request json with validation
		var datasetsResp any

		// 1- Try custom request
		rCustom := models.DataTableCustomRequest{}
		errCustom := c.ShouldBindBodyWith(&rCustom, binding.JSON)
		if errCustom == nil {
			// If error is nil, it means that this is the request that binds with rCustom, so process it
			datasetsResp = datasets.GetResults(ctx, c, rCustom)
			c.JSON(http.StatusOK,
				datasetsResp,
			)
			verboseControllerOutLog(start, "GetDatasets", rCustom, datasetsResp)
			return
		} else {
			// This is not the request that binds with rCustom
			tempReqBody, exists := c.Get(gin.BodyBytesKey)
			if !exists {
				log.Printf("[WARN] Request body for logging purpose does not exist")
			} else {
				log.Printf("[WARN] Request could not bind with custom DT request %#v", string(tempReqBody.([]byte)))
			}
		}

		// 2- Try plain request which is fired in first opening of the page
		rPlain := models.DataTableRequest{}
		errPlain := c.ShouldBindBodyWith(&rPlain, binding.JSON)
		if errPlain == nil {
			// If error is nil, it means that this is the request that binds with rPlain, so process it
			datasetsResp = datasets.GetResults(ctx, c, rCustom)
			c.JSON(http.StatusOK,
				datasetsResp,
			)
			verboseControllerOutLog(start, "GetDatasets", rPlain, datasetsResp)
			return
		} else {
			tempReqBody, exists := c.Get(gin.BodyBytesKey)
			if !exists {
				log.Printf("[WARN] Request body for logging purpose does not exist")
			} else {
				log.Printf("[ERROR] Request could not bind with custom DT request %#v", string(tempReqBody.([]byte)))
			}
			// Since this is the last try, after this point, it is certain that incoming request does not bind any request struct
			utils.ErrorResponse(c, "Bad request", errCustom, string(tempReqBody.([]byte)))
		}
		return
	}
}

// GetDetailedDs controller that returns datasets according to DataTable request json
func GetDetailedDs() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel, start, req := controllerInitialize(c)
		defer cancel()
		detailedDatasetsResp := detailed_datasets.GetResults(ctx, c, req)
		c.JSON(http.StatusOK,
			detailedDatasetsResp,
		)
		verboseControllerOutLog(start, "GetDetailedDs", req, detailedDatasetsResp)
		return
	}
}

// GetShortUrlParam controller that returns short url param which is md5 hash of the datatables request
func GetShortUrlParam() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		verboseControllerInitLog(c)
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), mongo.Timeout)

		// Get request json with validation
		req := models.ShortUrlRequest{}
		if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
			tempReqBody, exists := c.Get(gin.BodyBytesKey)
			if !exists {
				utils.ErrorResponse(c, "Bad request", err, "Request body for logging purpose does not exist")
			} else {
				utils.ErrorResponse(c, "Bad request", err, string(tempReqBody.([]byte)))
			}
		}
		defer cancel()
		requestHash := short_url.GetShortUrl(ctx, c, req)
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
		ctx, cancel := context.WithTimeout(context.Background(), mongo.Timeout)
		defer cancel()

		hashId := c.Param("id")
		log.Printf("[INFO] Hash Id: %s", hashId)
		shortUrlObj := short_url.GetRequestFromShortUrl(ctx, c, hashId)

		dataTimestamp := data_timestamp.GetDataSourceTimestamp(ctx, c)
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
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), mongo.Timeout)
		defer cancel()

		r := models.SingleDetailedDatasetsRequest{}
		if err := c.ShouldBindBodyWith(&r, binding.JSON); err != nil {
			x, exists := c.Get(gin.BodyBytesKey)
			if !exists {
				utils.ErrorResponse(c, "Bad request", err, "Request body for logging purpose does not exist")
			} else {
				utils.ErrorResponse(c, "Bad request", err, string(x.([]byte)))
			}
			return
		}
		detailedRows := detailed_datasets.GetSingleDataset(ctx, c, r)
		c.HTML(http.StatusOK,
			"rse_detail_table.html",
			gin.H{"data": detailedRows},
		)
		verboseControllerOutLog(start, "GetSingleDetailedDs", r, detailedRows)
		return
	}
}

// GetIndexPage serves index.html page
func GetIndexPage(c *gin.Context) {
	start := time.Now()
	verboseControllerInitLog(c)
	ctx, cancel := context.WithTimeout(context.Background(), mongo.Timeout)
	defer cancel()

	// get source data creation time
	dataTimestamp := data_timestamp.GetDataSourceTimestamp(ctx, c)
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
