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

// PushAlert - function for pushing modified alerts back to AlertManager
func PushAlert(data <-chan models.AmJSON) <-chan models.AmJSON {
	if utils.ConfigJSON.Server.Verbose > 0 {
		log.Println("PushAlert step")
	}
	c := make(chan models.AmJSON)

	go func() {
		defer close(c)
		for each := range data {
			if utils.ConfigJSON.Server.Verbose > 1 {
				log.Println(each.String())
			}
			if utils.ConfigJSON.Server.DryRun == false {
				err := utils.PostAlert(each)
				if err != nil {
					log.Printf("Could not push alert, error:%v\n", err)
					if utils.ConfigJSON.Server.Verbose > 1 {
						log.Printf("Alert Data: %s\n ", each)
					}
				}
			}
			utils.ChangeCounters.NoOfPushedAlerts++
			c <- each
		}
	}()
	return c
}
