package datasets

import (
	"context"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
	mymongo "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
)

var (
	collectionName   = "datasets"
	collection       *mongo.Collection
	TotalRecCountDs  int64
	UniqueSortColumn = "_id"
)

// CreateSortBson creates sort object which support multiple column sort
func CreateSortBson(dataTableRequest models.DataTableCustomRequest) bson.D {
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

// CreateSearchBson creates search query
func CreateSearchBson(dtRequest models.DataTableCustomRequest) bson.M {
	findQuery := bson.M{}

	customs := dtRequest.Custom
	if customs.Dataset != "" {
		findQuery["Dataset"] = primitive.Regex{Pattern: customs.Dataset, Options: "im"}
	}
	if customs.Rse != "" {
		findQuery["RSEs"] = primitive.Regex{Pattern: customs.Rse, Options: "im"}
	}
	if len(customs.RseType) == 1 {
		// If not 1, both DISK and TAPE
		findQuery["RseType"] = customs.RseType[0]
	}

	log.Printf("[INFO] Find Query is : %#v", findQuery)
	return findQuery
}

// GetTotalRecCount total document count in the datasetsDb
func GetTotalRecCount(ctx context.Context, c *gin.Context) int64 {
	if TotalRecCountDs == 0 {
		countTotal, err := mymongo.GetCount(ctx, collection, bson.M{"RseType": "DISK"})
		if err != nil {
			utils.ErrorResponse(c, "TotalRecCount query failed", err, "")
		}
		TotalRecCountDs = countTotal
	}
	log.Printf("[INFO] Total Count %d", TotalRecCountDs)
	return TotalRecCountDs
}

// GetResults get query results efficiently
func GetResults(ctx context.Context, c *gin.Context, r models.DataTableCustomRequest) models.DatatableDatasetsResponse {
	collection = mymongo.GetCollection(mymongo.DBClient, collectionName)
	var datasets []models.Dataset
	searchQuery := CreateSearchBson(r)
	sortQuery := CreateSortBson(r)
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
	totalRecCount := GetTotalRecCount(ctx, c)
	return models.DatatableDatasetsResponse{
		Draw:            r.Draw,
		RecordsTotal:    totalRecCount,
		RecordsFiltered: filteredRecCount,
		Data:            datasets,
	}
}
