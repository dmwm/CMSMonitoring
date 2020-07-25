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

//PushAlert - function for pushing modified alerts back to AlertManager
func PushAlert(data <-chan models.AmJSON) <-chan models.AmJSON {
	c := make(chan models.AmJSON)

	go func() {
		for each := range data {
			err := utils.PostAlert(each)
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
