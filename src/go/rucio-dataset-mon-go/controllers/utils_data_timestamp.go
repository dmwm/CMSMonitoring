package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
	mymongo "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

// getDataSourceTimestamp gets the creation time of used data in MongoDB (sqoop dumps)
func getDataSourceTimestamp(ctx context.Context, c *gin.Context, sourceTimeCollectionName string) models.DataSourceTS {
	var dataTimestampLst []models.DataSourceTS
	collection := mymongo.GetCollection(mymongo.DBClient, sourceTimeCollectionName)
	failTimeStamp := models.DataSourceTS{CreatedAt: "0000-00-00"}

	cursor, err := mymongo.GetFindOnlyMatchResults(ctx, collection, bson.M{})
	if err != nil {
		utils.ErrorResponse(c, "Find query failed for getDataSourceTimestamp", err, "")
		return failTimeStamp
	}
	if err = cursor.All(ctx, &dataTimestampLst); err != nil {
		utils.ErrorResponse(c, "datasets cursor failed", err, "")
		return failTimeStamp
	}
	if len(dataTimestampLst) != 1 {
		utils.ErrorResponse(c, "More than 1 document in source_timestamp collection! ", nil, "")
		return failTimeStamp
	}
	// There should be only 1 document
	return dataTimestampLst[0]
}