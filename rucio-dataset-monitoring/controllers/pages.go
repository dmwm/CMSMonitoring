package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	mymongo "github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/mongo"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

// GetIndexPage serves datasets.tmpl page
func GetIndexPage(collectionName, datasetsApiEP, shortUrlApiEP, rseDetailsApiEP string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()
		// get source data creation time
		dataTimestamp := GetDataSourceTimestamp(ctx, c, collectionName)
		c.HTML(
			http.StatusOK,
			"datasets.tmpl",
			gin.H{
				"title":                    "Home Page",
				"VERBOSITY":                utils.Verbose,
				"IS_SHORT_URL":             false,
				"SOURCE_DATE":              dataTimestamp.CreatedAt,
				"DATASETS_API_ENDPOINT":    datasetsApiEP,
				"SHORT_URL_API_ENDPOINT":   shortUrlApiEP,
				"RSE_DETAILS_API_ENDPOINT": rseDetailsApiEP,
			},
		)
		return
	}
}

// GetDetailsPage serves detailed_datasets.tmpl page
func GetDetailsPage(c *gin.Context) {
	c.HTML(
		http.StatusOK,
		"detailed_datasets.tmpl",
		gin.H{
			"title": "Detailed Datasets Page",
		},
	)
}

// GetIndexPageFromShortUrlId controller that returns page from short url hash id
func GetIndexPageFromShortUrlId(shortUrlCollectionName string, datasourceTimestampCollectionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
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
				"VERBOSITY":         utils.Verbose,
				"IS_SHORT_URL":      true,
				"SHORT_URL_REQUEST": shortUrlObj.Request,
				"DT_SAVED_STATE":    shortUrlObj.SavedState,
				"SOURCE_DATE":       dataTimestamp.CreatedAt,
			},
		)
		return
	}
}
