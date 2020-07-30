package main

import (
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

//variable storing custom defined no. of iteration
var iter int

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

func runNoOfTimes() {
	utils.NoOfAlertsBeforeIntModule = 0
	utils.NoOfPushedAlerts = 0
	utils.NoOfNewSilencedAlerts = 0
	utils.NoOfAlertsAfterIntModule = 0
	utils.NoOfActiveSilencesAfterIntModule = 0
	utils.NoOfExpiredSilencesAfterIntModule = 0
	var err error

	run()

	utils.FirstRunSinceRestart = false
	log.Printf("No of Alerts found into AlertManager before running int module : %d\n", utils.NoOfAlertsBeforeIntModule)
	log.Printf("No of Active Silences in AlertManager before runnint int module : %d\n", utils.NoOfActiveSilencesBeforeIntModule)
	log.Printf("No of Expired Silences in AlertManager before runnint int module : %d\n", utils.NoOfExpiredSilencesBeforeIntModule)
	log.Printf("No of Pending Silences in AlertManager before runnint int module : %d\n", utils.NoOfPendingSilencesBeforeIntModule)

	if *utils.DryRun == false {
		log.Printf("No of Alerts pushed into AlertManager : %d\n", utils.NoOfPushedAlerts)
		log.Printf("No of new Silences created into AlertManager : %d\n", utils.NoOfNewSilencedAlerts)
		_, utils.NoOfAlertsAfterIntModule, err = utils.GetAlerts(utils.ConfigJSON.Server.GetAlertsAPI, true)
		if err != nil {
			log.Printf("Could not fetch alerts from AlertManager, error:%v\n", err)
		}
		log.Printf("No of alerts in AlertManager after runnint int module : %d", utils.NoOfAlertsAfterIntModule)
		_, utils.NoOfActiveSilencesAfterIntModule, utils.NoOfExpiredSilencesAfterIntModule, utils.NoOfPendingSilencesAfterIntModule, err = utils.GetSilences()
		if err != nil {
			log.Printf("Could not fetch silences from AlertManager, error:%v\n", err)
		}
		log.Printf("No of Active Silences in AlertManager after runnint int module : %d\n", utils.NoOfActiveSilencesAfterIntModule)
		log.Printf("No of Expired Silences in AlertManager after runnint int module : %d\n", utils.NoOfExpiredSilencesAfterIntModule)
		log.Printf("No of Pending Silences in AlertManager after runnint int module : %d\n", utils.NoOfPendingSilencesAfterIntModule)
	} else {
		log.Printf("Dry Run.. No of Alerts meant to be pushed into AlertManager : %d\n", utils.NoOfPushedAlerts)
		log.Printf("Dry Run.. No of new Silences meant to be created into AlertManager : %d\n", utils.NoOfNewSilencedAlerts)
	}

	time.Sleep(utils.ConfigJSON.Server.Interval * time.Second)
}

func runDefinedIterations() {
	for i := 0; i < iter; i++ {
		runNoOfTimes()
	}
}

func runInfinite() {
	for true {
		runNoOfTimes()
	}
}

func main() {

	var verbose int
	var configFile string
	flag.StringVar(&configFile, "config", "", "Config File path")
	utils.DryRun = flag.Bool("dryRun", false, "Flag for dry running")
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
		runDefinedIterations()
	}
}
