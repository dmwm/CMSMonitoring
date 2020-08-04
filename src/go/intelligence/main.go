package main

import (
	"encoding/json"
	"flag"
	"go/intelligence/models"
	"go/intelligence/pipeline"
	"go/intelligence/utils"
	"log"
	"time"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CMS MONIT infrastructure Intelligence Module

// Function running all logics
// Processing data pipeline module is based on ideas presented in
// https://towardsdatascience.com/concurrent-data-pipelines-in-golang-85b18c2eecc2
func run() {
	var processedData []models.AmJSON
	a := pipeline.DeleteSilence(pipeline.Silence(
		pipeline.PushAlert(
			pipeline.MlBox(
				pipeline.AddAnnotation(
					pipeline.KeywordMatching(
						pipeline.Preprocess(
							pipeline.FetchAlert())))))))

	for d := range a {
		processedData = append(processedData, d)
	}

	if utils.ConfigJSON.Server.Verbose > 2 {
		log.Printf("Processed Alerts Data: %s\n", processedData)
	}
}

func changedCountersLogging() {
	byteVal, err := json.Marshal(utils.ChangeCounters)
	if err != nil {
		log.Printf("Could not parse ChangeCounters struct to JSON, error:%v\n", err)
	}
	log.Printf("Expected Status in AlertManager: %v\n", string(byteVal))

	silencedData, err := utils.GetSilences()
	if err != nil {
		log.Printf("Unable to fetch silences, error: %v", err)
	}
	for _, each := range silencedData.Data {
		if each.Status.State == utils.ConfigJSON.Silence.SilenceStatus[0] {
			utils.ChangeCounters.NoOfActiveSilences--
		}
		if each.Status.State == utils.ConfigJSON.Silence.SilenceStatus[1] {
			utils.ChangeCounters.NoOfExpiredSilences--
		}
		if each.Status.State == utils.ConfigJSON.Silence.SilenceStatus[2] {
			utils.ChangeCounters.NoOfPendingSilences--
		}
	}

	_, err = utils.GetAlerts(utils.ConfigJSON.Server.GetAlertsAPI, true)
	if err != nil {
		log.Printf("Could not fetch alerts from AlertManager, error:%v\n", err)
	}
	utils.ChangeCounters.NoOfAlerts -= len(utils.ExtAlertsMap)

	if utils.ChangeCounters.NoOfAlerts != 0 {
		log.Fatalf("No. of Alerts Mismatched.. Exiting..")
	}
	if utils.ChangeCounters.NoOfActiveSilences != 0 {
		log.Fatalf("No. of Active Silences Mismatched.. Exiting..")
	}
	if utils.ChangeCounters.NoOfExpiredSilences != 0 {
		log.Fatalf("No. of Expired Silences Mismatched.. Exiting..")
	}
}

func runDefinedIterations(iter int) {
	for i := 0; i < iter; i++ {
		run()
		utils.ChangeCounters = models.ChangeCounters{}
		utils.FirstRunSinceRestart = false
		changedCountersLogging()
		time.Sleep(utils.ConfigJSON.Server.Interval * time.Second)
	}
}

func runInfinite() {
	for true {
		utils.ChangeCounters = models.ChangeCounters{}
		run()
		utils.FirstRunSinceRestart = false
		changedCountersLogging()
		time.Sleep(utils.ConfigJSON.Server.Interval * time.Second)
	}
}

func main() {
	var verbose int
	var iter int
	var configFile string
	flag.StringVar(&configFile, "config", "", "Config File path")
	flag.IntVar(&iter, "iter", 0, "Custom defined no. of iterations for premature termination")
	flag.IntVar(&verbose, "verbose", 0, "Verbosity Level, can be overwritten in config")

	flag.Usage = func() {
		log.Println("Usage: intelligence [options]")
		flag.PrintDefaults()
	}

	flag.Parse()
	utils.ParseConfig(configFile, verbose)

	utils.FirstRunSinceRestart = true
	if iter == 0 {
		runInfinite()
	} else {
		runDefinedIterations(iter)
	}
}
