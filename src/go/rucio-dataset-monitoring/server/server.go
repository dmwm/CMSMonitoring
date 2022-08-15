package server

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>
//
// Inspired by https://github.com/dmwm/dbs2go/blob/master/web/server.go

import (
	"fmt"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/mongo"
	"github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring/utils"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"time"
)

// GitVersion git version comes from Makefile
var GitVersion string

// ServiceInfo defines server info comes from Makefile
var ServiceInfo string

// Serve run service
func Serve(configFile string) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var g errgroup.Group
	err := ParseConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}
	mongo.InitializeMongo(Config.EnvFile, Config.MongoConnectionTimeout)

	utils.Verbose = Config.Verbose

	mongoCollectionNames := Config.CollectionNames
	mainServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", Config.Port),
		Handler:      MainRouter(&mongoCollectionNames),
		ReadTimeout:  time.Duration(Config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(Config.WriteTimeout) * time.Second,
	}

	utils.InfoLogV0("rucio dataset monitoring service is starting %s", nil)
	utils.InfoLogV0("rucio dataset monitoring service is starting with config: %s", Config.String())
	g.Go(func() error {
		return mainServer.ListenAndServe()
	})

	if err := g.Wait(); err != nil {
		log.Printf("[ERROR] server failed %s", err)
	}
}
