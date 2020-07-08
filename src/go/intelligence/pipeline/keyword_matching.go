package pipeline

import (
	"go/intelligence/models"
	"go/intelligence/utils"
	"reflect"
	"strings"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CERN MONIT infrastructure Intelligence Module

//KeywordMatchFunction sd
type KeywordMatchFunction struct{}

//KeywordMatching function finds defined keywords in the shortDescription of alerts and assign severity level accordingly
func KeywordMatching(data <-chan models.AmJSON) <-chan models.AmJSON {

	dataWithSeverity := make(chan models.AmJSON)

	go func() {
		for each := range data {

			for _, service := range utils.ConfigJSON.Services {

				if each.Labels[utils.ConfigJSON.Alerts.ServiceLabel] == service.Name {
					// Using Reflection for dynamic keyword matching function calling
					// For details on Reflection see
					// http://golang.org/pkg/net/rpc/
					// http://stackoverflow.com/questions/12127585/go-lookup-function-by-name
					// https://play.golang.org/p/Yd-WDRzura
					t := reflect.ValueOf(KeywordMatchFunction{})
					m := t.MethodByName(service.KeywordMatchFunction)                         // associative function for keyword matching
					args := []reflect.Value{reflect.ValueOf(&each), reflect.ValueOf(service)} // list of function arguments
					m.Call(args)
				}

			}

			dataWithSeverity <- each
		}
		close(dataWithSeverity)
	}()

	return dataWithSeverity
}

//SsbKeywordMatching SSB alerts KeywordMatching function
func (k KeywordMatchFunction) SsbKeywordMatching(data *models.AmJSON, srv models.Service) {

	assignSeverityLevel := ""
	maxSeverityLevel := -1

	for key, value := range data.Annotations {
		if key == srv.KeywordLabel {
			for k, v := range srv.SeverityMap {
				if val, ok := value.(string); ok {
					if strings.Contains(strings.ToLower(val), k) {
						if utils.ConfigJSON.Alerts.SeverityLevels[v] > maxSeverityLevel {
							maxSeverityLevel = utils.ConfigJSON.Alerts.SeverityLevels[v]
							assignSeverityLevel = v
						}
					}
				}
			}
		}
	}

	for key := range data.Labels {
		if key == utils.ConfigJSON.Alerts.SeverityLabel {
			if assignSeverityLevel != "" {
				data.Labels[utils.ConfigJSON.Alerts.SeverityLabel] = assignSeverityLevel
			} else {
				data.Labels[utils.ConfigJSON.Alerts.SeverityLabel] = utils.ConfigJSON.Alerts.DefaultSeverityLevel
			}
		}
	}
}

//GgusKeywordMatching GGUS alerts KeywordMatching function
func (k KeywordMatchFunction) GgusKeywordMatching(data *models.AmJSON, srv models.Service) {

	assignSeverityLevel := ""
	for key, value := range data.Annotations {
		if key == srv.KeywordLabel {
			if val, ok := value.(string); ok {
				assignSeverityLevel = srv.SeverityMap[val]
			}
		}
	}

	for key := range data.Labels {
		if key == utils.ConfigJSON.Alerts.SeverityLabel {
			if assignSeverityLevel != "" {
				data.Labels[utils.ConfigJSON.Alerts.SeverityLabel] = assignSeverityLevel
			} else {
				data.Labels[utils.ConfigJSON.Alerts.SeverityLabel] = utils.ConfigJSON.Alerts.DefaultSeverityLevel
			}
		}
	}
}
