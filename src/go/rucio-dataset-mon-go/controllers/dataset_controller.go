package controllers

import (
	"context"
	"encoding/json"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/configs"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"log"
	"net/http"
	"time"
)

var (
	// connection instance for the "datasets" datasetsDb
	datasetsDb          = mongo.GetCollection(mongo.DBClient, configs.GetEnvVar("COLLECTION_DATASETS"))
	GlobalTotalRecCount int64
)

// getRequestBody parse datatables request
func getRequestBody(c *gin.Context) models.DataTableRequest {
	// === ~~~~~~ Decode incoming DataTable request json ~~~~~~ ===
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.ErrorResponse(c, "Request body read failed", err)
	}
	log.Printf("Request Body:\n%#v\n", string(body))
	var dataTableRequest models.DataTableRequest
	if err := json.Unmarshal(body, &dataTableRequest); err != nil {
		utils.ErrorResponse(c, "Unmarshal request body failed", err)
	}
	return dataTableRequest
}

// getSortBson creates sort object which support multiple column sort
func getSortBson(dataTableRequest models.DataTableRequest) bson.D {
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
	sortBson = append(sortBson, bson.E{Key: "_id", Value: 1})
	return sortBson
}

// getSearchBson creates main search query using regex by default
func getSearchBson(dataTableRequest models.DataTableRequest) bson.M {
	// DataTable main search request struct
	dtMainSearch := dataTableRequest.Search
	if dtMainSearch.Value == "" {
		return bson.M{}
	} else {
		log.Printf("[INFO] dtMainSearch.Value is : %s", dtMainSearch.Value)
		// TODO add individual column search
		return bson.M{"$and": bson.M{"Dataset": primitive.Regex{Pattern: dtMainSearch.Value, Options: "im"}}}
	}
}

// getTotalRecCount total document count in the datasetsDb
func getTotalRecCount(ctx context.Context, c *gin.Context) int64 {
	if GlobalTotalRecCount == 0 {
		countTotal, err := mongo.GetCount(ctx, datasetsDb, bson.M{})
		if err != nil {
			utils.ErrorResponse(c, "TotalRecCount query failed", err)
		}
		GlobalTotalRecCount = countTotal
	}
	log.Printf("[INFO] Total Count %d", GlobalTotalRecCount)
	return GlobalTotalRecCount
}

// getFilteredRecCount filtered document count in the datasetsDb
func getFilteredRecCount(ctx context.Context, c *gin.Context, findQuery bson.M) int64 {
	filteredRecCount, err := mongo.GetCount(ctx, datasetsDb, findQuery)
	if err != nil {
		utils.ErrorResponse(c, "FilteredRecCount query failed", err)
	}
	return filteredRecCount
}

// getQueryResults get query results efficiently
func getQueryResults(ctx context.Context, c *gin.Context, dataTableRequest models.DataTableRequest) []models.Dataset {
	var datasets []models.Dataset
	searchQuery := getSearchBson(dataTableRequest)
	sortQuery := getSortBson(dataTableRequest)
	limit := dataTableRequest.Length
	skip := dataTableRequest.Start

	datasetsCursor, err := mongo.GetResults(ctx, datasetsDb, searchQuery, sortQuery, skip, limit)
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

// GetDatasets controller that returns datasets according to DataTable request json
func GetDatasets() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(configs.EnvConnTimeout())*time.Second)
		defer cancel()
		dataTableRequest := getRequestBody(c)

		totalRecCount := getTotalRecCount(ctx, c)
		filteredRecCount := getFilteredRecCount(ctx, c, getSearchBson(dataTableRequest))
		datasets := getQueryResults(ctx, c, dataTableRequest)

		// Send response in DataTable required format
		//  - Need to return exactly same "Draw" value that DataTable sent in incoming request
		c.JSON(http.StatusOK,
			models.DatatableResponse{
				Draw:            dataTableRequest.Draw,
				RecordsTotal:    totalRecCount,
				RecordsFiltered: filteredRecCount,
				Data:            datasets,
			},
		)
	}
}
