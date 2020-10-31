package pipeline

import (
	"go/intelligence/models"
	"go/intelligence/utils"
	"log"
	"time"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CMS MONIT infrastructure Intelligence Module

// DeleteSilence - function for deleting expired silences
func DeleteSilence(data <-chan models.AmJSON) <-chan models.AmJSON {

	if utils.ConfigJSON.Server.Verbose > 0 {
		log.Println("DeleteSilence step")
	}

	finalData := make(chan models.AmJSON)
	go func() {
		defer close(finalData)
		if utils.ConfigJSON.Server.DryRun == false {
			deleteSilenceHelper()
		}
		for each := range data {
			if utils.ConfigJSON.Server.Verbose > 1 {
				log.Println(each.String())
			}
			finalData <- each
		}
	}()
	return finalData
}

// deleteSilenceHelper - helper function for deleting a silence
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

		utils.ChangeCounters.NoOfSilencesDeleted++
	}
}

// deleteSuppressedAlert - helper function for deleting a suppressed alert
func deleteSuppressedAlert(silencedAlert string) {
	utils.DataReadWriteLock.RLock()
	defer utils.DataReadWriteLock.RUnlock()
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

// deleteSilenceAPICall - helper function for making API call for deleting a silence
func deleteSilenceAPICall(silencedAlert, silenceID string) error {

	apiURL := utils.ValidateURL(utils.ConfigJSON.Server.CMSMONURL, utils.ConfigJSON.Server.DeleteSilenceAPI)
	apiURL = utils.ValidateURL(apiURL, "/"+silenceID)

	var headers [][]string
	headers = append(headers, []string{"Content-Type", "application/json"})
	resp := utils.HttpCall("DELETE", apiURL, headers, nil)
	defer resp.Body.Close()

	if utils.ConfigJSON.Server.Verbose > 2 {
		log.Printf("Silence Deleted for:\n%+v\n", silencedAlert)
	}

	return nil
}
