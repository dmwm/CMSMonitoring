package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
	mymongo "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/search_builder"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"reflect"
	"time"
)

// sortQueryBuilder creates sort object which support multiple column sort
func sortQueryBuilder(orders []models.DTReqOrder, columns []models.DTReqColumn, UniqueSortColumn string) bson.D {
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

// searchQueryBuilder creates search query for datasets
//   - models.DataTableSearchBuilderRequest request comes for /api/datasets, so uses search builder
//   - models.DataTableCustomRequest request comes for /api/rse-details, so requires custom queries
func searchQueryBuilder(req interface{}) bson.M {
	findQuery := bson.M{}
	switch r := req.(type) {
	case models.DataTableSearchBuilderRequest:
		utils.InfoLogV2("r.SearchBuilder: %#v", r.SearchBuilder)

		// Check if SearchBuilder object in the incoming request is empty
		if !reflect.DeepEqual(r.SearchBuilder, models.SearchBuilderRequest{}) {
			utils.InfoLogV2("searchBuilder is not null")
			findQuery = search_builder.GetSearchBuilderBson(r.SearchBuilder)
		}
	case models.DataTableCustomRequest:
		customs := r.Custom
		utils.InfoLogV2("Custom query is : %#v", customs)
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
		if (len(customs.Accounts) != len(prodAccounts)) && (len(customs.Accounts) > 0) {
			// Prod lock account search
			var accountsBson []bson.M
			for _, acc := range customs.Accounts {
				accountsBson = append(accountsBson, bson.M{"ProdAccts": primitive.Regex{Pattern: acc, Options: "im"}})
			}
			findQuery["$or"] = accountsBson
		}
	}
	utils.InfoLogV1("Find Query is : %#v", findQuery)

	return findQuery
}

// getSingleDatasetResults returns single dataset filtered with name
func getSingleDatasetResults(ctx context.Context, c *gin.Context, r models.SingleDetailedDatasetsRequest) []models.DetailedDataset {
	collection := mymongo.GetCollection(mymongo.DBClient, detailedDsCollectionName)
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

// initializeController initialize controller requirements
//   initialize context, bind request json for the controller, prints initial logs, gets start time etc.
func initializeController(c *gin.Context, req interface{}) (context.Context, context.CancelFunc, time.Time, interface{}) {
	verboseControllerInitLog(c)
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), mymongo.Timeout)

	req, err := bindRequest(c, req)
	if err != nil {
		tempReqBody, exists := c.Get(gin.BodyBytesKey)
		if !exists {
			utils.ErrorResponse(c, "Bad request", err, "Request body does not exist in Gin context")
		} else {
			utils.ErrorResponse(c, "Bad request", err, string(tempReqBody.([]byte)))
		}
	}
	return ctx, cancel, start, req
}

// bindRequest binds request according to provided type
func bindRequest(c *gin.Context, req interface{}) (any, error) {
	// Get request json with validation
	switch r := req.(type) {
	case models.DataTableSearchBuilderRequest:
		err := c.ShouldBindBodyWith(&r, binding.JSON)
		utils.InfoLogV2("Incoming request body bind to: %s", "DataTableSearchBuilderRequest")
		return r, err
	case models.DataTableCustomRequest:
		err := c.ShouldBindBodyWith(&r, binding.JSON)
		utils.InfoLogV2("Incoming request body bind to: %s", "DataTableCustomRequest")
		return r, err
	case models.ShortUrlRequest:
		err := c.ShouldBindBodyWith(&r, binding.JSON)
		utils.InfoLogV2("Incoming request body bind to: %s", "ShortUrlRequest")
		return r, err
	case models.SingleDetailedDatasetsRequest:
		err := c.ShouldBindBodyWith(&r, binding.JSON)
		utils.InfoLogV2("Incoming request body bind to: %s", "SingleDetailedDatasetsRequest")
		return r, err
	default:
		utils.ErrorLog("Unknown request struct, it did not match. Req: %#v", req)
		return nil, errors.New("unknown request struct, no match in switch case")
	}
}

// verboseControllerInitLog prints logs of incoming request's url path
func verboseControllerInitLog(c *gin.Context) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	utils.InfoLogV0("Incoming request to: %s", c.FullPath())
}

// verboseControllerOutLog prints debug logs after controller processed the api call
func verboseControllerOutLog(start time.Time, name string, req any, data any) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if Verbose > 0 {
		elapsed := time.Since(start)
		req, err := json.Marshal(req)
		if err != nil {
			log.Printf("[ERROR] ------ Cannot marshall request, err:%s", err)
		} else {
			r := string(req)
			if Verbose >= 2 {
				// Response returns all query results, its verbosity should be at least 2
				data, err1 := json.Marshal(data)
				if err1 != nil {
					log.Printf("[ERROR] ------ Cannot marshall additional verbose log data, err:%s", err1)
				} else {
					d := string(data)
					log.Printf("[DEBUG] ------\n -Query time [%s] : %s\n\n -Request body: %s\n\n -Response: %s\n\n", name, elapsed, r, d)
				}
			} else {
				log.Printf("[INFO] ------\n -Query time [%s] : %s\n\n -Request body: %s\n\n -Response: %#v\n\n", name, elapsed, req, nil)
			}
		}
	}
}
