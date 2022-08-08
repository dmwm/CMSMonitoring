package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
	mymongo "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

var (
	// detailedDsUniqueSortColumn required for pagination in order
	detailedDsUniqueSortColumn = "_id"
)

// GetDetailedDs controller that returns datasets according to DataTable request json
func GetDetailedDs(collectionName string, prodLockAccounts *[]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.DataTableCustomRequest to the controller initializer and use same type in casting
		ctx, cancel, start, req := InitializeController(c, models.DataTableRequest{})
		defer cancel()
		detailedDatasetsResp := getDetailedDsResults(ctx, c, collectionName, prodLockAccounts, req.(models.DataTableRequest))
		c.JSON(http.StatusOK,
			detailedDatasetsResp,
		)
		VerboseControllerOutLog(start, "GetDetailedDs", req, detailedDatasetsResp)
		return
	}
}

// getDetailedDsResults get query results efficiently
func getDetailedDsResults(ctx context.Context, c *gin.Context, collectionName string, prodLockAccounts *[]string, req models.DataTableRequest) models.DatatableBaseResponse {
	collection := mymongo.GetCollection(collectionName)
	var detailedDatasets []models.DetailedDataset

	// Should use Custom Request for search query
	searchQuery := mymongo.SearchQueryBuilderForCustomRequest(&req.Custom, prodLockAccounts)
	sortQuery := mymongo.SortQueryBuilder(&req, detailedDsUniqueSortColumn)
	length := req.Length
	skip := req.Start

	cursor, err := mymongo.GetFindQueryResults(ctx, collection, searchQuery, sortQuery, skip, length)
	if err != nil {
		utils.ErrorResponse(c, "Find query failed", err, "")
	}
	if err = cursor.All(ctx, &detailedDatasets); err != nil {
		utils.ErrorResponse(c, "detailed datasets cursor failed", err, "")
	}

	filteredRecCount := length + skip + 1
	return models.DatatableBaseResponse{
		Draw:            req.Draw,
		RecordsTotal:    filteredRecCount,
		RecordsFiltered: filteredRecCount,
		Data:            detailedDatasets,
	}
}
