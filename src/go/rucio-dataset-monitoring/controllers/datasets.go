package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/models"
	mymongo "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"net/http"
	"reflect"
)

var (
	// datasetsQueryHolder holds last search query to compare with next one
	datasetsQueryHolder bson.M

	// datasetsFilteredCountHolder holds filtered count value of last query
	datasetsFilteredCountHolder int64

	// datasetsUniqueSortColumn required for pagination in order
	datasetsUniqueSortColumn = "_id"
)

// GetDatasets datasets controller
func GetDatasets(collectionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.DataTableSearchBuilderRequest to the controller initializer and use same type in casting
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.DataTableRequest{})
		defer cancel()
		detailedDatasetsResp := getDatasetResults(ctx, c, collectionName, req.(models.DataTableRequest))
		c.JSON(http.StatusOK,
			detailedDatasetsResp,
		)
		return
	}
}

// getDatasetResults get query results efficiently
func getDatasetResults(ctx context.Context, c *gin.Context, datasetsCollectionName string, req models.DataTableRequest) models.DatatableBaseResponse {
	collection := mymongo.GetCollection(datasetsCollectionName)
	var datasets []models.Dataset

	// Should use SearchBuilderRequest query
	searchQuery := mymongo.SearchQueryForSearchBuilderRequest(&req.SearchBuilderRequest)
	sortQuery := mymongo.SortQueryBuilder(&req, datasetsUniqueSortColumn)
	length := req.Length
	skip := req.Start

	cursor, err := mymongo.GetFindQueryResults(ctx, collection, searchQuery, sortQuery, skip, length)
	if err != nil {
		utils.ErrorResponse(c, "Find query failed", err, "")
	}
	if err = cursor.All(ctx, &datasets); err != nil {
		utils.ErrorResponse(c, "datasets cursor failed", err, "")
	}

	filteredRecCount := length + skip + 1
	totalRecCount := getFilteredCount(ctx, c, collection, searchQuery, req.Draw)
	return models.DatatableBaseResponse{
		Draw:            req.Draw,
		RecordsTotal:    totalRecCount,
		RecordsFiltered: filteredRecCount,
		Data:            datasets,
	}
}

// getFilteredCount total document count in the datasetsDb
func getFilteredCount(ctx context.Context, c *gin.Context, collection *mongo.Collection, query bson.M, draw int) int64 {
	if draw < 1 {
		log.Fatalf("datatables draw value cannot be less than 1, it is: %d", draw)
	} else if (draw == 1) || (!reflect.DeepEqual(datasetsQueryHolder, query)) {
		// First opening of the page or search query is different from the previous one
		cnt, err := mymongo.GetCount(ctx, collection, query)
		if err != nil {
			utils.ErrorResponse(c, "TotalRecCount query failed", err, "")
		}
		datasetsFilteredCountHolder = cnt
		datasetsQueryHolder = query
		utils.InfoLogV1("filter query comparison: MIS-MATCH %s", nil)
	} else {
		// If search query is still same, count should be same, so return GFilteredCount
		utils.InfoLogV1("filter query comparison: MATCH %s", nil)
	}
	return datasetsFilteredCountHolder
}
