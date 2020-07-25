package pipeline

import (
	"encoding/json"
	"errors"
	"go/intelligence/models"
	"go/intelligence/utils"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
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
						if utils.IfSilencedMap[val].IfAvail != 1 {
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

	var data models.AllSilences
	apiurl := utils.ValidateURL(utils.ConfigJSON.Server.CMSMONURL, utils.ConfigJSON.Server.GetSilencesAPI) //GET API for fetching all AM Silences.

	req, err := http.NewRequest("GET", apiurl, nil)
	if err != nil {
		log.Printf("Request Error, error: %v\n", err)
		return err
	}
	req.Header.Add("Accept-Encoding", "identity")
	req.Header.Add("Accept", "application/json")

	timeout := time.Duration(utils.ConfigJSON.Server.HTTPTimeout) * time.Second
	client := &http.Client{Timeout: timeout}

	if utils.ConfigJSON.Server.Verbose > 1 {
		log.Println("GET", apiurl)
	} else if utils.ConfigJSON.Server.Verbose > 1 {
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
		return errors.New("Respose Error")
	}

	defer resp.Body.Close()

	byteValue, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Unable to read JSON Data from AlertManager Silence GET API, error: %v\n", err)
		return err
	}

	if utils.ConfigJSON.Server.Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}

	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		if utils.ConfigJSON.Server.Verbose > 0 {
			log.Println(string(byteValue))
		}
		log.Printf("Unable to parse JSON Data from AlertManager Silence GET API, error: %v\n", err)
		return err
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
