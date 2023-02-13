package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/models"
	mymongo "github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/mongo"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

// GetDataSourceTimestamp gets the creation time of used data in MongoDB (sqoop dumps)
func GetDataSourceTimestamp(ctx context.Context, c *gin.Context, sourceTimeCollectionName string) models.DataSourceTS {
	var dataTimestamp models.DataSourceTS
	collection := mymongo.GetCollection(sourceTimeCollectionName)
	failTimeStamp := models.DataSourceTS{CreatedAt: "0000-00-00"}
	if err := mymongo.GetFindOneResult(ctx, collection, bson.M{}).Decode(&dataTimestamp); err != nil {
		utils.ErrorResponse(c, "Find query failed for getDataSourceTimestamp: ", err, "")
		return failTimeStamp
	}
	return dataTimestamp
}
