package mongo

// Copyright (c) 2023 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"github.com/dmwm/CMSMonitoring/cpueff-goweb/models"
	"github.com/dmwm/CMSMonitoring/cpueff-goweb/utils"
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
)

// SearchQueryForSearchBuilderRequest creates search query
func SearchQueryForSearchBuilderRequest(req *models.SearchBuilderRequest, sourceService string) bson.M {
	findQuery := bson.M{}
	utils.InfoLogV2("incoming SearchBuilderRequest: " + req.String())

	// Check if SearchBuilderRequest object in the incoming request is empty
	if !reflect.DeepEqual(req, models.SearchBuilderRequest{}) {
		utils.InfoLogV2("SearchBuilder is not null: " + req.String())
		findQuery = utils.GetSearchBuilderBson(req, sourceService)
	}
	utils.InfoLogV1("find query is : %#v", findQuery)
	return findQuery
}

// SortQueryBuilder creates sort object which support multiple column sort
func SortQueryBuilder(r *models.DataTableRequest, uniqueSortColumn string) bson.D {
	sortBson := bson.D{}
	for _, order := range r.Orders {
		intSortDirection := utils.ConvertOrderEnumToMongoInt(order.Dir)
		if intSortDirection != 0 {
			sortBson = append(sortBson, bson.E{
				Key:   r.Columns[order.Column].Data, // column name
				Value: intSortDirection,             // column direction as int value (0/1)
			})
		}
	}
	// Always add unique id column at the end to be able fetch non-unique columns in order
	sortBson = append(sortBson, bson.E{Key: uniqueSortColumn, Value: 1})
	return sortBson
}
