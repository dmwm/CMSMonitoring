package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/models"
	mymongo "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
)

// GetSingleDetailedDs controller that returns detailed dataset in TAPE or DISK
func GetSingleDetailedDs(collectionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel, start, req := InitializeCtxAndBindRequestBody(c, models.SingleDetailedDatasetsRequest{})
		defer cancel()

		// Cast interface to request
		r := req.(models.SingleDetailedDatasetsRequest)

		collection := mymongo.GetCollection(collectionName)

		var detailedRows []models.DetailedDataset
		cursor, err := mymongo.GetFindOnlyMatchResults(ctx, collection, bson.M{"Dataset": r.Dataset, "Type": r.Type})
		if err != nil {
			utils.ErrorResponse(c, "Find query failed", err, "")
		}
		if err = cursor.All(ctx, &detailedRows); err != nil {
			utils.ErrorResponse(c, "single detailed dataset cursor failed", err, "")
		}

		c.HTML(http.StatusOK,
			"rse_detail_table.tmpl",
			gin.H{"data": detailedRows},
		)
		VerboseControllerOutLog(start, "GetSingleDetailedDs", req, detailedRows)
		return
	}
}
