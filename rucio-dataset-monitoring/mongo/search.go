package mongo

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/models"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/utils"
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
)

// SearchQueryForSearchBuilderRequest creates search query for datasets
func SearchQueryForSearchBuilderRequest(req *models.SearchBuilderRequest) bson.M {
	findQuery := bson.M{}
	utils.InfoLogV2("incoming SearchBuilderRequest: " + req.String())

	// Check if SearchBuilderRequest object in the incoming request is empty
	if !reflect.DeepEqual(req, models.SearchBuilderRequest{}) {
		utils.InfoLogV2("SearchBuilder is not null: " + req.String())
		findQuery = utils.GetSearchBuilderBson(req)
	}
	utils.InfoLogV1("find query is : %#v", findQuery)
	return findQuery
}
