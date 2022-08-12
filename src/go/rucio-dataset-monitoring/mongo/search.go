package mongo

import (
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/models"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
)

// SearchQueryForSearchBuilderRequest creates search query for datasets
func SearchQueryForSearchBuilderRequest(req *models.SearchBuilderRequest) bson.M {
	findQuery := bson.M{}
	utils.InfoLogV2("SearchBuilderRequest: %#v", req.String())

	// Check if SearchBuilderRequest object in the incoming request is empty
	if !reflect.DeepEqual(req, models.SearchBuilderRequest{}) {
		utils.InfoLogV2("SearchBuilder is not null")
		findQuery = utils.GetSearchBuilderBson(req)
	}
	utils.InfoLogV1("find query is : %#v", findQuery)
	return findQuery
}

// SearchQueryBuilderForCustomRequest creates search query for CustomRequest request
func SearchQueryBuilderForCustomRequest(customR *models.CustomRequest, prodAccounts *[]string) bson.M {
	findQuery := bson.M{}
	utils.InfoLogV2("CustomRequest query is : %#v", customR.String())
	if customR.Dataset != "" {
		findQuery["Dataset"] = primitive.Regex{Pattern: customR.Dataset, Options: "im"}
	}
	if customR.Rse != "" {
		findQuery["RSE"] = primitive.Regex{Pattern: customR.Rse, Options: "im"}
	}
	if customR.Tier != "" {
		findQuery["Tier"] = primitive.Regex{Pattern: customR.Tier, Options: "im"}
	}
	if customR.RseCountry != "" {
		findQuery["C"] = primitive.Regex{Pattern: customR.RseCountry, Options: "im"}
	}
	if customR.RseKind != "" {
		findQuery["RseKind"] = primitive.Regex{Pattern: customR.RseKind, Options: "im"}
	}
	if len(customR.RseType) == 1 {
		// If not 1, both DISK and TAPE
		findQuery["Type"] = customR.RseType[0]
	}
	if (len(customR.Accounts) != len(*prodAccounts)) && (len(customR.Accounts) > 0) {
		// Prod lock account search
		var accountsBson []bson.M
		for _, acc := range customR.Accounts {
			accountsBson = append(accountsBson, bson.M{"ProdAccts": primitive.Regex{Pattern: acc, Options: "im"}})
		}
		findQuery["$or"] = accountsBson
	}
	utils.InfoLogV1("find query is : %#v", findQuery)

	return findQuery
}
