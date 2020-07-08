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

//PushAlert function for pushing modified alerts back to AlertManager
func PushAlert(data <-chan models.AmJSON) <-chan models.AmJSON {
	c := make(chan models.AmJSON)

	go func() {
		for each := range data {
			err := post(each)
			if err != nil {
				log.Printf("Could not push alert, error:%v\n", err)
				if utils.ConfigJSON.Server.Verbose > 1 {
					log.Printf("Alert Data: %s\n ", each)
				}
			}
			c <- each
		}
		close(c)
	}()
	return c
}

//post function for making post request on /api/v1/alerts alertmanager endpoint for creating alerts.
func post(data models.AmJSON) error {
	apiurl := utils.ValidateURL(utils.ConfigJSON.Server.CMSMONURL, utils.ConfigJSON.Server.PostAlertsAPI)
	var finalData []models.AmJSON
	finalData = append(finalData, data)

	jsonStr, err := json.Marshal(finalData)
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
		log.Println("Pushed Alerts: ", string(jsonStr))
	}

	return nil
}
