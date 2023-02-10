package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/models"
	mymongo "github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/mongo"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"net/http"
	"reflect"
)

var (
	// mainDatasetsQueryHolder holds last search query to compare with next one
	mainDatasetsQueryHolder bson.M

	// mainDatasetsFilteredCountHolder holds filtered count value of last query
	mainDatasetsFilteredCountHolder int64

	// mainDatasetsUniqueSortColumn required for pagination in order
	mainDatasetsUniqueSortColumn = "_id"
)

// GetMainDatasets controller
func GetMainDatasets(collectionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.DataTableSearchBuilderRequest to the controller initializer and use same type in casting
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.DataTableRequest{})
		defer cancel()

		c.JSON(http.StatusOK, getMainDatasetResults(ctx, c, collectionName, req.(models.DataTableRequest)))
		return
	}
}

// getMainDatasetResults get query results efficiently
func getMainDatasetResults(ctx context.Context, c *gin.Context, collectionName string, req models.DataTableRequest) models.DatatableBaseResponse {
	collection := mymongo.GetCollection(collectionName)
	var mainDatasets []models.MainDataset

	// Should use SearchBuilderRequest query
	searchQuery := mymongo.SearchQueryForSearchBuilderRequest(&req.SearchBuilderRequest)
	sortQuery := mymongo.SortQueryBuilder(&req, mainDatasetsUniqueSortColumn)
	length := req.Length
	skip := req.Start

	cursor, err := mymongo.GetFindQueryResults(ctx, collection, searchQuery, sortQuery, skip, length)
	if err != nil {
		utils.ErrorResponse(c, "Find query failed", err, "")
	}
	if err = cursor.All(ctx, &mainDatasets); err != nil {
		utils.ErrorResponse(c, "mainDatasets cursor failed", err, "")
	}

	totalRecCount := getFilteredCount(ctx, c, collection, searchQuery, req.Draw)
	filteredRecCount := totalRecCount
	return models.DatatableBaseResponse{
		Draw:            req.Draw,
		RecordsTotal:    totalRecCount,
		RecordsFiltered: filteredRecCount,
		Data:            mainDatasets,
	}
}

// getFilteredCount total document count of the filter result in the main-datasets DB
func getFilteredCount(ctx context.Context, c *gin.Context, collection *mongo.Collection, query bson.M, draw int) int64 {
	if draw < 1 {
		log.Fatalf("datatables draw value cannot be less than 1, it is: %d", draw)
	} else if (draw == 1) || (!reflect.DeepEqual(mainDatasetsQueryHolder, query)) {
		// First opening of the page or search query is different from the previous one
		cnt, err := mymongo.GetCount(ctx, collection, query)
		if err != nil {
			utils.ErrorResponse(c, "TotalRecCount query failed", err, "")
		}
		mainDatasetsFilteredCountHolder = cnt
		mainDatasetsQueryHolder = query
		utils.InfoLogV1("filter query comparison: MIS-MATCH %s", nil)
	} else {
		// If search query is still same, count should be same, so return GFilteredCount
		utils.InfoLogV1("filter query comparison: MATCH %s", nil)
	}
	return mainDatasetsFilteredCountHolder
}
