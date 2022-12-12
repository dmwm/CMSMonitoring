package main

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"flag"
	"fmt"
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/server"
	"os"
	"runtime"
	"time"
)

// gitVersion version of the code which set in build process by Makefile
var gitVersion string //

// info function returns version string of the server
func info() string {
	goVersion := runtime.Version()
	tstamp := time.Now().Format("2006-02-01")
	return fmt.Sprintf("rucio-dataset-monitoring git=%s go=%s date=%s", gitVersion, goVersion, tstamp)
}

// main
func main() {
	var config string
	flag.StringVar(&config, "config", "config.json", "config file")
	var version bool
	flag.BoolVar(&version, "version", false, "Show version")
	flag.Parse()

	if version {
		fmt.Println(info())
		os.Exit(0)
	}

	server.GitVersion = gitVersion
	server.ServiceInfo = info()
	server.Serve(config)
}
