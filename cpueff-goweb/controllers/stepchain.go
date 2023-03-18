package controllers

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
	"strings"
)

var (
	// stepchainUniqueSortColumn required for pagination in order
	stepchainUniqueSortColumn = "_id"
)

// --------------------------- Stepchain Main Controller Functions ------------------------------------------

// ScTaskCtrl direct API: controller of Task
func ScTaskCtrl(configs models.Configuration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.DataTableSearchBuilderRequest to the controller initializer and use same type in casting
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.DataTableRequest{})
		defer cancel()

		c.JSON(http.StatusOK, getScTaskResults(ctx, c, configs, req.(models.DataTableRequest)))
		return
	}
}

// ScTaskCmsrunJobtypeCtrl direct API: controller of TaskCmsrunJobtype (each of them grouped by, so includes group-by columns)
func ScTaskCmsrunJobtypeCtrl(configs models.Configuration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.DataTableCustomRequest to the controller initializer and use same type in casting
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.DataTableRequest{})
		defer cancel()

		c.JSON(http.StatusOK, getScTaskCmsrunJobtypeResults(ctx, c, configs, req.(models.DataTableRequest)))
		return
	}
}

// ScTaskCmsrunJobtypeSiteCtrl direct API: controller of TaskCmsrunJobtypeSite (each of them grouped by so includes groupe-by columns)
func ScTaskCmsrunJobtypeSiteCtrl(configs models.Configuration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.DataTableCustomRequest to the controller initializer and use same type in casting
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.DataTableRequest{})
		defer cancel()

		c.JSON(http.StatusOK, getScTaskCmsrunJobtypeSiteResults(ctx, c, configs, req.(models.DataTableRequest)))
		return
	}
}

// ScEachSiteDetailCtrl row details template render controller: task row details API: returns a single Task's Cmsrun+Jobtype+Site cpu efficiencies
func ScEachSiteDetailCtrl(configs models.Configuration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var detailedRows []models.StepchainTaskCmsrunJobtypeSite
		var err error
		var cursor *mongo.Cursor
		var sortType = bson.D{bson.E{Key: "StepName", Value: 1}}
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.ScTaskEachDetailedRequest{})
		defer cancel()

		query := bson.M{}
		// Cast interface to request
		r := req.(models.ScTaskEachDetailedRequest)
		if r.Task != "" {
			query["Task"] = r.Task
		}
		if r.StepName != "" {
			query["StepName"] = r.StepName
		}
		if r.JobType != "" {
			query["JobType"] = r.JobType
		}

		collection := mymongo.GetCollection(configs.CollectionNames.ScTaskCmsrunJobtypeSite)
		cursor, err = mymongo.GetFindQueryResults(ctx, collection,
			query,
			sortType, 0, 0,
		)
		if err != nil {
			utils.ErrorResponse(c, "Find query failed", err, "")
		}
		if err = cursor.All(ctx, &detailedRows); err != nil {
			utils.ErrorResponse(c, "single detailed task details cursor failed", err, "")
		}

		// Get processed data time period: start date and end date that
		dataTimestamp := GetDataSourceTimestamp(ctx, c, configs.CollectionNames.DatasourceTimestamp)
		// Add Links of other services for the workflow/task to the 'Links' column
		utilScTaskCmsrunJobtypeSiteExternalLinks(detailedRows, dataTimestamp, configs.ExternalLinks)

		c.HTML(http.StatusOK,
			"sc_task_each_detailed.tmpl",
			gin.H{"data": detailedRows},
		)
		return
	}
}

// -----------------------------------------------------------------------------------------------

// [Stepchain Task controller utils] -------------------------------------------------------------

// getScTaskResults get query results efficiently
func getScTaskResults(ctx context.Context, c *gin.Context, configs models.Configuration, req models.DataTableRequest) models.DatatableBaseResponse {
	collection := mymongo.GetCollection(configs.CollectionNames.ScTask)
	var cpuEffs []models.StepchainTask

	// Should use SearchBuilderRequest query
	searchQuery := mymongo.SearchQueryForSearchBuilderRequest(&req.SearchBuilderRequest, models.Stepchain)
	sortQuery := mymongo.SortQueryBuilder(&req, stepchainUniqueSortColumn)
	length := req.Length
	skip := req.Start

	cursor, err := mymongo.GetFindQueryResults(ctx, collection, searchQuery, sortQuery, skip, length)
	if err != nil {
		utils.ErrorResponse(c, "Find query failed", err, "")
	}
	if err = cursor.All(ctx, &cpuEffs); err != nil {
		utils.ErrorResponse(c, "stepchainTask cursor failed", err, "")
	}
	// Get processed data time period: start date and end date that
	dataTimestamp := GetDataSourceTimestamp(ctx, c, configs.CollectionNames.DatasourceTimestamp)

	// Add Links of other services for the workflow/task to the 'Links' column
	utilScTaskAddExternalLinks(cpuEffs, dataTimestamp, configs.ExternalLinks)

	totalRecCount := getScFilteredCount(ctx, c, collection, searchQuery, req.Draw)
	if totalRecCount < 0 {
		utils.ErrorResponse(c, "getFilteredCount cursor failed", err, "datatables draw value cannot be less than 1, it is: "+string(rune(req.Draw)))
	}
	filteredRecCount := totalRecCount
	return models.DatatableBaseResponse{
		Draw:            req.Draw,
		RecordsTotal:    totalRecCount,
		RecordsFiltered: filteredRecCount,
		Data:            cpuEffs,
	}
}

// "RegMgr2 - WMArchive(MONIT)" links to 'Links' columns. Check configs.json
// utilScTaskAddExternalLinks creates external links for the task
func utilScTaskAddExternalLinks(cpuEffs []models.StepchainTask, dataTimestamp models.DataSourceTS, links models.ExternalLinks) {
	for i := 0; i < len(cpuEffs); i++ {
		replacer := strings.NewReplacer(
			links.StrStartDate, dataTimestamp.StartDate,
			links.StrEndDate, dataTimestamp.EndDate,
			links.StrTaskNamePrefix, utilGetTaskNamePrefix(cpuEffs[i].Task),
			links.StrTaskName, cpuEffs[i].Task,
		)
		// Kibana needs full task name, but reqmgr2 accepts only "X" from task name of "/X/Y"
		// Assign complete link HTML dom to the 'Links' column : "ReqMgr - WMArchive(MONIT)"
		cpuEffs[i].Links = template.HTML(replacer.Replace(links.LinkReqMgr + " - " + links.LinkEsWmarchive))
	}
}

// [Stepchain Task+Cmsrun+Jobtype controller utils] -------------------------------------------

// getScTaskCmsrunJobtypeResults get query results efficiently
func getScTaskCmsrunJobtypeResults(ctx context.Context, c *gin.Context, configs models.Configuration, req models.DataTableRequest) models.DatatableBaseResponse {
	collection := mymongo.GetCollection(configs.CollectionNames.ScTaskCmsrunJobtype)
	var cpuEffs []models.StepchainTaskCmsrunJobtype

	// Should use SearchBuilderRequest query
	searchQuery := mymongo.SearchQueryForSearchBuilderRequest(&req.SearchBuilderRequest, models.Stepchain)
	sortQuery := mymongo.SortQueryBuilder(&req, stepchainUniqueSortColumn)
	length := req.Length
	skip := req.Start
	cursor, err := mymongo.GetFindQueryResults(ctx, collection, searchQuery, sortQuery, skip, length)
	if err != nil {
		utils.ErrorResponse(c, "Find query failed", err, "")
	}
	if err = cursor.All(ctx, &cpuEffs); err != nil {
		utils.ErrorResponse(c, "stepchain task-cmsrun-jobtype cpu eff cursor failed", err, "")
	}

	// Get processed data time period: start date and end date that
	dataTimestamp := GetDataSourceTimestamp(ctx, c, configs.CollectionNames.DatasourceTimestamp)

	// Add Links of other services for the workflow/task to the 'Links' column
	utilScTaskCmsrunJobtypeExternalLinks(cpuEffs, dataTimestamp, configs.ExternalLinks)

	totalRecCount := getScFilteredCount(ctx, c, collection, searchQuery, req.Draw)
	if totalRecCount < 0 {
		utils.ErrorResponse(c, "getFilteredCount cursor failed", err, "datatables draw value cannot be less than 1, it is: "+string(rune(req.Draw)))
	}
	filteredRecCount := totalRecCount
	return models.DatatableBaseResponse{
		Draw:            req.Draw,
		RecordsTotal:    totalRecCount,
		RecordsFiltered: filteredRecCount,
		Data:            cpuEffs,
	}
}

// [Stepchain Task+Cmsrun+Jobtype+Site controller utils] -------------------------------------------

// getScTaskCmsrunJobtypeSiteResults get query results efficiently
func getScTaskCmsrunJobtypeSiteResults(ctx context.Context, c *gin.Context, configs models.Configuration, req models.DataTableRequest) models.DatatableBaseResponse {
	collection := mymongo.GetCollection(configs.CollectionNames.ScTaskCmsrunJobtypeSite)
	var cpuEffs []models.StepchainTaskCmsrunJobtypeSite

	// Should use SearchBuilderRequest query
	searchQuery := mymongo.SearchQueryForSearchBuilderRequest(&req.SearchBuilderRequest, models.Stepchain)
	sortQuery := mymongo.SortQueryBuilder(&req, stepchainUniqueSortColumn)
	length := req.Length
	skip := req.Start
	cursor, err := mymongo.GetFindQueryResults(ctx, collection, searchQuery, sortQuery, skip, length)
	if err != nil {
		utils.ErrorResponse(c, "Find query failed", err, "")
	}
	if err = cursor.All(ctx, &cpuEffs); err != nil {
		utils.ErrorResponse(c, "stepchain task-cmsrun-jobtype-site cpu eff cursor failed", err, "")
	}

	// Get processed data time period: start date and end date that
	dataTimestamp := GetDataSourceTimestamp(ctx, c, configs.CollectionNames.DatasourceTimestamp)

	// Add Links of other services for the workflow/task to the 'Links' column
	utilScTaskCmsrunJobtypeSiteExternalLinks(cpuEffs, dataTimestamp, configs.ExternalLinks)

	totalRecCount := getScFilteredCount(ctx, c, collection, searchQuery, req.Draw)
	if totalRecCount < 0 {
		utils.ErrorResponse(c, "getFilteredCount cursor failed", err, "datatables draw value cannot be less than 1, it is: "+string(rune(req.Draw)))
	}
	filteredRecCount := totalRecCount
	return models.DatatableBaseResponse{
		Draw:            req.Draw,
		RecordsTotal:    totalRecCount,
		RecordsFiltered: filteredRecCount,
		Data:            cpuEffs,
	}
}

// ""RegMgr2 - WMArchive(MONIT)" links to 'Links' columns. Check configs.json
// utilScTaskCmsrunJobtypeSiteExternalLinks creates external links for the task
func utilScTaskCmsrunJobtypeExternalLinks(cpuEffs []models.StepchainTaskCmsrunJobtype, dataTimestamp models.DataSourceTS, links models.ExternalLinks) {
	for i := 0; i < len(cpuEffs); i++ {
		replacer := strings.NewReplacer(
			links.StrStartDate, dataTimestamp.StartDate,
			links.StrEndDate, dataTimestamp.EndDate,
			links.StrTaskNamePrefix, utilGetTaskNamePrefix(cpuEffs[i].Task),
			links.StrTaskName, cpuEffs[i].Task,
		)
		// Kibana needs full task name, but reqmgr2 accepts only "X" from task name of "/X/Y"
		// Assign complete link HTML dom to the 'Links' column : "ReqMgr - WMArchive(MONIT)"
		cpuEffs[i].Links = template.HTML(replacer.Replace(links.LinkReqMgr + " - " + links.LinkEsWmarchive))
	}
}

// "WMArchive(MONIT)" links to 'Links' columns. Check configs.json
// utilScTaskCmsrunJobtypeSiteExternalLinks creates external links for the task
func utilScTaskCmsrunJobtypeSiteExternalLinks(cpuEffs []models.StepchainTaskCmsrunJobtypeSite, dataTimestamp models.DataSourceTS, links models.ExternalLinks) {
	for i := 0; i < len(cpuEffs); i++ {
		replacer := strings.NewReplacer(
			links.StrStartDate, dataTimestamp.StartDate,
			links.StrEndDate, dataTimestamp.EndDate,
			links.StrTaskName, cpuEffs[i].Task,
			links.StrJobType, cpuEffs[i].JobType,
		)
		// Assign complete link HTML dom to the 'Links' column : "WMArchive(MONIT)"
		cpuEffs[i].Links = template.HTML(replacer.Replace(links.LinkEsWmarchiveJobType))
	}
}

// getScFilteredCount total document count of the filter result in the stepchain DBs
func getScFilteredCount(ctx context.Context, c *gin.Context, collection *mongo.Collection, query bson.M, draw int) int64 {
	if draw < 1 {
		return -1
	}
	// First opening of the page or search query is different from the previous one
	cnt, err := mymongo.GetCount(ctx, collection, query)
	if err != nil {
		utils.ErrorResponse(c, "TotalRecCount query failed", err, "")
	}
	return cnt
}

func utilGetTaskNamePrefix(task string) string {
	return strings.Split(task, "/")[1]
}
