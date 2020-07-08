package pipeline

import (
	"go/intelligence/models"
	"go/intelligence/utils"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CERN MONIT infrastructure Intelligence Module

//Preprocess function make required changes to alerts and filter only SSB and GGUS alerts
func Preprocess(data <-chan models.AmJSON) <-chan models.AmJSON {

	preprocessedData := make(chan models.AmJSON)
	go func() {
		for each := range data {
			for _, service := range utils.ConfigJSON.Services {
				if each.Labels[utils.ConfigJSON.Alerts.ServiceLabel] == service.Name {
					preprocessedData <- each
				}
			}
		}
		close(preprocessedData)
	}()

	return preprocessedData
}
