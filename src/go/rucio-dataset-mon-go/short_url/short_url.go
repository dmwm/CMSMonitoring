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

var (
	collectionName = "short_url"
	collection     *mongo.Collection
)

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
func checkIdHashExists(ctx context.Context, c *gin.Context, hashId string) int64 {
	count, err := mymongo.GetCount(ctx, collection, bson.M{"hash_id": hashId})
	if err != nil {
		utils.ErrorResponse(c, "ShortUrl checkIdHashExists failed", err, "")
		return 0
	}
	return count
}

// GetShortUrl get query results efficiently
func GetShortUrl(ctx context.Context, c *gin.Context, req models.ShortUrlRequest) string {
	collection = mymongo.GetCollection(mymongo.DBClient, collectionName)
	requestHash := getRequestHash(c, req)
	if checkIdHashExists(ctx, c, requestHash) > 0 {
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
func GetRequestFromShortUrl(ctx context.Context, c *gin.Context, hashId string) models.ShortUrl {
	var shortUrlList []models.ShortUrl
	collection = mymongo.GetCollection(mymongo.DBClient, collectionName)
	cursor, err := mymongo.GetFindOnlyMatchResults(ctx, collection, bson.M{"hash_id": hashId})
	if err != nil {
		utils.ErrorResponse(c, "GetRequestFromShortUrl find query failed", err, "")
	}
	if err = cursor.All(ctx, &shortUrlList); err != nil {
		utils.ErrorResponse(c, "ShortUrl cursor failed", err, "")
	}
	return shortUrlList[0] // return only one, should be only one
}
