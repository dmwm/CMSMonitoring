package controllers

import (
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/responses"
	"github.com/gin-gonic/gin"
	"net/http"
)

var (
	// ServerInfo defines server info comes from Makefile
	ServerInfo string
	// GitVersion git version comes from Makefile
	GitVersion string
)

// GetServerInfo provides basic functionality of status response
func GetServerInfo(c *gin.Context) {
	c.JSON(http.StatusOK,
		responses.ServerInfo{
			ServiceVersion: GitVersion,
			Server:         ServerInfo,
		})
}
