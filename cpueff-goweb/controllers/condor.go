package controllers

// Copyright (c) 2023 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	"github.com/dmwm/CMSMonitoring/cpueff-goweb/models"
	mymongo "github.com/dmwm/CMSMonitoring/cpueff-goweb/mongo"
	"github.com/dmwm/CMSMonitoring/cpueff-goweb/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"html/template"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	// condorMainQueryHolder holds last search query to compare with next one
	condorMainQueryHolder bson.M

	// condorMainFilteredCountHolder holds filtered count value of last query
	condorMainFilteredCountHolder int64

	// condorMainUniqueSortColumn required for pagination in order
	condorMainUniqueSortColumn = "_id"

	// condorDetailedUniqueSortColumn required for pagination in order
	condorDetailedUniqueSortColumn = "_id"
)

// ------------------------------------------------------------------ Condor main workflows controller

// CondorMainCtrl controller
func CondorMainCtrl(configs models.Configuration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.DataTableSearchBuilderRequest to the controller initializer and use same type in casting
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.DataTableRequest{})
		defer cancel()

		c.JSON(http.StatusOK,
			getCondorWfCpuEffResults(
				ctx,
				c,
				req.(models.DataTableRequest),
				configs,
			))
		return
	}
}

// getCondorWfCpuEffResults get query results efficiently
func getCondorWfCpuEffResults(ctx context.Context, c *gin.Context, req models.DataTableRequest, configs models.Configuration) models.DatatableBaseResponse {
	collection := mymongo.GetCollection(configs.CollectionNames.CondorMainWorkflows)
	var wfCpuEffs []models.CondorWfCpuEff

	// Should use SearchBuilderRequest query
	searchQuery := mymongo.SearchQueryForSearchBuilderRequest(&req.SearchBuilderRequest, models.Condor)
	sortQuery := mymongo.SortQueryBuilder(&req, condorMainUniqueSortColumn)
	length := req.Length
	skip := req.Start

	cursor, err := mymongo.GetFindQueryResults(ctx, collection, searchQuery, sortQuery, skip, length)
	if err != nil {
		utils.ErrorResponse(c, "Find query failed", err, "")
	}
	if err = cursor.All(ctx, &wfCpuEffs); err != nil {
		utils.ErrorResponse(c, "wfCpuEffs cursor failed", err, "")
	}

	// Get processed data time period: start date and end date that
	dataTimestamp := GetDataSourceTimestamp(ctx, c, configs.CollectionNames.DatasourceTimestamp)

	// Add Links of other services for the workflow/task to the 'Links' column
	utilCondorMainAddExternalLinks(wfCpuEffs, dataTimestamp, configs.ExternalLinks)

	totalRecCount := getFilteredCount(ctx, c, collection, searchQuery, req.Draw)
	if totalRecCount < 0 {
		utils.ErrorResponse(c, "getFilteredCount cursor failed", err, "datatables draw value cannot be less than 1, it is: "+string(rune(req.Draw)))
	}

	filteredRecCount := totalRecCount
	return models.DatatableBaseResponse{
		Draw:            req.Draw,
		RecordsTotal:    totalRecCount,
		RecordsFiltered: filteredRecCount,
		Data:            wfCpuEffs,
	}
}

// "McM - PMon(dmytro) - Unified(report) - ES_T1T2" links to 'Links' columns. Check configs.json
// utilCondorMainAddExternalLinks creates external links for the workflow
func utilCondorMainAddExternalLinks(wfCpuEffs []models.CondorWfCpuEff, dataTimestamp models.DataSourceTS, links models.ExternalLinks) {
	for i := 0; i < len(wfCpuEffs); i++ {
		replacer := strings.NewReplacer(
			links.StrStartDate, dataTimestamp.StartDate,
			links.StrEndDate, dataTimestamp.EndDate,
			links.StrWorkflow, wfCpuEffs[i].Workflow,
			links.StrWmagentRequestName, wfCpuEffs[i].WmagentRequestName,
			links.StrStartDateMsec, dateToMsecStr(dataTimestamp.StartDate),
			links.StrEndDateMsec, dateToMsecStr(dataTimestamp.EndDate),
		)
		// Assign complete link HTML dom to the 'Links' column : "McM - PMon - Unified - ES_T1T2"
		wfCpuEffs[i].Links = template.HTML(replacer.Replace(
			links.LinkMonteCarloManagement + " - " + links.LinkDmytroProdMon + " - " +
				links.LinkUnifiedReport + " - " + links.LinkEsCms + " - " + links.LinkGrafana,
		))
	}
}

// ------------------------------------------------------------------ Condor main workflow each entry details controller

// CondorMainEachDetailedRowCtrl controller that returns a single wf cpu efficiencies by Site
func CondorMainEachDetailedRowCtrl(configs models.Configuration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			detailedRows []models.CondorSiteCpuEff
			err          error
			cursor       *mongo.Cursor
			sortType     = bson.D{bson.E{Key: "Site", Value: 1}} //
		)
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.CondorMainEachDetailedRequest{})
		defer cancel()

		// Cast interface to request
		r := req.(models.CondorMainEachDetailedRequest)

		collection := mymongo.GetCollection(configs.CollectionNames.CondorDetailed)
		cursor, err = mymongo.GetFindQueryResults(ctx, collection,
			bson.M{
				"Type":               r.Type,
				"Workflow":           r.Workflow,
				"WmagentRequestName": r.WmagentRequestName,
				"CpuEffOutlier":      r.CpuEffOutlier},
			sortType, 0, 0,
		)

		if err != nil {
			utils.ErrorResponse(c, "Find query failed", err, "")
		}
		if err = cursor.All(ctx, &detailedRows); err != nil {
			utils.ErrorResponse(c, "single detailed workflow by Site cursor failed", err, "")
		}
		// Get processed data time period: start date and end date that
		dataTimestamp := GetDataSourceTimestamp(ctx, c, configs.CollectionNames.DatasourceTimestamp)

		// Add Links of other services for the workflow/task to the 'Links' column
		utilCondorSiteDetailedAddExternalLinks(detailedRows, dataTimestamp, configs.ExternalLinks)

		c.HTML(http.StatusOK,
			"condor_row_site_detailed.tmpl",
			gin.H{"data": detailedRows},
		)
		return
	}
}

// 'Unified Logs' and 'ES_t1t2'(es-cms) links to 'Links' columns. Check configs.json
// utilCondorSiteDetailedAddExternalLinks creates external links for the workflow
func utilCondorSiteDetailedAddExternalLinks(siteCpuEffs []models.CondorSiteCpuEff, dataTimestamp models.DataSourceTS, links models.ExternalLinks) {
	for i := 0; i < len(siteCpuEffs); i++ {
		replacer := strings.NewReplacer(
			links.StrStartDate, dataTimestamp.StartDate,
			links.StrEndDate, dataTimestamp.EndDate,
			links.StrWorkflow, siteCpuEffs[i].Workflow,
			links.StrWmagentRequestName, siteCpuEffs[i].WmagentRequestName,
			links.StrSite, siteCpuEffs[i].Site,
			links.StrStartDateMsec, dateToMsecStr(dataTimestamp.StartDate),
			links.StrEndDateMsec, dateToMsecStr(dataTimestamp.EndDate),
		)
		// Assign complete link HTML dom to the 'Links' column : "Unified - ES_T1T2"
		siteCpuEffs[i].Links = template.HTML(replacer.Replace(
			links.LinkUnifiedReport + " - " + links.LinkEsCmsWithSite + " - " + links.LinkGrafanaWithSite,
		))
	}
}

// ------------------------------------------------------------------ Condor workflow details controller (Separate page)

// CondorDetailedCtrl controller that returns condor site details cpu efficiencies according to DataTable request json
func CondorDetailedCtrl(configs models.Configuration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.DataTableCustomRequest to the controller initializer and use same type in casting
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.DataTableRequest{})
		defer cancel()

		c.JSON(http.StatusOK, getCondorDetailedResults(ctx, c, configs, req.(models.DataTableRequest)))
		return
	}
}

// getFilteredCount total document count of the filter result in the condor-main DB
func getFilteredCount(ctx context.Context, c *gin.Context, collection *mongo.Collection, query bson.M, draw int) int64 {
	if draw < 1 {
		return -1
	} else if (draw == 1) || (!reflect.DeepEqual(condorMainQueryHolder, query)) {
		// First opening of the page or search query is different from the previous one
		cnt, err := mymongo.GetCount(ctx, collection, query)
		if err != nil {
			utils.ErrorResponse(c, "TotalRecCount query failed", err, "")
		}
		condorMainFilteredCountHolder = cnt
		condorMainQueryHolder = query
		utils.InfoLogV1("filter query comparison with previous query: MATCH")
	} else {
		// If search query is still same, count should be same, so return GFilteredCount
		utils.InfoLogV1("filter query comparison with previous query: MISMATCH")
	}
	return condorMainFilteredCountHolder
}

// getCondorDetailedResults get query results efficiently
func getCondorDetailedResults(ctx context.Context, c *gin.Context, configs models.Configuration, req models.DataTableRequest) models.DatatableBaseResponse {
	collection := mymongo.GetCollection(configs.CollectionNames.CondorDetailed)
	var siteCpuEffs []models.CondorSiteCpuEff

	// Should use SearchBuilderRequest query
	searchQuery := mymongo.SearchQueryForSearchBuilderRequest(&req.SearchBuilderRequest, models.Condor)
	sortQuery := mymongo.SortQueryBuilder(&req, condorDetailedUniqueSortColumn)
	length := req.Length
	skip := req.Start

	cursor, err := mymongo.GetFindQueryResults(ctx, collection, searchQuery, sortQuery, skip, length)
	if err != nil {
		utils.ErrorResponse(c, "Find query failed", err, "")
	}
	if err = cursor.All(ctx, &siteCpuEffs); err != nil {
		utils.ErrorResponse(c, "condor site cpu eff cursor failed", err, "")
	}

	// Get processed data time period: start date and end date that
	dataTimestamp := GetDataSourceTimestamp(ctx, c, configs.CollectionNames.DatasourceTimestamp)

	// Add Links of other services for the workflow/task to the 'Links' column
	utilCondorSiteDetailedAddExternalLinks(siteCpuEffs, dataTimestamp, configs.ExternalLinks)

	totalRecCount := getFilteredCount(ctx, c, collection, searchQuery, req.Draw)
	if totalRecCount < 0 {
		utils.ErrorResponse(c, "getFilteredCount cursor failed", err, "datatables draw value cannot be less than 1, it is: "+string(rune(req.Draw)))
	}

	filteredRecCount := totalRecCount
	return models.DatatableBaseResponse{
		Draw:            req.Draw,
		RecordsTotal:    totalRecCount,
		RecordsFiltered: filteredRecCount,
		Data:            siteCpuEffs,
	}
}

// dateToMsecStr used to define Grafana dashboard msec "from" "to" dates from time duration of Spark job
func dateToMsecStr(dateString string) string {
	date, _ := time.Parse("2006-01-02", dateString)
	return strconv.FormatInt(date.UnixMilli(), 10)
}
