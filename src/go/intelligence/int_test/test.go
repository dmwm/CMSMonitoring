package main

import (
	"encoding/json"
	"flag"
	"go/intelligence/models"
	"go/intelligence/pipeline"
	"go/intelligence/utils"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CMS MONIT infrastructure Intelligence Module

func runPipeline() {
	utils.ChangeCounters = models.ChangeCounters{}
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

func pushTestAlerts() {
	log.Printf("Pushing Test alerts into AlertManager \n")

	testAlertData := []models.AmJSON{}

	file, err := os.Open(utils.ConfigJSON.Server.Testing.TestFile)
	if err != nil {
		log.Fatalf("Unable to open JSON file. Testing failed! error: %v\n", err)
	}

	defer file.Close()

	jsonData, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("Unable to read JSON file. Testing failed! error: %v\n", err)
	}

	err = json.Unmarshal(jsonData, &testAlertData)
	if err != nil {
		log.Fatalf("Unable to Unmarshal Data. Testing failed! error: %v\n", err)
	}

	timeNow := time.Now()

	for _, each := range testAlertData {

		each.StartsAt = timeNow
		each.EndsAt = time.Now().Add(utils.ConfigJSON.Server.Testing.LifetimeOfTestAlerts * time.Minute)
		timeNow = timeNow.Add(utils.ConfigJSON.Server.Testing.LifetimeOfTestAlerts * time.Minute * -1) // Adjusting startTime for fake alerts so that they don't overlap in the dashboards

		err := utils.PostAlert(each)
		if err != nil {
			if utils.ConfigJSON.Server.Verbose > 1 {
				log.Printf("Alert Data: %s\n ", each)
			}
			log.Fatalf("Could not push alert. Testing failed! error:%v\n", err)
		}
	}

	log.Printf("Test alerts has been pushed into AlertManager successfully.\n\n")
}

func runTest() {

	log.Printf("Starting Intelligence Pipeline Testing!\n\n")

	pushTestAlerts()
	log.Printf("Data getting persisted in AlertManager...\n\n")
	time.Sleep(10 * time.Second)

	log.Printf("Snapshot of AlertManager before starting Testing... \n")
	snapshotBefore := getAMSnapshot()

	runPipeline()
	log.Printf("Changes made during the pipeline execution.... \n")
	log.Printf("Number of Alerts Pushed : %d \n", utils.ChangeCounters.NoOfPushedAlerts)
	log.Printf("Number of Silences Created : %d \n", utils.ChangeCounters.NoOfSilencesCreated)
	log.Printf("Number of Silences Deleted : %d \n\n", utils.ChangeCounters.NoOfSilencesDeleted)

	log.Printf("Snapshot of AlertManager after completing Testing... \n")
	snapshotAfter := getAMSnapshot()

	if snapshotBefore.NoOfActiveSilences+utils.ChangeCounters.NoOfSilencesCreated != snapshotAfter.NoOfActiveSilences {
		log.Fatalf("Number of Active Silences Mismatched... Testing Failed !!")
	}

	if snapshotBefore.NoOfExpiredSilences+utils.ChangeCounters.NoOfSilencesDeleted != snapshotAfter.NoOfExpiredSilences {
		log.Fatalf("Number of Expired Silences Mismatched... Testing Failed !!")
	}

	if utils.ConfigJSON.Server.Testing.AnnotateTestStatus == false {
		log.Fatalf("Unable to Annotate Dashboard... Testing Failed !!")
	}

	log.Printf("Testing Successful!\n\n")
}

func main() {
	var verbose int
	var configFile string
	flag.StringVar(&configFile, "config", "", "Config File path")
	flag.IntVar(&verbose, "verbose", 0, "Verbosity Level, can be overwritten in config")

	flag.Parse()
	utils.ParseConfig(configFile, verbose)

	runTest()
}

func getAMSnapshot() models.ChangeCounters {
	currentCounters := models.ChangeCounters{}

	data, err := utils.GetAlerts(utils.ConfigJSON.Server.GetAlertsAPI, true)
	if err != nil {
		log.Fatalf("Could not fetch alerts from AlertManager. Testing Failed! error:%v\n", err)
	}
	silenceData, err := utils.GetSilences()
	if err != nil {
		log.Fatalf("Could not fetch silences from AlertManager. Testing Failed! error:%v\n", err)
	}

	currentCounters.NoOfAlerts = len(data.Data)

	for _, each := range silenceData.Data {
		if each.Status.State == utils.ConfigJSON.Silence.SilenceStatus[0] {
			currentCounters.NoOfActiveSilences++
		}
		if each.Status.State == utils.ConfigJSON.Silence.SilenceStatus[1] {
			currentCounters.NoOfExpiredSilences++
		}
		if each.Status.State == utils.ConfigJSON.Silence.SilenceStatus[2] {
			currentCounters.NoOfPendingSilences++
		}
	}

	log.Printf("Number of Alerts : %d\n", currentCounters.NoOfAlerts)
	log.Printf("Number of Active Silences : %d\n", currentCounters.NoOfActiveSilences)
	log.Printf("Number of Expired Silences : %d\n", currentCounters.NoOfExpiredSilences)
	log.Printf("Number of Pending Silences : %d\n\n", currentCounters.NoOfPendingSilences)

	return currentCounters
}
