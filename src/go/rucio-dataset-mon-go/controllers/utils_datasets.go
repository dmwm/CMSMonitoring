// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// Package controllers contains all get functions of controllers
package controllers

import (
	"context"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
	mymongo "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"reflect"
)

var (
	datasetsQueryHolder         bson.M // search query to store last query
	datasetsFilteredCountHolder int64  // count number to store last query filtered count
	datasetsUniqueSortColumn    = "_id"
)

// getDatasetResults get query results efficiently
func getDatasetResults(ctx context.Context, c *gin.Context, datasetsCollectionName string, r models.DataTableSearchBuilderRequest) models.DatasetsResp {
	collection := mymongo.GetCollection(mymongo.DBClient, datasetsCollectionName)
	var datasets []models.Dataset
	searchQuery := searchQueryBuilder(r)
	sortQuery := sortQueryBuilder(r.Orders, r.Columns, datasetsUniqueSortColumn)
	length := r.Length
	skip := r.Start

	cursor, err := mymongo.GetFindQueryResults(ctx, collection, searchQuery, sortQuery, skip, length)
	if err != nil {
		utils.ErrorResponse(c, "datasetsDb.Find query failed", err, "")
	}
	if err = cursor.All(ctx, &datasets); err != nil {
		utils.ErrorResponse(c, "datasets cursor failed", err, "")
	}

	filteredRecCount := length + skip + 1
	totalRecCount := getFilteredCount(ctx, c, collection, searchQuery, r.Draw)
	return models.DatasetsResp{
		Draw:            r.Draw,
		RecordsTotal:    totalRecCount,
		RecordsFiltered: filteredRecCount,
		Data:            datasets,
	}
}

// getFilteredCount total document count in the datasetsDb
func getFilteredCount(ctx context.Context, c *gin.Context, collection *mongo.Collection, query bson.M, draw int) int64 {
	if draw < 1 {
		log.Fatalf("Datatables draw value cannot be less than 1, it is: %d", draw)
	} else if (draw == 1) || (!reflect.DeepEqual(datasetsQueryHolder, query)) {
		// First opening of the page or search query is different from the previous one
		cnt, err := mymongo.GetCount(ctx, collection, query)
		if err != nil {
			utils.ErrorResponse(c, "TotalRecCount query failed", err, "")
		}
		datasetsFilteredCountHolder = cnt
		datasetsQueryHolder = query
		utils.InfoLogV1("Filter query comparison: MIS-MATCH", nil)
	} else {
		// If search query is still same, count should be same, so return GFilteredCount
		utils.InfoLogV1("Filter query comparison: MATCH")
	}
	return datasetsFilteredCountHolder
}
