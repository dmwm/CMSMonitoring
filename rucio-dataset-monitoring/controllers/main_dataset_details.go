package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/models"
	mymongo "github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/mongo"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
)

// GetMainDatasetDetails controller that returns detailed dataset in TAPE or DISK
func GetMainDatasetDetails(collectionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.MainDatasetDetailsRequest{})
		defer cancel()

		// Cast interface to request
		r := req.(models.MainDatasetDetailsRequest)

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
			"main_dataset_details.tmpl",
			gin.H{"data": detailedRows},
		)
		return
	}
}
