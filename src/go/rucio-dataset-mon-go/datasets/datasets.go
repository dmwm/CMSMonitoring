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
	"reflect"
)

var (
	collectionName   = "datasets"
	collection       *mongo.Collection
	GQuery           bson.M // search query to store last query
	GFilteredCount   int64  // count number to store last query filtered count
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

// GetFilteredCount total document count in the datasetsDb
func GetFilteredCount(ctx context.Context, c *gin.Context, query bson.M, draw int) int64 {
	if draw < 1 {
		log.Fatalf("Datatables draw value cannot be less than 1, it is: %d", draw)
	} else if (draw == 1) || (!reflect.DeepEqual(GQuery, query)) {
		// First opening of the page or search query is different thand the previous one
		cnt, err := mymongo.GetCount(ctx, collection, query)
		if err != nil {
			utils.ErrorResponse(c, "TotalRecCount query failed", err, "")
		}
		GFilteredCount = cnt
		GQuery = query
		log.Println("[INFO] Filter query comparison: MIS-MATCH")
	} else {
		// If search query is still same, count should be same, so return GFilteredCount
		log.Println("[INFO] Filter query comparison: MATCH")
	}
	return GFilteredCount
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
	totalRecCount := GetFilteredCount(ctx, c, searchQuery, r.Draw)
	return models.DatatableDatasetsResponse{
		Draw:            r.Draw,
		RecordsTotal:    totalRecCount,
		RecordsFiltered: filteredRecCount,
		Data:            datasets,
	}
}
