package controllers

// Copyright (c) 2023 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	mymongo "github.com/dmwm/CMSMonitoring/cpueff-goweb/mongo"
	"github.com/dmwm/CMSMonitoring/cpueff-goweb/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

// ------------------------------------------------------------------ Condor

// GetCondorMainCpuEfficiencyPage serves condor_main.tmpl page
func GetCondorMainCpuEfficiencyPage(collectionName, condorMainApiEP, condorMainEachDetailedApiEP, shortUrlApiEP, baseEP string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()
		// get source data creation time
		dataTimestamp := GetDataSourceTimestamp(ctx, c, collectionName)
		c.HTML(
			http.StatusOK,
			"condor_main.tmpl",
			gin.H{
				"govar_VERBOSITY":                              utils.Verbose,
				"govar_IS_SHORT_URL":                           false,
				"govar_SOURCE_DATE":                            dataTimestamp.StartDate + " - " + dataTimestamp.EndDate,
				"govar_CONDOR_MAIN_API_ENDPOINT":               condorMainApiEP,
				"govar_CONDOR_EACH_MAIN_DETAILED_API_ENDPOINT": condorMainEachDetailedApiEP,
				"govar_SHORT_URL_API_ENDPOINT":                 shortUrlApiEP,
				"govar_BASE_EP":                                baseEP,
			},
		)
		return
	}
}

// GetCondorDetailedCpuEfficiencyPage serves condor_detailed.tmpl page
func GetCondorDetailedCpuEfficiencyPage(collectionName, condorDetailedApiEP, shortUrlApiEP, baseEP string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()
		// get source data creation time
		dataTimestamp := GetDataSourceTimestamp(ctx, c, collectionName)
		c.HTML(
			http.StatusOK,
			"condor_detailed.tmpl",
			gin.H{
				"govar_VERBOSITY":                    utils.Verbose,
				"govar_IS_SHORT_URL":                 false,
				"govar_SOURCE_DATE":                  dataTimestamp.StartDate + " - " + dataTimestamp.EndDate,
				"govar_CONDOR_DETAILED_API_ENDPOINT": condorDetailedApiEP,
				"govar_SHORT_URL_API_ENDPOINT":       shortUrlApiEP,
				"govar_BASE_EP":                      baseEP,
			},
		)
		return
	}
}

// ------------------------------------------------------------------ Stepchain

// GetScTaskPage serves sc_task.tmpl page
func GetScTaskPage(collectionName, scTaskApiEP, scTaskEachDetailedApiEP, shortUrlApiEP, baseEP string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()
		// get source data creation time
		dataTimestamp := GetDataSourceTimestamp(ctx, c, collectionName)
		c.HTML(
			http.StatusOK,
			"sc_task.tmpl",
			gin.H{
				"govar_VERBOSITY":                          utils.Verbose,
				"govar_IS_SHORT_URL":                       false,
				"govar_SOURCE_DATE":                        dataTimestamp.StartDate + " - " + dataTimestamp.EndDate,
				"govar_SC_TASK_API_ENDPOINT":               scTaskApiEP,
				"govar_SC_EACH_TASK_DETAILED_API_ENDPOINT": scTaskEachDetailedApiEP,
				"govar_SHORT_URL_API_ENDPOINT":             shortUrlApiEP,
				"govar_BASE_EP":                            baseEP,
			},
		)
		return
	}
}

// GetScTaskCmsrunJobtypePage serves sc_task_cmsrun_jobtype.tmpl page
func GetScTaskCmsrunJobtypePage(collectionName, scTaskCmsrunJobtypeApiEP, scTaskEachDetailedApiEP, shortUrlApiEP, baseEP string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()
		// get source data creation time
		dataTimestamp := GetDataSourceTimestamp(ctx, c, collectionName)
		c.HTML(
			http.StatusOK,
			"sc_task_cmsrun_jobtype.tmpl",
			gin.H{
				"govar_VERBOSITY":                           utils.Verbose,
				"govar_IS_SHORT_URL":                        false,
				"govar_SOURCE_DATE":                         dataTimestamp.StartDate + " - " + dataTimestamp.EndDate,
				"govar_SC_TASK_CMSRUN_JOBTYPE_API_ENDPOINT": scTaskCmsrunJobtypeApiEP,
				"govar_SC_EACH_TASK_DETAILED_API_ENDPOINT":  scTaskEachDetailedApiEP,
				"govar_SHORT_URL_API_ENDPOINT":              shortUrlApiEP,
				"govar_BASE_EP":                             baseEP,
			},
		)
		return
	}
}

// ------------------------------------------------------------------ Common

// GetIndexPageFromShortUrlId controller that returns page from short url hash id
func GetIndexPageFromShortUrlId(shortUrlCollectionName, datasourceTimestampCollectionName,
	condorMainApiEP, condorDetailedApiEP, condorMainEachDetailedApiEP,
	scTaskApiEP, ScTaskCmsrunJobtypeApiEP, scTaskEachDetailedApiEP,
	shortUrlApiEP, baseEP string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var templateName string
		ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)
		defer cancel()

		hashId := c.Param("id")
		utils.InfoLogV1("hash Id: %s", hashId)
		shortUrlObj := GetRequestFromShortUrl(ctx, c, shortUrlCollectionName, hashId)
		dataTimestamp := GetDataSourceTimestamp(ctx, c, datasourceTimestampCollectionName)

		utils.InfoLogV1("ShortObj: " + shortUrlObj.Page)

		// Page is coming from JS `PAGE_ENDPOINT` variable value. Assign different template file for each page
		switch shortUrlObj.Page {
		case "condor-main":
			templateName = "condor_main.tmpl"
		case "condor-detailed":
			templateName = "condor_detailed.tmpl"
		case "sc-main":
			templateName = "sc_task.tmpl"
		case "sc-detail-task-cmsrun-jobtype":
			templateName = "sc_task_cmsrun_jobtype.tmpl"
		default:
			utils.ErrorLog("No Page definition found in Short Url request: " + shortUrlObj.Page)
			return
		}

		utils.InfoLogV1("ShortObj template name: " + templateName)
		c.HTML(
			http.StatusOK,
			templateName,
			gin.H{
				"govar_VERBOSITY":                              utils.Verbose,
				"govar_IS_SHORT_URL":                           true,
				"govar_SHORT_URL_REQUEST":                      shortUrlObj.Request,
				"govar_DT_SAVED_STATE":                         shortUrlObj.SavedState,
				"govar_SOURCE_DATE":                            dataTimestamp.StartDate + " - " + dataTimestamp.EndDate,
				"govar_CONDOR_MAIN_API_ENDPOINT":               condorMainApiEP,
				"govar_CONDOR_DETAILED_API_ENDPOINT":           condorDetailedApiEP,
				"govar_CONDOR_EACH_MAIN_DETAILED_API_ENDPOINT": condorMainEachDetailedApiEP,
				"govar_SC_TASK_API_ENDPOINT":                   scTaskApiEP,
				"govar_SC_TASK_CMSRUN_JOBTYPE_API_ENDPOINT":    ScTaskCmsrunJobtypeApiEP,
				"govar_SC_EACH_TASK_DETAILED_API_ENDPOINT":     scTaskEachDetailedApiEP,
				"govar_SHORT_URL_API_ENDPOINT":                 shortUrlApiEP,
				"govar_BASE_EP":                                baseEP,
			},
		)
		return
	}
}
