package pipeline

import (
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

//DeleteSilence - function for deleting expired silences
func DeleteSilence(data <-chan models.AmJSON) <-chan models.AmJSON {

	finalData := make(chan models.AmJSON)
	go func() {
		defer close(finalData)
		if *utils.DryRun == false {
			deleteSilenceHelper()
		}
		for each := range data {
			finalData <- each
		}
	}()
	return finalData
}

//deleteSilenceHelper - helper function for deleting a silence
func deleteSilenceHelper() {
	utils.DataReadWriteLock.RLock()
	defer utils.DataReadWriteLock.RUnlock()
	for silencedAlert, val := range utils.IfSilencedMap {
		if utils.ExtAlertsMap[silencedAlert] == 1 {
			continue
		}
		deleteSuppressedAlert(silencedAlert)
		err := deleteSilenceAPICall(silencedAlert, val.SilenceID)
		if err != nil {
			log.Printf("Could not delete expired silence for: %s, error:%v\n", silencedAlert, err)
		}

		utils.ChangeCounters.NoOfExpiredSilences++
	}
}

//deleteSuppressedAlert - helper function for deleting a suppressed alert
func deleteSuppressedAlert(silencedAlert string) {
	utils.SuppressedAlertsDataReadWriteLock.RLock()
	defer utils.SuppressedAlertsDataReadWriteLock.RUnlock()
	if data, ifDataFound := utils.ExtSuppressedAlertsMap[silencedAlert]; ifDataFound {
		data.EndsAt = time.Now()
		err := utils.PostAlert(data)
		if err != nil {
			log.Printf("Could not delete suppressed alert, error:%v\n", err)
			if utils.ConfigJSON.Server.Verbose > 1 {
				log.Printf("Suppressed Alert Data: %s\n ", data)
			}
		}
	}
}

//deleteSilenceAPICall - helper function for making API call for deleting a silence
func deleteSilenceAPICall(silencedAlert, silenceID string) error {

	apiurl := utils.ValidateURL(utils.ConfigJSON.Server.CMSMONURL, utils.ConfigJSON.Server.DeleteSilenceAPI) // DELETE API for deleting silences.
	apiurl = utils.ValidateURL(apiurl, "/"+silenceID)

	req, err := http.NewRequest("DELETE", apiurl, nil)
	if err != nil {
		log.Printf("Request Error, error: %v\n", err)
		return err
	}

	timeout := time.Duration(utils.ConfigJSON.Server.HTTPTimeout) * time.Second
	client := &http.Client{Timeout: timeout}

	if utils.ConfigJSON.Server.Verbose > 1 {
		log.Println("DELETE", apiurl)
	} else if utils.ConfigJSON.Server.Verbose > 2 {
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

	if utils.ConfigJSON.Server.Verbose > 2 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}

	if utils.ConfigJSON.Server.Verbose > 2 {
		log.Printf("Silence Deleted for:\n%+v\n", silencedAlert)
	}

	return nil
}
