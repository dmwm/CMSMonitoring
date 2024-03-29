package controllers

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

// GetServiceInfo provides basic functionality of status response
func GetServiceInfo(gitVersion string, serviceInfo string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK,
			models.ServerInfoResp{
				ServiceVersion: gitVersion,
				Server:         serviceInfo,
			})
	}
}
