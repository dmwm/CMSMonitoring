package pipeline

import (
	"go/intelligence/models"
	"go/intelligence/utils"
	"log"
	"strings"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CMS MONIT infrastructure Intelligence Module
// Code is based on
// https://towardsdatascience.com/concurrent-data-pipelines-in-golang-85b18c2eecc2

//KeywordMatching - function finds defined keywords in the shortDescription of alerts and assign severity level accordingly
func KeywordMatching(data <-chan models.AmJSON) <-chan models.AmJSON {

	if utils.ConfigJSON.Server.Verbose > 0 {
		log.Println("KeyworkMatching pipeline")
	}
	dataWithSeverity := make(chan models.AmJSON)

	go func() {
		defer close(dataWithSeverity)
		for each := range data {
			if utils.ConfigJSON.Server.Verbose > 1 {
				log.Println(each.String())
			}
			changedData := each
			for _, service := range utils.ConfigJSON.Services {
				lock.RLock()
				slabel, ok := each.Labels[utils.ConfigJSON.Alerts.ServiceLabel]
				lock.RUnlock()
				if ok && slabel == service.Name {
					keywordMatchingHelper(&changedData, service)
				}
				//                 if changedData.Labels[utils.ConfigJSON.Alerts.ServiceLabel] == service.Name {
				//                     keywordMatchingHelper(&changedData, service)
				//                 }
			}
			dataWithSeverity <- changedData
		}
	}()
	return dataWithSeverity
}

//keywordMatchingHelper - helper function which matches keywords and assigns severity levels
func keywordMatchingHelper(data *models.AmJSON, srv models.Service) {
	/*
				Common Structure of an alert

				{	".",
					".",
					"labels": {
						"alertname",
						"service",
						"tag",
						"severity"
						".",
						"."
		    		},
					"annotations": {
						"shortDescription" or "Priority",
						".",
						".",
						"."
		    		}
				}  here "." represents other fields can be/are introduced as well.

				Each alert will have a field in it's "Annotations" which can be helpful to determine it's severity.
				Ex. SSB has "shortDescription", GGUS has "Priority".

				We have defined some set of rules or in other words some set of keywords which defines the severity of an alert and upon finding such keyword
				in decided Annotation's field of the alerts, we assign corresponding severity level.

				Ex. When SSB's shortDescription says "Short network interruption in building 182-275" , we can extract relevant keyword like "interruption"
				and we will assign mapped severity level to this keyword i.e. "interruption": "medium".

	*/

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
