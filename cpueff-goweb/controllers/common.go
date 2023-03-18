package controllers

// Copyright (c) 2023 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"context"
	"github.com/dmwm/CMSMonitoring/cpueff-goweb/models"
	mymongo "github.com/dmwm/CMSMonitoring/cpueff-goweb/mongo"
	"github.com/dmwm/CMSMonitoring/cpueff-goweb/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"net/http"
	"sort"
)

// ServiceCtrl provides basic functionality of status response
func ServiceCtrl(gitVersion string, serviceInfo string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK,
			models.ServerInfoResp{
				ServiceVersion: gitVersion,
				Server:         serviceInfo,
			})
	}
}

// GetDataSourceTimestamp last update of the page
func GetDataSourceTimestamp(ctx context.Context, c *gin.Context, sourceTimeCollectionName string) models.DataSourceTS {
	var dataTimestamp models.DataSourceTS
	collection := mymongo.GetCollection(sourceTimeCollectionName)
	failTimeStamp := models.DataSourceTS{StartDate: "0000-00-00", EndDate: "0000-00-00"}
	if err := mymongo.GetFindOneResult(ctx, collection, bson.M{}).Decode(&dataTimestamp); err != nil {
		utils.ErrorResponse(c, "Find query failed for getDataSourceTimestamp: ", err, "")
		return failTimeStamp
	}
	return dataTimestamp
}

// GetTierEfficiencies tiers
func GetTierEfficiencies(ctx context.Context, c *gin.Context, mongoCol string) []models.CondorTierEfficiencies {
	var tierCpuEffs []models.CondorTierEfficiencies
	collection := mymongo.GetCollection(mongoCol)
	cursor, err := mymongo.GetFindQueryResults(ctx, collection, bson.M{}, bson.D{}, 0, 0)
	if err != nil {
		utils.ErrorResponse(c, "Find query failed for GetTierEfficiencies", err, "")
	}
	if err = cursor.All(ctx, &tierCpuEffs); err != nil {
		utils.ErrorResponse(c, "wfCpuEffs cursor failed", err, "")
	}
	// Sort by Type and then Tier. "Production" on the top.
	sort.Slice(tierCpuEffs, func(i, j int) bool {
		iv, jv := tierCpuEffs[i], tierCpuEffs[j]
		switch {
		case iv.Type != jv.Type:
			if iv.Type == "production" {
				return true
			} else {
				return iv.Type < jv.Type

			}
		case iv.Tier != jv.Tier:
			return iv.Tier < jv.Tier
		default:
			return iv.TierCpuEff < jv.TierCpuEff
		}
	})
	log.Println(tierCpuEffs)
	return tierCpuEffs
}
