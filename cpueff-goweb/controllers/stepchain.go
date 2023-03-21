package controllers

import (
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

// --------------------------- Row Site detail controller ------------------------------------------

// StepchainRowSiteDetailCtrl row details template render controller: task row details API: returns a single Task's Cmsrun+Jobtype+Site cpu efficiencies
func StepchainRowSiteDetailCtrl(configs models.Configuration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var detailedRows []models.StepchainTaskWithCmsrunJobtypeSite
		var err error
		var cursor *mongo.Cursor
		var sortType = bson.D{bson.E{Key: "StepName", Value: 1}}
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.StepchainRowDetailRequest{})
		defer cancel()

		query := bson.M{}
		// Cast interface to request
		r := req.(models.StepchainRowDetailRequest)
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
			"stepchain_row_site_detail.tmpl",
			gin.H{"data": detailedRows},
		)
		return
	}
}

// "WMArchive(MONIT)" links to 'Links' columns. Check configs.json
// utilScTaskCmsrunJobtypeSiteExternalLinks creates external links for the task
func utilScTaskCmsrunJobtypeSiteExternalLinks(cpuEffs []models.StepchainTaskWithCmsrunJobtypeSite, dataTimestamp models.DataSourceTS, links models.ExternalLinks) {
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

// --------------------------- Row cmsRun detail controller ------------------------------------------

// StepchainRowCmsrunDetailCtrl row details template render controller: task row details API: returns a single Task's Cmsrun+Jobtype cpu efficiencies
func StepchainRowCmsrunDetailCtrl(configs models.Configuration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var detailedCmsruns []models.StepchainTaskWithCmsrunJobtype
		var err error
		var cursor *mongo.Cursor
		var sortType = bson.D{bson.E{Key: "StepName", Value: 1}}
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.StepchainRowDetailRequest{})
		defer cancel()

		query := bson.M{}
		// Cast interface to request
		r := req.(models.StepchainRowDetailRequest)
		if r.Task != "" {
			query["Task"] = r.Task
		}
		if r.StepName != "" {
			query["StepName"] = r.StepName
		}
		if r.JobType != "" {
			query["JobType"] = r.JobType
		}

		collectionCmsrunJobtype := mymongo.GetCollection(configs.CollectionNames.ScTaskCmsrunJobtype)
		cursor, err = mymongo.GetFindQueryResults(ctx, collectionCmsrunJobtype, query, sortType, 0, 0)
		if err != nil {
			utils.ErrorResponse(c, "Find query failed", err, "")
		}
		if err = cursor.All(ctx, &detailedCmsruns); err != nil {
			utils.ErrorResponse(c, "single detailed task details cursor failed", err, "")
		}

		// Get processed data time period: start date and end date that
		dataTimestamp := GetDataSourceTimestamp(ctx, c, configs.CollectionNames.DatasourceTimestamp)
		// Add Links of other services for the workflow/task to the 'Links' column
		utilScTaskCmsrunJobtypeExternalLinks(detailedCmsruns, dataTimestamp, configs.ExternalLinks)

		c.HTML(http.StatusOK,
			"stepchain_row_cmsrun_detail.tmpl",
			gin.H{
				"data": detailedCmsruns,
			},
		)
		return
	}
}
