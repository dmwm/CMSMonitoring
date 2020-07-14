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

//Function running all logics
func run() {
	var processedData []models.AmJSON
	a := pipeline.Silence(pipeline.PushAlert(
		pipeline.MlBox(
			pipeline.AddAnnotation(
				pipeline.KeywordMatching(
					pipeline.Preprocess(
						pipeline.FetchAlert()))))))

	for d := range a {
		processedData = append(processedData, d)
	}

	if utils.ConfigJSON.Server.Verbose > 2 {
		log.Printf("Processed Alerts Data: %s\n", processedData)
	}
}

func runInfinite() {
	utils.FirstRunSinceRestart = true
	for true {
		run()
		utils.FirstRunSinceRestart = false
		time.Sleep(utils.ConfigJSON.Server.Interval * time.Second)
	}
}

func main() {

	var verbose int
	var configFile string
	flag.StringVar(&configFile, "config", "", "Config File path")
	flag.IntVar(&verbose, "verbose", 0, "Verbosity Level, can be overwritten in config")

	flag.Usage = func() {
		log.Println("Usage: intelligence [options]")
		flag.PrintDefaults()
	}

	flag.Parse()
	utils.ParseConfig(configFile, verbose)

	runInfinite()
}
