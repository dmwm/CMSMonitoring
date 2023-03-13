package server

// Copyright (c) 2023 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>
//
// Inspired by https://github.com/dmwm/dbs2go/blob/master/web/config.go

import (
	"encoding/json"
	"github.com/dmwm/CMSMonitoring/cpueff-goweb/models"
	"log"
	"os"
)

// Config represents global configuration object
var Config models.Configuration

// ParseConfig parses given configuration file and initialize Config object
func ParseConfig(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Println("unable to read config file", configFile, err)
		return err
	}
	err = json.Unmarshal(data, &Config)
	if err != nil {
		log.Println("unable to parse config file", configFile, err)
		return err
	}
	return nil
}
