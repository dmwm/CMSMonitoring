package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/models"
	mymongo "github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/mongo"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
)

// GetShortUrlParam controller that returns short url parameter which is md5 hash of the datatables request
func GetShortUrlParam(collectionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// We need to provide models.ShortUrlRequest to the controller initializer and use same type in casting
		ctx, cancel, req := InitializeCtxAndBindRequestBody(c, models.ShortUrlRequest{})
		defer cancel()
		requestHash := GetShortUrl(ctx, c, collectionName, req.(models.ShortUrlRequest))
		c.JSON(http.StatusOK,
			requestHash,
		)
		return
	}
}

// GetShortUrl get query results efficiently
func GetShortUrl(ctx context.Context, c *gin.Context, shortUrlCollectionName string, req models.ShortUrlRequest) string {
	collection := mymongo.GetCollection(shortUrlCollectionName)
	requestHash := getRequestHash(c, req)
	if checkIdHashExists(ctx, c, collection, requestHash) > 0 {
		// Request exists
		return requestHash
	} else {
		if err := mymongo.Insert(
			ctx,
			collection,
			models.ShortUrl{HashId: requestHash, Page: req.Page, Request: req.Request, SavedState: req.SavedState},
		); err != nil {
			utils.ErrorResponse(c, "ShortUrl insert failed", err, "")
		}
		return requestHash
	}
}

// GetRequestFromShortUrl get query results efficiently
func GetRequestFromShortUrl(ctx context.Context, c *gin.Context, shortUrlCollectionName string, hashId string) models.ShortUrl {
	var shortUrlObj models.ShortUrl
	collection := mymongo.GetCollection(shortUrlCollectionName)
	if err := mymongo.GetFindOneResult(ctx, collection, bson.M{"hashId": hashId}).Decode(&shortUrlObj); err != nil {
		utils.ErrorResponse(c, "ShortUrl result not found for hashId: "+hashId, err, "")
	}
	return shortUrlObj
}

// getRequestHash returns MD5 hash of datatable request
func getRequestHash(c *gin.Context, req models.ShortUrlRequest) string {
	// MD5 Hash
	out, err := json.Marshal(req)
	if err != nil {
		utils.ErrorResponse(c, "Json marshall error", err, req.String())
	}
	md5Instance := md5.New()
	md5Instance.Write(out)
	md5Hash := hex.EncodeToString(md5Instance.Sum(nil))[0:8]

	// Short URL structure: "md5 8 character + Page(main/detailed/etc.) + SearchBuilder request as query params"
	return md5Hash + "+" + req.Page + "+" + req.Request.SearchBuilderRequest.GetPrettyURL()
}

// checkIdHashExists check if request hash is exists in the MongoDB collection
func checkIdHashExists(ctx context.Context, c *gin.Context, collection *mongo.Collection, hashId string) int64 {
	count, err := mymongo.GetCount(ctx, collection, bson.M{"hashId": hashId})
	if err != nil {
		utils.ErrorResponse(c, "ShortUrl checkIdHashExists failed", err, "")
		return 0
	}
	return count
}
