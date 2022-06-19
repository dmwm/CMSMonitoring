package main

import (
	"flag"
	"fmt"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/configs"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go/controllers"
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
	flag.Parse()
	if version {
		fmt.Println(info())
		os.Exit(0)

	}

	// connect to database
	configs.ConnectDB()
	controllers.GitVersion = gitVersion
	controllers.ServerInfo = info()

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
