package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
	mymongo "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
)

var detailedDsUniqueSortColumn = "_id"

// getDetailedDsResults get query results efficiently
func getDetailedDsResults(ctx context.Context, c *gin.Context, dtRequest models.DataTableCustomRequest) models.DetailedDatasetsResp {
	collection := mymongo.GetCollection(mymongo.DBClient, detailedDsCollectionName)
	var detailedDatasets []models.DetailedDs
	searchQuery := searchQueryBuilder(dtRequest)
	sortQuery := sortQueryBuilder(dtRequest.Orders, dtRequest.Columns, detailedDsUniqueSortColumn)
	length := dtRequest.Length
	skip := dtRequest.Start

	cursor, err := mymongo.GetFindQueryResults(ctx, collection, searchQuery, sortQuery, skip, length)
	if err != nil {
		utils.ErrorResponse(c, "Find query failed", err, "")
	}
	if err = cursor.All(ctx, &detailedDatasets); err != nil {
		utils.ErrorResponse(c, "detailed datasets cursor failed", err, "")
	}

	filteredRecCount := length + skip + 1
	return models.DetailedDatasetsResp{
		Draw:            dtRequest.Draw,
		RecordsTotal:    filteredRecCount,
		RecordsFiltered: filteredRecCount,
		Data:            detailedDatasets,
	}
}
