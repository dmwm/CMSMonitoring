package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/models"
	mymongo "github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/mongo"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
)

// GetEachRseDetails controller that returns detailed dataset in TAPE or DISK
func GetEachRseDetails(collectionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			detailedRows []models.DetailedDataset
			err          error
			cursor       *mongo.Cursor
			sortType     = bson.D{bson.E{Key: "Type", Value: -1}} // TAPE first
		)
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.EachRseDetailsRequest{})
		defer cancel()

		// Cast interface to request
		r := req.(models.EachRseDetailsRequest)

		collection := mymongo.GetCollection(collectionName)
		if r.Type != "" {
			cursor, err = mymongo.GetFindOnlyMatchResults(ctx, collection, bson.M{"Dataset": r.Dataset, "Type": r.Type})
		} else {
			// in tape and disk
			cursor, err = mymongo.GetFindQueryResults(ctx, collection, bson.M{"Dataset": r.Dataset}, sortType, 0, 0)
		}

		if err != nil {
			utils.ErrorResponse(c, "Find query failed", err, "")
		}
		if err = cursor.All(ctx, &detailedRows); err != nil {
			utils.ErrorResponse(c, "single detailed dataset cursor failed", err, "")
		}

		c.HTML(http.StatusOK,
			"each_rse_details.tmpl",
			gin.H{"data": detailedRows},
		)
		return
	}
}
