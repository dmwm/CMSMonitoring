package controllers

import (
	"context"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/configs"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/datasets"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/detailed_datasets"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/models"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/responses"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/utils"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
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

// GetDatasets controller that returns datasets according to DataTable request json
func GetDatasets() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(configs.EnvConnTimeout())*time.Second)
		defer cancel()

		// Get request json with validation
		dtRequest := models.DataTableRequest{}
		if err := c.ShouldBindJSON(&dtRequest); err != nil {
			utils.ErrorResponse(c, "Bad request", err)
			return
		}
		log.Printf("Requst: %#v", dtRequest)
		totalRecCount := datasets.GetTotalRecCount(ctx, c)
		filteredRecCount := datasets.GetFilteredRecCount(ctx, c, datasets.GetSearchBson(dtRequest))
		datasetResults := datasets.GetResults(ctx, c, dtRequest)
		// Send response in DataTable required format
		//  - Need to return exactly same "Draw" value that DataTable sent in incoming request
		c.JSON(http.StatusOK,
			models.DatatableDatasetsResponse{
				Draw:            dtRequest.Draw,
				RecordsTotal:    totalRecCount,
				RecordsFiltered: filteredRecCount,
				Data:            datasetResults,
			},
		)
	}
}

// GetDetailedDs controller that returns datasets according to DataTable request json
func GetDetailedDs() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(configs.EnvConnTimeout())*time.Second)
		defer cancel()

		// Get request json with validation
		dtRequest := models.DataTableCustomRequest{}
		if err := c.ShouldBindJSON(&dtRequest); err != nil {
			utils.ErrorResponse(c, "Bad request", err)
			return
		}
		log.Printf("Requst: %#v", dtRequest)
		detailedDatasetsResp := detailed_datasets.GetResults(ctx, c, dtRequest)

		// Send response in DataTable required format
		//  - Need to return exactly same "Draw" value that DataTable sent in incoming request
		c.JSON(http.StatusOK,
			detailedDatasetsResp,
		)
	}
}

// GetIndexPage serves index.html page
func GetIndexPage(c *gin.Context) {
	// Call the HTML method of the Context to render a template
	c.HTML(
		// Set the HTTP status to 200 (OK)
		http.StatusOK,
		// Use the index.html template
		"index.html",
		// Pass the data that the page uses (in this case, 'title')
		gin.H{
			"title": "Home Page",
		},
	)
}

// GetDetailsPage serves details.html page
func GetDetailsPage(c *gin.Context) {
	// Call the HTML method of the Context to render a template
	c.HTML(
		// Set the HTTP status to 200 (OK)
		http.StatusOK,
		// Use the index.html template
		"details.html",
		// Pass the data that the page uses (in this case, 'title')
		gin.H{
			"title": "Detailed Datasets Page",
		},
	)
}
