package pipeline

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/intelligence/models"
	"go/intelligence/utils"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CMS MONIT infrastructure Intelligence Module

//Silence - function silences the old alert
func Silence(data <-chan models.AmJSON) <-chan models.AmJSON {

	silencedData := make(chan models.AmJSON)
	go func() {
		for each := range data {
			err := silenceAlert(each)
			if err != nil {
				log.Printf("Could not silence alert, error:%v\n", err)
				if utils.ConfigJSON.Server.Verbose > 1 {
					log.Printf("Silence Data: %s\n ", each)
				}
			}
			silencedData <- each
		}
		close(silencedData)
	}()
	return silencedData
}

//silenceAlert - helper function for silencing old alerts
func silenceAlert(data models.AmJSON) error {

	apiurl := utils.ValidateURL(utils.ConfigJSON.Server.CMSMONURL, utils.ConfigJSON.Server.PostSilenceAPI) // POST API for creating silences.

	var sData models.SilenceData
	var alertnameMatcher models.Matchers
	var severityMatcher models.Matchers

	sData.StartsAt = data.StartsAt //Start Time equal to main alert	   //So that the silence remains on old alert till the lifetime of the alert
	sData.EndsAt = data.EndsAt     //End Time equal to main alert	  //So that the silence remains on old alert till the lifetime of the alert
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
				if data.Labels[utils.ConfigJSON.Alerts.ServiceLabel] == service.Name {
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

	req, err := http.NewRequest("POST", apiurl, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Printf("Request Error, error: %v\n", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	timeout := time.Duration(utils.ConfigJSON.Server.HTTPTimeout) * time.Second
	client := &http.Client{Timeout: timeout}

	if utils.ConfigJSON.Server.Verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("Request: ", string(dump))
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Response Error, error: %v\n", err)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Http Response Code Error, status code: %d", resp.StatusCode)
		return errors.New("Http Response Code Error")
	}

	defer resp.Body.Close()

	if utils.ConfigJSON.Server.Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}

	if utils.ConfigJSON.Server.Verbose > 1 {
		log.Printf("Silence Data:\n%+v\n", string(jsonStr))
	}

	return nil
}
