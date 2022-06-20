package main

import (
	"flag"
	"fmt"
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

// Reference: https://github.com/gin-gonic/gin/issues/346

var (
	g          errgroup.Group
	gitVersion string // version of the code
	verbose    int
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
	flag.Parse()
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
	// connect to database
	mongo.GetMongoClient()
	controllers.GitVersion = gitVersion
	controllers.ServerInfo = info()
	controllers.Verbose = verbose

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
