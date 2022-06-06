package controllers

import (
    "context"
    "encoding/json"
    "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/configs"
    "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
    "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/responses"
    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "io"
    "log"
    "net/http"
    "strings"
    "time"
)

// connection instance for the "datasets" collection
var datasetCollection = configs.GetCollection(configs.DB, configs.GetEnvVar("COLLECTION_DATASETS"))

// MiddlewareReqHandler handles CORS and HTTP request settings for the context router
func MiddlewareReqHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        //c.Writer.Header().Set("Content-Type", "application/json")
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        c.Next()
    }
}

// getTotalRecCount total document count in the collection
func getTotalRecCount(ctx context.Context) int64 {
    countTotal, err := datasetCollection.CountDocuments(ctx, bson.M{})
    if err != nil {
        log.Fatal(err)
    }
    return countTotal
}

// convertOrderEnumToMongoInt converts DataTable enums ("asc" and "desc") to Mongo sorting integer definitions (1,-1)
func convertOrderEnumToMongoInt(dir string) int {
    switch strings.ToLower(dir) {
    case "asc":
        return 1
    case "desc":
        return -1
    default:
        return 0
    }
}

// generateSortOrder creates sort order for the Mongo query by iterating over DataTable json request
func generateSortOrder(dataTableRequest models.DataTableRequest) *options.FindOptions {
    orders := dataTableRequest.Orders
    columns := dataTableRequest.Columns
    sortOpts := options.Find()
    for _, order := range orders {
        intSortDirection := convertOrderEnumToMongoInt(order.Dir)
        if intSortDirection != 0 {
            sortOpts = sortOpts.SetSort(
                bson.D{{
                    Key:   columns[order.Column].Data, // column name
                    Value: intSortDirection,           // column direction as int value
                }},
            )
        }
    }
    return sortOpts
}

// generateFindQuery creates main search query using regex by default
func generateFindQuery(dataTableRequest models.DataTableRequest) []bson.M {
    // DataTable main search request struct
    dtMainSearch := dataTableRequest.Search
    var findQuery []bson.M
    findQuery = append(findQuery, bson.M{"dataset": primitive.Regex{Pattern: dtMainSearch.Value, Options: "im"}})
    // i: case insensitive, m: can use ^ and $. Ref: // https://www.mongodb.com/docs/v5.0/reference/operator/query/regex/
    // TODO add individual column search
    return findQuery
}

// GetDatasets controller that returns datasets according to DataTable request json
func GetDatasets() gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx, cancel := context.WithTimeout(context.Background(), time.Duration(configs.EnvConnTimeout())*time.Second)
        var datasets []models.Dataset
        defer cancel()

        // === ~~~~~~ Decode incoming DataTable request json ~~~~~~ ===
        log.SetFlags(log.LstdFlags | log.Lshortfile)
        body, err := io.ReadAll(c.Request.Body)
        if err != nil {
            log.Fatal(err)
        }
        log.Printf("Request Body:\n%#v\n", string(body))
        var dataTableRequest models.DataTableRequest
        if err := json.Unmarshal(body, &dataTableRequest); err != nil {
            log.Fatal(err)
        }
        // === ~~~~~~ ------------------------------ ~~~~~~ ===

        sortOptsOfFind := generateSortOrder(dataTableRequest)
        findQuery := generateFindQuery(dataTableRequest)

        // Count filter query record count
        filteredRecCount, err := datasetCollection.CountDocuments(ctx, bson.M{"$and": findQuery})
        if err != nil {
            log.Fatal(err)
        }

        // Get main results
        datasetResults, err := datasetCollection.Find(ctx, bson.M{"$and": findQuery}, sortOptsOfFind)
        if err != nil {
            log.Fatal(err)
        }

        // reading from the db in an optimal way
        defer func(results *mongo.Cursor, ctx context.Context) {
            if err := results.Close(ctx); err != nil {
                log.Fatal(err)
            }
        }(datasetResults, ctx)

        for datasetResults.Next(ctx) {
            var singleDataset models.Dataset
            if err = datasetResults.Decode(&singleDataset); err != nil {
                c.JSON(http.StatusInternalServerError,
                    responses.DtErrorResponse{
                        Status:  http.StatusInternalServerError,
                        Message: "error",
                        Data:    map[string]interface{}{"data": err.Error()},
                    })
            }
            datasets = append(datasets, singleDataset)
        }

        // Send response in DataTable required format
        //  - Need to return exactly same "Draw" value that DataTable sent in incoming request
        c.JSON(http.StatusOK,
            models.DatatableResponse{
                Draw:            dataTableRequest.Draw,
                RecordsTotal:    getTotalRecCount(ctx),
                RecordsFiltered: filteredRecCount,
                Data:            datasets,
            },
        )
	}
}
