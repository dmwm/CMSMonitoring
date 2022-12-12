package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	"errors"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/models"
	mymongo "github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/mongo"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// InitializeCtxAndBindRequestBody initialize controller requirements
//
//	initialize context, bind request json for the controller, prints initial logs, gets start time etc.
func InitializeCtxAndBindRequestBody(c *gin.Context, req interface{}) (context.Context, context.CancelFunc, interface{}) {
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
	return ctx, cancel, req
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