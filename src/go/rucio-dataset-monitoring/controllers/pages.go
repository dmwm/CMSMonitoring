package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	mymongo "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// GetIndexPage serves datasets.tmpl page
func GetIndexPage(collectionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		VerboseControllerInitLog(c)
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()
		// get source data creation time
		dataTimestamp := GetDataSourceTimestamp(ctx, c, collectionName)
		c.HTML(
			http.StatusOK,
			"datasets.tmpl",
			gin.H{
				"title":         "Home Page",
				"isShortUrl":    "false", // Should be set as string since it will be required by JS in datasets.tmpl
				"dataTimestamp": dataTimestamp.CreatedAt,
			},
		)
		VerboseControllerOutLog(start, "GetIndexPage", nil, dataTimestamp)
		return
	}
}

// GetDetailsPage serves detailed_datasets.tmpl page
func GetDetailsPage(c *gin.Context) {
	VerboseControllerInitLog(c)
	start := time.Now()
	c.HTML(
		http.StatusOK,
		"detailed_datasets.tmpl",
		gin.H{
			"title": "Detailed Datasets Page",
		},
	)
	VerboseControllerOutLog(start, "GetDetailsPage", nil, nil)
}

// GetIndexPageFromShortUrlId controller that returns page from short url hash id
func GetIndexPageFromShortUrlId(shortUrlCollectionName string, datasourceTimestampCollectionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		VerboseControllerInitLog(c)
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()

		hashId := c.Param("id")
		utils.InfoLogV1("hash Id: %s", hashId)
		shortUrlObj := GetRequestFromShortUrl(ctx, c, shortUrlCollectionName, hashId)

		dataTimestamp := GetDataSourceTimestamp(ctx, c, datasourceTimestampCollectionName)
		c.HTML(
			http.StatusOK,
			"datasets.tmpl",
			gin.H{
				"title":             "Home Page",
				"isShortUrl":        "true",
				"dtRequestShortUrl": shortUrlObj.Request,
				"dtSavedState":      shortUrlObj.SavedState,
				"dataTimestamp":     dataTimestamp.CreatedAt,
			},
		)
		VerboseControllerOutLog(start, "GetIndexPageFromShortUrlId", shortUrlObj, hashId)
		return
	}
}
