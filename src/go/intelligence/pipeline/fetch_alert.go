package pipeline

import (
	"github.com/dmwm/CMSMonitoring/src/go/intelligence/models"
	"github.com/dmwm/CMSMonitoring/src/go/intelligence/utils"
	"log"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CMS MONIT infrastructure Intelligence Module
// Code is based on
// https://towardsdatascience.com/concurrent-data-pipelines-in-golang-85b18c2eecc2

// FetchAlert - function for fetching all active alerts from AlertManager
func FetchAlert() <-chan models.AmJSON {
	if utils.ConfigJSON.Server.Verbose > 0 {
		log.Println("FetchAlert step")
	}

	fetchedData := make(chan models.AmJSON)

	_, err := utils.GetAlerts(utils.ConfigJSON.Server.GetSuppressedAlertsAPI, false)
	if err != nil {
		log.Printf("Could not fetch suppressed alerts from AlertManager, error:%v\n", err)
	}

	data, err := utils.GetAlerts(utils.ConfigJSON.Server.GetAlertsAPI, true)
	if err != nil {
		log.Printf("Could not fetch alerts from AlertManager, error:%v\n", err)
	}
	utils.ChangeCounters.NoOfAlerts = len(utils.ExtAlertsMap)

	go func() {
		defer close(fetchedData)
		for _, each := range data.Data {
			if utils.ConfigJSON.Server.Verbose > 1 {
				log.Println(each.String())
			}
			fetchedData <- each
		}
	}()
	return fetchedData
}
