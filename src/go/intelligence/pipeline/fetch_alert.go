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
// Description: CERN MONIT infrastructure Intelligence Module

//FetchAlert function for fetching all active alerts from AlertManager
func FetchAlert() <-chan models.AmJSON {
	fetchedData := make(chan models.AmJSON)
	go func() {
		data, err := get()
		if err != nil {
			log.Printf("Could not fetch alerts from AlertManager, error:%v\n", err)
		}
		for _, each := range data.Data {
			fetchedData <- each
		}
		close(fetchedData)
	}()
	return fetchedData
}

//get function for get request on /api/v1/alerts alertmanager endpoint for fetching alerts.
func get() (models.AmData, error) {

	var data models.AmData
	apiurl := utils.ValidateURL(utils.ConfigJSON.Server.CMSMONURL, utils.ConfigJSON.Server.GetAlertsAPI) //GET API for fetching all AM alerts.

	req, err := http.NewRequest("GET", apiurl, nil)
	if err != nil {
		log.Printf("Request Error, error: %v\n", err)
		return data, err
	}
	req.Header.Add("Accept-Encoding", "identity")
	req.Header.Add("Accept", "application/json")

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
		return data, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Http Response Code Error, status code: %d", resp.StatusCode)
		return data, errors.New("Respose Error")
	}

	defer resp.Body.Close()

	byteValue, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Unable to read JSON Data from AlertManager GET API, error: %v\n", err)
		return data, err
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
		log.Printf("Unable to parse JSON Data from AlertManager GET API, error: %v\n", err)
		return data, err
	}

	return data, nil
}
