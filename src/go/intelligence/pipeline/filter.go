package pipeline

import (
	"go/intelligence/models"
	"go/intelligence/utils"
	"log"
)

// Module     : intelligence
// Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
// Created    : Fri Mar 12 09:54:45 EST 2021
// Description: CMS MONIT infrastructure Intelligence Module
// Code is based on
// https://towardsdatascience.com/concurrent-data-pipelines-in-golang-85b18c2eecc2

// Filter provide filtering of incoming data
func Filter(data <-chan models.AmJSON) <-chan models.AmJSON {

	if utils.ConfigJSON.Server.Verbose > 0 {
		log.Println("Filter pipeline")
	}
	out := make(chan models.AmJSON)

	go func() {
		defer close(out)
		for each := range data {
			if utils.ConfigJSON.Server.Verbose > 1 {
				log.Println(each.String())
			}
			diff := each.EndsAt.Sub(each.StartsAt)
			if diff.Hours() < utils.ConfigJSON.Alerts.DurationThreshold {
				// if our alert time range (defined between starts and ends timestamps)
				// is less than our threshold we'll keep it, otherwise the alert will be
				// rejected (i.e. we'll not pass it to next pipeline level)
				out <- each
			} else {
				log.Println("filter out", each.String())
			}
		}
	}()
	return out
}
