package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	mymongo "github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/mongo"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

// GetMainDatasetsPage serves main_datasets.tmpl page
func GetMainDatasetsPage(collectionName, mainDsApiEP, shortUrlApiEP, mainDsDetailsApiEP string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()
		// get source data creation time
		dataTimestamp := GetDataSourceTimestamp(ctx, c, collectionName)
		c.HTML(
			http.StatusOK,
			"main_datasets.tmpl",
			gin.H{
				"title":                             "Main Datasets",
				"VERBOSITY":                         utils.Verbose,
				"IS_SHORT_URL":                      false,
				"SOURCE_DATE":                       dataTimestamp.CreatedAt,
				"MAIN_DATASETS_API_ENDPOINT":        mainDsApiEP,
				"SHORT_URL_API_ENDPOINT":            shortUrlApiEP,
				"MAIN_DATASET_DETAILS_API_ENDPOINT": mainDsDetailsApiEP,
			},
		)
		return
	}
}

// GetDetailedDatasetsPage serves detailed_datasets.tmpl page
func GetDetailedDatasetsPage(collectionName, detailedDsApiEP, shortUrlApiEP string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()
		// get source data creation time
		dataTimestamp := GetDataSourceTimestamp(ctx, c, collectionName)
		c.HTML(
			http.StatusOK,
			"detailed_datasets.tmpl",
			gin.H{
				"title":                          "Detailed MainDatasets Page",
				"VERBOSITY":                      utils.Verbose,
				"IS_SHORT_URL":                   false,
				"SOURCE_DATE":                    dataTimestamp.CreatedAt,
				"DETAILED_DATASETS_API_ENDPOINT": detailedDsApiEP,
				"SHORT_URL_API_ENDPOINT":         shortUrlApiEP,
			},
		)
		return
	}
}

// GetIndexPageFromShortUrlId controller that returns page from short url hash id
func GetIndexPageFromShortUrlId(shortUrlCollectionName, datasourceTimestampCollectionName, mainDsApiEP, shortUrlApiEP, mainDsDetailsApiEP string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()

		hashId := c.Param("id")
		utils.InfoLogV1("hash Id: %s", hashId)
		shortUrlObj := GetRequestFromShortUrl(ctx, c, shortUrlCollectionName, hashId)

		dataTimestamp := GetDataSourceTimestamp(ctx, c, datasourceTimestampCollectionName)
		c.HTML(
			http.StatusOK,
			"main_datasets.tmpl",
			gin.H{
				"title":                             "Home Page",
				"VERBOSITY":                         utils.Verbose,
				"IS_SHORT_URL":                      true,
				"SHORT_URL_REQUEST":                 shortUrlObj.Request,
				"DT_SAVED_STATE":                    shortUrlObj.SavedState,
				"SOURCE_DATE":                       dataTimestamp.CreatedAt,
				"MAIN_DATASETS_API_ENDPOINT":        mainDsApiEP,
				"SHORT_URL_API_ENDPOINT":            shortUrlApiEP,
				"MAIN_DATASET_DETAILS_API_ENDPOINT": mainDsDetailsApiEP,
			},
		)
		return
	}
}
