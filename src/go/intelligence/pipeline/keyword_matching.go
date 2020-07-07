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

//KeywordMatching function finds defined keywords in the shortDescription of alerts and assign severity level accordingly
func KeywordMatching(data <-chan models.AmJSON) <-chan models.AmJSON {

	dataWithSeverity := make(chan models.AmJSON)
	go func() {
		for each := range data {
			if each.Labels["service"] == "SSB" {
				ssbKeywordMatching(&each)
			}
			if each.Labels["service"] == "GGUS" {
				ggusKeywordMatching(&each)
			}

			dataWithSeverity <- each
		}
		close(dataWithSeverity)
	}()

	return dataWithSeverity
}

//SSB alerts KeywordMatching function
func ssbKeywordMatching(data *models.AmJSON) {
	assignSeverityLevel := ""
	maxSeverityLevel := -1

	for key, value := range data.Annotations {
		if key == utils.ConfigJSON.SsbKeywordLabel {
			for k, v := range utils.ConfigJSON.SsbSeverityMap {
				if val, ok := value.(string); ok {
					if strings.Contains(val, k) {
						if utils.ConfigJSON.SeverityLevels[v] > maxSeverityLevel {
							maxSeverityLevel = utils.ConfigJSON.SeverityLevels[v]
							assignSeverityLevel = v
						}
					}
				}
			}
			break
		}
	}

	for key := range data.Labels {
		if key == utils.ConfigJSON.SeverityLabel {
			if assignSeverityLevel != "" {
				data.Labels[utils.ConfigJSON.SeverityLabel] = assignSeverityLevel
			} else {
				data.Labels[utils.ConfigJSON.SeverityLabel] = utils.ConfigJSON.DefaultSeverityLevel
			}
			break
		}
	}
}

//GGUS alerts KeywordMatching function
func ggusKeywordMatching(data *models.AmJSON) {
	assignSeverityLevel := ""
	for key, value := range data.Annotations {
		if key == utils.ConfigJSON.GGUSKeywordLabel {
			if val, ok := value.(string); ok {
				assignSeverityLevel = utils.ConfigJSON.GGUSSeverityMap[val]
			}
			break
		}
	}

	for key := range data.Labels {
		if key == utils.ConfigJSON.SeverityLabel {
			if assignSeverityLevel != "" {
				data.Labels[utils.ConfigJSON.SeverityLabel] = assignSeverityLevel
			} else {
				data.Labels[utils.ConfigJSON.SeverityLabel] = utils.ConfigJSON.DefaultSeverityLevel
			}
			break
		}
	}
}
