package pipeline

import (
	"go/intelligence/models"
	"go/intelligence/utils"
	"strings"
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
			for key, value := range each.Annotations {
				if key == utils.ConfigJSON.SsbKeywordLabel {
					if val, ok := value.(string); ok {
						each.Annotations[key] = strings.ToLower(val)
					}
				}
			}

			if each.Labels["service"] == "SSB" || each.Labels["service"] == "GGUS" {
				preprocessedData <- each
			}
		}
		close(preprocessedData)
	}()

	return preprocessedData
}
