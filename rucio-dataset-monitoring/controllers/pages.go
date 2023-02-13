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
func GetMainDatasetsPage(collectionName, mainDsApiEP, shortUrlApiEP, rseDetailsApiEP, baseEP string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()
		// get source data creation time
		dataTimestamp := GetDataSourceTimestamp(ctx, c, collectionName)
		c.HTML(
			http.StatusOK,
			"main_datasets.tmpl",
			gin.H{
				"title":                               "Main Datasets",
				"govar_VERBOSITY":                     utils.Verbose,
				"govar_IS_SHORT_URL":                  false,
				"govar_SOURCE_DATE":                   dataTimestamp.CreatedAt,
				"govar_MAIN_DATASETS_API_ENDPOINT":    mainDsApiEP,
				"govar_SHORT_URL_API_ENDPOINT":        shortUrlApiEP,
				"govar_EACH_RSE_DETAILS_API_ENDPOINT": rseDetailsApiEP,
				"govar_BASE_EP":                       baseEP,
			},
		)
		return
	}
}

// GetDetailedDatasetsPage serves detailed_datasets.tmpl page
func GetDetailedDatasetsPage(collectionName, detailedDsApiEP, shortUrlApiEP, baseEP string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()
		// get source data creation time
		dataTimestamp := GetDataSourceTimestamp(ctx, c, collectionName)
		c.HTML(
			http.StatusOK,
			"detailed_datasets.tmpl",
			gin.H{
				"title":                                "Detailed MainDatasets Page",
				"govar_VERBOSITY":                      utils.Verbose,
				"govar_IS_SHORT_URL":                   false,
				"govar_SOURCE_DATE":                    dataTimestamp.CreatedAt,
				"govar_DETAILED_DATASETS_API_ENDPOINT": detailedDsApiEP,
				"govar_SHORT_URL_API_ENDPOINT":         shortUrlApiEP,
				"govar_BASE_EP":                        baseEP,
			},
		)
		return
	}
}

// GetDatasetsInTapeDiskPage serves datasets_in_tape_disk.tmpl page
func GetDatasetsInTapeDiskPage(collectionName, inTapeDiskApiEP, rseDetailsApiEP, shortUrlApiEP, baseEP string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()
		// get source data creation time
		dataTimestamp := GetDataSourceTimestamp(ctx, c, collectionName)
		c.HTML(
			http.StatusOK,
			"datasets_in_tape_disk.tmpl",
			gin.H{
				"title":              "Datasets in Both Tape and Disk Page",
				"govar_VERBOSITY":    utils.Verbose,
				"govar_IS_SHORT_URL": false,
				"govar_SOURCE_DATE":  dataTimestamp.CreatedAt,
				"govar_DATASETS_IN_TAPE_DISK_API_ENDPOINT": inTapeDiskApiEP,
				"govar_EACH_RSE_DETAILS_API_ENDPOINT":      rseDetailsApiEP,
				"govar_SHORT_URL_API_ENDPOINT":             shortUrlApiEP,
				"govar_BASE_EP":                            baseEP,
			},
		)
		return
	}
}

// GetIndexPageFromShortUrlId controller that returns page from short url hash id
func GetIndexPageFromShortUrlId(shortUrlCollectionName, datasourceTimestampCollectionName,
	mainDsApiEP, detailedDsApiEP, inTapeDiskApiEP, rseDetailsApiEP, shortUrlApiEP, baseEP string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var templateName string
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()

		hashId := c.Param("id")
		utils.InfoLogV1("hash Id: %s", hashId)
		shortUrlObj := GetRequestFromShortUrl(ctx, c, shortUrlCollectionName, hashId)
		dataTimestamp := GetDataSourceTimestamp(ctx, c, datasourceTimestampCollectionName)

		utils.InfoLogV1("ShortObj: " + shortUrlObj.Page)
		if shortUrlObj.Page == "main" {
			templateName = "main_datasets.tmpl"
		} else if shortUrlObj.Page == "detailed" {
			templateName = "detailed_datasets.tmpl"
		} else if shortUrlObj.Page == "in-tape-disk" {
			templateName = "datasets_in_tape_disk.tmpl"
		} else {
			utils.ErrorLog("No Page definition found in Short Url request: " + shortUrlObj.Page)
			return
		}
		utils.InfoLogV1("ShortObj template name: " + templateName)
		c.HTML(
			http.StatusOK,
			templateName,
			gin.H{
				"govar_VERBOSITY":                          utils.Verbose,
				"govar_IS_SHORT_URL":                       true,
				"govar_SHORT_URL_REQUEST":                  shortUrlObj.Request,
				"govar_DT_SAVED_STATE":                     shortUrlObj.SavedState,
				"govar_SOURCE_DATE":                        dataTimestamp.CreatedAt,
				"govar_MAIN_DATASETS_API_ENDPOINT":         mainDsApiEP,
				"govar_DETAILED_DATASETS_API_ENDPOINT":     detailedDsApiEP,
				"govar_DATASETS_IN_TAPE_DISK_API_ENDPOINT": inTapeDiskApiEP,
				"govar_EACH_RSE_DETAILS_API_ENDPOINT":      rseDetailsApiEP,
				"govar_SHORT_URL_API_ENDPOINT":             shortUrlApiEP,
				"govar_BASE_EP":                            baseEP,
			},
		)
		return
	}
}
