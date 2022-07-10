/*
Package main

Rucio dataset monitoring using aggregated Spark data

What is this
    Main aim of the project is to show all Rucio dataset information in a web page with required functionalities

What is used
    DataTables: very popular JQuery library to show pretty tables with nice UI and search/query functionalities
    MongoDB: used to store following data in separate collections
                - aggregated Rucio datasets results,
                - detailed dataset results,
                - short url hash_id:request binding
                - data source timestamp
             multiple MongoDB indexes are created to use full performance of it
    JQuery/JS: to manipulate and customize DataTables, JQuery and JS used

What Go service provide
    `gin-gonic` web framework is used to both
        - MongoDB APIs, see `routes.main_router`
        - serve html pages using Go templates

Main page functionalities
    Main page provides
        - Sort
        - Detailed RSE functionality: green "+" button
        - Paging
        - Count of search result
        - Search using SearchBuilder conditions: "Add condition". Even though SB allows nested conditions, now it supports depth=1
        - Buttons:copy, excel,PDF,column visibility
        - Short URL: which is the advanced functionality of this service. Please see its documentation for more details.

*/
package main

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>
// Reference: https://github.com/gin-gonic/gin/issues/346

import (
	"flag"
	"fmt"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/configs"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/controllers"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/routes"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

var (
	g          errgroup.Group
	gitVersion string // version of the code
	verbose    int
	envFile    string
)

// info function returns version string of the server
func info() string {
	goVersion := runtime.Version()
	tstamp := time.Now().Format("2006-02-01")
	return fmt.Sprintf("rucio-dataset-mon-go git=%s go=%s date=%s", gitVersion, goVersion, tstamp)
}

// main
func main() {
	var version bool
	flag.BoolVar(&version, "version", false, "Show version")
	flag.IntVar(&verbose, "verbose", 0, "Prints verbose logs, 0: warn, 1: info, 2: debug")
	flag.StringVar(&envFile, "env", "", "Environment secret files")
	flag.Parse()

	// Initials
	configs.EnvFile = envFile
	configs.InitialChecks()

	controllers.Verbose = verbose
	controllers.GitVersion = gitVersion
	controllers.ServerInfo = info()
	mongo.InitializeClient()

	if version {
		fmt.Println(info())
		os.Exit(0)
	}
	log.Printf("[INFO] Verbosity : %#v", verbose)
	if verbose > 0 {
		log.Println("[INFO] ---Settings of go web service using MongoDB")
		log.Printf("[INFO] MONGO_URI: %s", mongo.URI)
		log.Printf("[INFO] MONGO_DATABASE: %s", mongo.DB)
		log.Printf("[INFO] MONGO CONNECTION TIMEOUT: %#v", mongo.ConnectionTimeout)
	}

	mainServer := &http.Server{
		Addr:         ":8080",
		Handler:      routes.MainRouter(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	g.Go(func() error {
		return mainServer.ListenAndServe()
	})

	if err := g.Wait(); err != nil {
		log.Printf("[ERROR] SERVER FAILED %s", err)
	}
}
