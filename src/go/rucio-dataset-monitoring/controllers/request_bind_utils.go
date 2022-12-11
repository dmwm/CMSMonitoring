package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/models"
	mymongo "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"log"
	"time"
)

// InitializeCtxAndBindRequestBody initialize controller requirements
//
//	initialize context, bind request json for the controller, prints initial logs, gets start time etc.
func InitializeCtxAndBindRequestBody(c *gin.Context, req interface{}) (context.Context, context.CancelFunc, time.Time, interface{}) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)

	req, err := bindRequest(c, req)
	if err != nil {
		tempReqBody, exists := c.Get(gin.BodyBytesKey)
		if !exists {
			utils.ErrorResponse(c, "bad request", err, "Request body does not exist in Gin context")
		} else {
			utils.ErrorResponse(c, "bad request", err, string(tempReqBody.([]byte)))
		}
	}
	return ctx, cancel, start, req
}

// bindRequest binds request according to provided type
func bindRequest(c *gin.Context, req interface{}) (any, error) {
	// Get request json with validation
	switch r := req.(type) {
	case models.DataTableRequest:
		err := c.ShouldBindBodyWith(&r, binding.JSON)
		utils.InfoLogV2("incoming request body bind to: %s", "DataTableSearchBuilderRequest")
		return r, err
	case models.ShortUrlRequest:
		err := c.ShouldBindBodyWith(&r, binding.JSON)
		utils.InfoLogV2("incoming request body bind to: %s", "ShortUrlRequest")
		return r, err
	case models.SingleDetailedDatasetsRequest:
		err := c.ShouldBindBodyWith(&r, binding.JSON)
		utils.InfoLogV2("incoming request body bind to: %s", "SingleDetailedDatasetsRequest")
		return r, err
	default:
		utils.ErrorLog("unknown request struct, it did not match: %#v", req)
		return nil, errors.New("unknown request struct, no match in switch case")
	}
}

// VerboseControllerOutLog prints debug logs after controller processed the api call
func VerboseControllerOutLog(start time.Time, name string, req any, data any) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if utils.Verbose > 0 {
		elapsed := time.Since(start)
		req, err := json.Marshal(req)
		if err != nil {
			log.Printf("[ERROR] ------ cannot marshall request, err:%s", err)
		} else {
			r := string(req)
			if utils.Verbose >= 2 {
				// Response returns all query results, its verbosity should be at least 2
				data, err1 := json.Marshal(data)
				if err1 != nil {
					log.Printf("[ERROR] ------ cannot marshall additional verbose log data, err:%s", err1)
				} else {
					d := string(data)
					log.Printf("[DEBUG] ------\n -query time [%s] : %s\n\n -request body: %s\n\n -response: %s\n\n", name, elapsed, r, d)
				}
			} else {
				log.Printf("[INFO] ------\n -query time [%s] : %s\n\n -request body: %s\n\n -response: %#v\n\n", name, elapsed, req, nil)
			}
		}
	}
}
