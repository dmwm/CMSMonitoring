package short_url

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
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
func getRequestHash(c *gin.Context, req models.DataTableCustomRequest) string {
	var encBuff bytes.Buffer
	enc := gob.NewEncoder(&encBuff)
	if err := enc.Encode(req); err != nil {
		utils.ErrorResponse(c, "Encode error", err, "")
	}
	md5Instance := md5.New()
	md5Instance.Write(encBuff.Bytes())
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
func GetShortUrl(ctx context.Context, c *gin.Context, req models.DataTableCustomRequest) string {
	collection = mymongo.GetCollection(mymongo.DBClient, collectionName)
	req.Draw = 1 // Set Draw value as 1 which is default
	requestHash := getRequestHash(c, req)
	if checkIdHashExists(ctx, c, requestHash) > 0 {
		// Request exists
		return requestHash
	} else {
		if err := mymongo.Insert(ctx, collection, models.ShortUrl{HashId: requestHash, Request: req}); err != nil {
			utils.ErrorResponse(c, "ShortUrl insert failed", err, "")
		}
		return requestHash
	}
}

// GetRequestFromShortUrl get query results efficiently
func GetRequestFromShortUrl(ctx context.Context, c *gin.Context, hashId string) models.DataTableCustomRequest {
	var shortUrlList []models.ShortUrl
	collection = mymongo.GetCollection(mymongo.DBClient, collectionName)
	cursor, err := mymongo.GetFindOnlyMatchResults(ctx, collection, bson.M{"hash_id": hashId})
	if err != nil {
		utils.ErrorResponse(c, "GetRequestFromShortUrl find query failed", err, "")
	}
	if err = cursor.All(ctx, &shortUrlList); err != nil {
		utils.ErrorResponse(c, "ShortUrl cursor failed", err, "")
	}
	return shortUrlList[0].Request // return only one, should be only one
}
