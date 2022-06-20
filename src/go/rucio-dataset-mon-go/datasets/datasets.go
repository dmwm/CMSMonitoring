package datasets

import (
	"context"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/configs"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
	mymongo "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
)

var (
	// connection instance for the "datasets" collection
	datasetsDb       = mymongo.GetCollection(mymongo.DBClient, configs.GetEnvVar("COLLECTION_DATASETS"))
	TotalRecCountDs  int64
	UniqueSortColumn = "_id"
)

// GetSortBson creates sort object which support multiple column sort
func GetSortBson(dataTableRequest models.DataTableRequest) bson.D {
	orders := dataTableRequest.Orders
	columns := dataTableRequest.Columns
	sortBson := bson.D{}
	for _, order := range orders {
		intSortDirection := utils.ConvertOrderEnumToMongoInt(order.Dir)
		if intSortDirection != 0 {
			sortBson = append(sortBson, bson.E{
				Key:   columns[order.Column].Data, // column name
				Value: intSortDirection,           // column direction as int value (0/1)
			})
		}
	}
	// Always add unique id column at the end to be able fetch non-unique columns in order
	sortBson = append(sortBson, bson.E{Key: UniqueSortColumn, Value: 1})
	return sortBson
}

// GetSearchBson creates main search query using regex by default
func GetSearchBson(dataTableRequest models.DataTableRequest) bson.M {
	// Main search(dataTableRequest.Search) is disabled, only individual column search will be applied
	findQuery := bson.M{}
	for _, column := range dataTableRequest.Columns {
		if column.Search.Value != "" {
			// Add k(column name): v(column search text) pairs to search bson.M
			findQuery[column.Data] = primitive.Regex{Pattern: column.Search.Value, Options: "im"}
		}
	}
	log.Printf("[INFO] Find Query is : %#v", findQuery)
	return findQuery
}

// GetTotalRecCount total document count in the datasetsDb
func GetTotalRecCount(ctx context.Context, c *gin.Context) int64 {
	if TotalRecCountDs == 0 {
		countTotal, err := mymongo.GetCount(ctx, datasetsDb, bson.M{})
		if err != nil {
			utils.ErrorResponse(c, "TotalRecCount query failed", err)
		}
		TotalRecCountDs = countTotal
	}
	log.Printf("[INFO] Total Count %d", TotalRecCountDs)
	return TotalRecCountDs
}

// GetFilteredRecCount filtered document count in the datasetsDb
func GetFilteredRecCount(ctx context.Context, c *gin.Context, findQuery bson.M) int64 {
	filteredRecCount, err := mymongo.GetCount(ctx, datasetsDb, findQuery)
	if err != nil {
		utils.ErrorResponse(c, "FilteredRecCount query failed", err)
	}
	return filteredRecCount
}

// GetResults get query results efficiently
func GetResults(ctx context.Context, c *gin.Context, dataTableRequest models.DataTableRequest) []models.Dataset {
	var datasets []models.Dataset
	searchQuery := GetSearchBson(dataTableRequest)
	sortQuery := GetSortBson(dataTableRequest)
	length := dataTableRequest.Length
	skip := dataTableRequest.Start

	datasetsCursor, err := mymongo.GetAggQueryResults(ctx, datasetsDb, searchQuery, sortQuery, skip, length)
	if err != nil {
		utils.ErrorResponse(c, "datasetsDb.Find query failed", err)
	}
	for datasetsCursor.Next(ctx) {
		var singleDataset models.Dataset
		if err = datasetsCursor.Decode(&singleDataset); err != nil {
			utils.ErrorResponse(c, "datasetResults.Decode failed", err)
		}
		datasets = append(datasets, singleDataset)
	}
	return datasets
}
