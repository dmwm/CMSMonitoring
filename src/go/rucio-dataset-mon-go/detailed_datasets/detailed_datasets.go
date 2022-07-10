package detailed_datasets

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

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
	collectionName = "detailed_datasets"
	collection     *mongo.Collection
	ProdAccounts   = []string{"transfer_ops", "wma_prod", "wmcore_output", "wmcore_transferor", "crab_tape_recall", "sync"}
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

// GetResults get query results efficiently
func GetResults(ctx context.Context, c *gin.Context, dtRequest models.DataTableCustomRequest) models.DatatableDetailedDsResponse {
	collection = mymongo.GetCollection(mymongo.DBClient, collectionName)
	var detailedDatasets []models.DetailedDs
	searchQuery := CreateSearchBson(dtRequest)
	sortQuery := CreateSortBson(dtRequest)
	length := dtRequest.Length
	skip := dtRequest.Start

	cursor, err := mymongo.GetFindQueryResults(ctx, collection, searchQuery, sortQuery, skip, length)
	if err != nil {
		utils.ErrorResponse(c, "Find query failed", err, "")
	}
	if err = cursor.All(ctx, &detailedDatasets); err != nil {
		utils.ErrorResponse(c, "detailed datasets cursor failed", err, "")
	}

	filteredRecCount := length + skip + 1
	return models.DatatableDetailedDsResponse{
		Draw:            dtRequest.Draw,
		RecordsTotal:    filteredRecCount,
		RecordsFiltered: filteredRecCount,
		Data:            detailedDatasets,
	}
}

// GetSingleDataset returns single dataset filtered with name
func GetSingleDataset(ctx context.Context, c *gin.Context, r models.SingleDetailedDatasetsRequest) []models.DetailedDataset {
	collection = mymongo.GetCollection(mymongo.DBClient, collectionName)
	var rows []models.DetailedDataset
	cursor, err := mymongo.GetFindOnlyMatchResults(ctx, collection, bson.M{"Dataset": r.Dataset, "Type": r.Type})
	if err != nil {
		utils.ErrorResponse(c, "Find query failed", err, "")
	}
	if err = cursor.All(ctx, &rows); err != nil {
		utils.ErrorResponse(c, "single detailed dataset cursor failed", err, "")
	}
	return rows
}
