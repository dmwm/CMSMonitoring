package detailed_datasets

import (
	"context"
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
	detailedDsDb     = mymongo.GetCollection(mymongo.DBClient, "detailed_datasets")
	UniqueSortColumn = "_id"
	ProdAccounts     = []string{"transfer_ops", "wma_prod", "wmcore_output", "wmcore_transferor", "crab_tape_recall", "sync"}
)

// CreateSortBson creates sort object which support multiple column sort
func CreateSortBson(dtRequest models.DataTableCustomRequest) bson.D {
	orders := dtRequest.Orders
	columns := dtRequest.Columns
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
	// Always add unique id column at the end to be able to fetch non-unique columns in order
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
		findQuery["RSE"] = primitive.Regex{Pattern: customs.Rse, Options: "im"}
	}
	if customs.Tier != "" {
		findQuery["Tier"] = primitive.Regex{Pattern: customs.Tier, Options: "im"}
	}
	if customs.RseCountry != "" {
		findQuery["C"] = primitive.Regex{Pattern: customs.RseCountry, Options: "im"}
	}
	if customs.RseKind != "" {
		findQuery["RseKind"] = primitive.Regex{Pattern: customs.RseKind, Options: "im"}
	}
	if len(customs.RseType) == 1 {
		// If not 1, both DISK and TAPE
		findQuery["Type"] = customs.RseType[0]
	}
	if (len(customs.Accounts) != len(ProdAccounts)) && (len(customs.Accounts) > 0) {
		// Prod lock account search
		var accountsBson []bson.M
		for _, acc := range customs.Accounts {
			accountsBson = append(accountsBson, bson.M{"ProdAccts": primitive.Regex{Pattern: acc, Options: "im"}})
		}
		findQuery["$or"] = accountsBson
	}
	log.Printf("[INFO] Find Query is : %#v", findQuery)
	return findQuery
}

// GetFilteredRecCount filtered document count in the datasetsDb
func GetFilteredRecCount(ctx context.Context, c *gin.Context, findQuery bson.M) int64 {
	cnt, err := mymongo.GetCount(ctx, detailedDsDb, findQuery)
	if err != nil {
		utils.ErrorResponse(c, "FilteredRecCount query failed", err)
	}
	log.Printf("[INFO] Filtered Count %d", cnt)
	return cnt
}

// GetResults get query results efficiently
func GetResults(ctx context.Context, c *gin.Context, dtRequest models.DataTableCustomRequest) models.DatatableDetailedDsResponse {
	var detailedDatasets []models.DetailedDs
	searchQuery := CreateSearchBson(dtRequest)
	sortQuery := CreateSortBson(dtRequest)
	length := dtRequest.Length
	skip := dtRequest.Start

	detailedDsCursor, err := mymongo.GetFindQueryResults(ctx, detailedDsDb, searchQuery, sortQuery, skip, length)
	if err != nil {
		utils.ErrorResponse(c, "Find query failed", err)
	}
	for detailedDsCursor.Next(ctx) {
		var singleDetailedDs models.DetailedDs
		if err = detailedDsCursor.Decode(&singleDetailedDs); err != nil {
			utils.ErrorResponse(c, "detailedDsCursor.Decode failed", err)
		}
		detailedDatasets = append(detailedDatasets, singleDetailedDs)
	}

	//totalRecCount := GetTotalRecCount(ctx, c)
	filteredRecCount := GetFilteredRecCount(ctx, c, searchQuery)

	return models.DatatableDetailedDsResponse{
		Draw:            dtRequest.Draw,
		RecordsTotal:    filteredRecCount,
		RecordsFiltered: filteredRecCount,
		Data:            detailedDatasets,
	}
}
