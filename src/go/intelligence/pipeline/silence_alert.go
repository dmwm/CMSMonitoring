package pipeline

import (
	"bytes"
	"encoding/json"
	"github.com/dmwm/CMSMonitoring/src/go/intelligence/models"
	"github.com/dmwm/CMSMonitoring/src/go/intelligence/utils"
	"log"
	"time"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CMS MONIT infrastructure Intelligence Module

// Silence - function silences the old alert
func Silence(data <-chan models.AmJSON) <-chan models.AmJSON {

	if utils.ConfigJSON.Server.Verbose > 0 {
		log.Println("Silence step")
	}
	silencedData := make(chan models.AmJSON)
	go func() {
		defer close(silencedData)
		for each := range data {
			if utils.ConfigJSON.Server.Verbose > 1 {
				log.Println(each.String())
			}
			if utils.ConfigJSON.Server.DryRun == false {
				err := silenceAlert(each)
				if err != nil {
					log.Printf("Could not silence alert, error:%v\n", err)
					if utils.ConfigJSON.Server.Verbose > 1 {
						log.Printf("Silence Data: %s\n ", each)
					}
				}
			}
			utils.ChangeCounters.NoOfSilencesCreated++
			silencedData <- each
		}
	}()
	return silencedData
}

// silenceAlert - helper function for silencing old alerts
func silenceAlert(data models.AmJSON) error {

	apiURL := utils.ValidateURL(utils.ConfigJSON.Server.CMSMONURL, utils.ConfigJSON.Server.PostSilenceAPI)

	var sData models.SilenceData
	var alertnameMatcher models.Matchers
	var severityMatcher models.Matchers

	if data.StartsAt.After(time.Now()) {
		// wtart Time equal to current time if it's starting in future
		// so that the silence remains on old alert till the lifetime of the alert
		sData.StartsAt = time.Now()
	} else {
		// start Time equal to main alert
		// so that the silence remains on old alert till the lifetime of the alert
		sData.StartsAt = data.StartsAt
	}

	// end Time equal to main alert
	// so that the silence remains on old alert till the lifetime of the alert
	sData.EndsAt = data.EndsAt
	sData.CreatedBy = utils.ConfigJSON.Silence.CreatedBy
	sData.Comment = utils.ConfigJSON.Silence.Comment

	alertnameMatcher.Name = utils.ConfigJSON.Alerts.UniqueLabel
	severityMatcher.Name = utils.ConfigJSON.Alerts.SeverityLabel

	for k, v := range data.Labels {
		if k == utils.ConfigJSON.Alerts.UniqueLabel {
			if val, ok := v.(string); ok {
				alertnameMatcher.Value = val
			}
		}

		if k == utils.ConfigJSON.Alerts.SeverityLabel {
			for _, service := range utils.ConfigJSON.Services {
				slabel, ok := utils.Get(data.Labels, utils.ConfigJSON.Alerts.ServiceLabel)
				if ok && slabel == service.Name {
					severityMatcher.Value = service.DefaultLevel
				}
			}
		}
	}

	sData.Matchers = append(sData.Matchers, alertnameMatcher)
	sData.Matchers = append(sData.Matchers, severityMatcher)
	jsonStr, err := json.Marshal(sData)
	if err != nil {
		log.Printf("Unable to convert JSON Data, error: %v\n", err)
		return err
	}

	var headers [][]string
	headers = append(headers, []string{"Content-Type", "application/json"})
	resp := utils.HttpCall("POST", apiURL, headers, bytes.NewBuffer(jsonStr))
	defer resp.Body.Close()

	if utils.ConfigJSON.Server.Verbose > 1 {
		log.Printf("Silence Data:\n%+v\n", string(jsonStr))
	}

	return nil
}
