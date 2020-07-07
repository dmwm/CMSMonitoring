package main

import (
	"flag"
	"fmt"
	"go/intelligence/pipeline"
	"go/intelligence/utils"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CERN MONIT infrastructure Intelligence Module

//Function running all logics
func run() {
	a := pipeline.Silence(pipeline.PushAlert(
		pipeline.MlBox(
			pipeline.KeywordMatching(
				pipeline.Preprocess(
					pipeline.FetchAlert())))))

	if utils.ConfigJSON.Verbose > 2 {
		fmt.Println("Processed Alerts Data:")
		for d := range a {
			fmt.Println(d)
		}
	}
}

func main() {

	var verbose int
	var configFile string
	flag.StringVar(&configFile, "config", "", "Config File path")
	flag.IntVar(&verbose, "verbose", 0, "Verbosity Level, can be overwritten in config")

	flag.Usage = func() {
		fmt.Println("Usage: intelligence [options]")
		flag.PrintDefaults()
	}

	flag.Parse()
	utils.ParseConfig(configFile, verbose)

	run()
}
