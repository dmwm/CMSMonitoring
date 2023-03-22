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

// StepchainMainPageCtrl direct API: controller of Task
func StepchainMainPageCtrl(configs models.Configuration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.DataTableSearchBuilderRequest to the controller initializer and use same type in casting
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.DataTableRequest{})
		defer cancel()

		c.JSON(http.StatusOK, getStepchainTaskResults(ctx, c, configs, req.(models.DataTableRequest)))
		return
	}
}

// getStepchainTaskResults get query results efficiently
func getStepchainTaskResults(ctx context.Context, c *gin.Context, configs models.Configuration, req models.DataTableRequest) models.DatatableBaseResponse {
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

// --------------------------- Stepchain Detailed(Task+Cmsrun+Jobtype) Controller Functions ------------------------------------------

// StepchainDetailedPageCtrl direct API: controller of TaskCmsrunJobtype (each of them grouped by, so includes group-by columns)
func StepchainDetailedPageCtrl(configs models.Configuration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.DataTableCustomRequest to the controller initializer and use same type in casting
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.DataTableRequest{})
		defer cancel()

		c.JSON(http.StatusOK, getStepchainTaskCmsrunJobtypeResults(ctx, c, configs, req.(models.DataTableRequest)))
		return
	}
}

// getStepchainTaskCmsrunJobtypeResults get query results efficiently
func getStepchainTaskCmsrunJobtypeResults(ctx context.Context, c *gin.Context, configs models.Configuration, req models.DataTableRequest) models.DatatableBaseResponse {
	collection := mymongo.GetCollection(configs.CollectionNames.ScTaskCmsrunJobtype)
	var cpuEffs []models.StepchainTaskWithCmsrunJobtype

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

// ""RegMgr2 - WMArchive(MONIT)" links to 'Links' columns. Check configs.json
// utilScTaskCmsrunJobtypeSiteExternalLinks creates external links for the task
func utilScTaskCmsrunJobtypeExternalLinks(cpuEffs []models.StepchainTaskWithCmsrunJobtype, dataTimestamp models.DataSourceTS, links models.ExternalLinks) {
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

// --------------------------- Stepchain SiteDetailed(Task+Cmsrun+Jobtype+Site) Controller Functions ------------------------------------------

// StepchainSiteDetailedPageCtrl direct API: controller of TaskCmsrunJobtypeSite (each of them grouped by, so includes group-by columns)
func StepchainSiteDetailedPageCtrl(configs models.Configuration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.DataTableCustomRequest to the controller initializer and use same type in casting
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.DataTableRequest{})
		defer cancel()

		c.JSON(http.StatusOK, getStepchainTaskCmsrunJobtypeSiteResults(ctx, c, configs, req.(models.DataTableRequest)))
		return
	}
}

// getStepchainTaskCmsrunJobtypeSiteResults get query results efficiently
func getStepchainTaskCmsrunJobtypeSiteResults(ctx context.Context, c *gin.Context, configs models.Configuration, req models.DataTableRequest) models.DatatableBaseResponse {
	collection := mymongo.GetCollection(configs.CollectionNames.ScTaskCmsrunJobtypeSite)
	var cpuEffs []models.StepchainTaskWithCmsrunJobtypeSite

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

// ------------------------------- Common utils -----------------------------------------------------------------------

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
