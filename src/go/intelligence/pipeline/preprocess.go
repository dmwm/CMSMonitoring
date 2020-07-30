package pipeline

import (
	"go/intelligence/models"
	"go/intelligence/utils"
	"log"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CMS MONIT infrastructure Intelligence Module

//Preprocess - function make required changes to alerts and filter only SSB and GGUS alerts
func Preprocess(data <-chan models.AmJSON) <-chan models.AmJSON {
	utils.IfSilencedMap = make(map[string]utils.SilenceMapVals)

	err := updateSilencedMap()
	if err != nil {
		log.Printf("Unable to update the IfSilenced Map, error: %v\n", err)
	}

	if utils.ConfigJSON.Server.Verbose > 1 {
		log.Printf("Current IfSilenced Map: %v", utils.IfSilencedMap)
	}

	preprocessedData := make(chan models.AmJSON)
	go func() {
		defer close(preprocessedData)
		for each := range data {
			for _, service := range utils.ConfigJSON.Services {
				if each.Labels[utils.ConfigJSON.Alerts.ServiceLabel] == service.Name {
					if val, ok := each.Labels[utils.ConfigJSON.Alerts.UniqueLabel].(string); ok {
						if _, alertFoundInSilencedMap := utils.IfSilencedMap[val]; !alertFoundInSilencedMap {
							preprocessedData <- each
						}
					}
				}
			}
		}
	}()

	return preprocessedData
}

//updateSilencedMap -function for updating the ifSilenced Map to help us not to push redundant silences
func updateSilencedMap() error {

	var err error
	var data models.AllSilences

	data, utils.NoOfActiveSilencesBeforeIntModule, utils.NoOfExpiredSilencesBeforeIntModule, utils.NoOfPendingSilencesBeforeIntModule, err = utils.GetSilences()
	if err != nil {
		log.Printf("Unable to Update Silence Map, error: %v", err)
	}

	for _, each := range data.Data {

		for _, matcher := range each.Matchers {
			if matcher.Name == utils.ConfigJSON.Alerts.UniqueLabel {
				for _, sStatus := range utils.ConfigJSON.Silence.SilenceStatus {
					if each.Status.State == sStatus {
						utils.IfSilencedMap[matcher.Value] = utils.SilenceMapVals{IfAvail: 1, SilenceID: each.ID}
					}
				}
			}
		}
	}

	return nil
}
