// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

/*
Package short_url This controller creates short url using incoming request's MD5 hash and MongoDB collection to store hash and request match

In Short:
    - Create a MD5 hash from incoming request
    - Store hash id - request couple in MongoDB collection
    - When a user provide hash id, return a page which is restored from the matching request and state with hash id in MongoDB

How to create short url works:
    - When user clicked the "Copy state link to clipboard" button, DataTables sends request to short_url API
    - You can see incoming request structure: models.ShortUrlRequest
    - This JSOn request converted to JSON string and an MD5 hash is created from this string in `getRequestHash` function
    - This hash id is checked if there is an already matching hash id in MongoDB `short_url` collection
    - If not, `getShortUrl` function inserts models.ShortUrl to MongoDB.
    - models.ShortUrl contains all we need to restore the DataTables page state: shared query and shared state(HTML/CSS)
    - After insertion, hash id can be shared safely with other users.

How to use short url:
    - User can use shared hash id in `short-url/:id` endpoint
    - Controller fetches the models.ShortUrl from the given id
    - Fetched DataTable request and state used in `index.html` to restore the state if the page to shared hash id request
    - Mainly Go template helps to modify JavaScript variables/objects and HTML
*/
package short_url

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
	mymongo "github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetShortUrl get query results efficiently
func GetShortUrl(ctx context.Context, c *gin.Context, shortUrlCollectionName string, req models.ShortUrlRequest) string {
	collection := mymongo.GetCollection(mymongo.DBClient, shortUrlCollectionName)
	requestHash := getRequestHash(c, req)
	if checkIdHashExists(ctx, c, collection, requestHash) > 0 {
		// Request exists
		return requestHash
	} else {
		if err := mymongo.Insert(ctx, collection, models.ShortUrl{HashId: requestHash, Request: req.Request, SavedState: req.SavedState}); err != nil {
			utils.ErrorResponse(c, "ShortUrl insert failed", err, "")
		}
		return requestHash
	}
}

// GetRequestFromShortUrl get query results efficiently
func GetRequestFromShortUrl(ctx context.Context, c *gin.Context, shortUrlCollectionName string, hashId string) models.ShortUrl {
	var shortUrlList []models.ShortUrl
	collection := mymongo.GetCollection(mymongo.DBClient, shortUrlCollectionName)
	cursor, err := mymongo.GetFindOnlyMatchResults(ctx, collection, bson.M{"hash_id": hashId})
	if err != nil {
		utils.ErrorResponse(c, "getRequestFromShortUrl find query failed", err, "")
	}
	if err = cursor.All(ctx, &shortUrlList); err != nil {
		utils.ErrorResponse(c, "ShortUrl cursor failed", err, "")
	}
	return shortUrlList[0] // return only one, should be only one
}

// getRequestHash returns MD5 hash of datatable request
func getRequestHash(c *gin.Context, req models.ShortUrlRequest) string {
	out, err := json.Marshal(req)
	if err != nil {
		utils.ErrorResponse(c, "Json marshall error", err, "")
	}
	md5Instance := md5.New()
	md5Instance.Write(out)
	return hex.EncodeToString(md5Instance.Sum(nil))
}

// checkIdHashExists check if request hash is exists in the MongoDB collection
func checkIdHashExists(ctx context.Context, c *gin.Context, collection *mongo.Collection, hashId string) int64 {
	count, err := mymongo.GetCount(ctx, collection, bson.M{"hash_id": hashId})
	if err != nil {
		utils.ErrorResponse(c, "ShortUrl checkIdHashExists failed", err, "")
		return 0
	}
	return count
}
