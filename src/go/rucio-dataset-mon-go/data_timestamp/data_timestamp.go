package data_timestamp

import (
	"context"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
	mymongo "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

var collectionName = "source_timestamp"

// GetDataSourceTimestamp gets the creation time of used data in MongoDB (sqoop dumps)
func GetDataSourceTimestamp(ctx context.Context, c *gin.Context) models.DataSourceTS {
	collection := mymongo.GetCollection(mymongo.DBClient, collectionName)
	var dataTimestampLst []models.DataSourceTS

	cursor, err := mymongo.GetFindOnlyMatchResults(ctx, collection, bson.M{})
	if err != nil {
		utils.ErrorResponse(c, "datasetsDb.Find query failed for DataTimeStamp", err, "")
	}
	if err = cursor.All(ctx, &dataTimestampLst); err != nil {
		utils.ErrorResponse(c, "datasets cursor failed", err, "")
	}
	// There should be only 1 document
	return dataTimestampLst[0]
}
