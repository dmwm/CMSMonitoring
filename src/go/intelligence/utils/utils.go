package utils

import (
	"encoding/json"
	"go/intelligence/models"
	"log"
	"net/url"
	"os"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CERN MONIT infrastructure Intelligence Module

//ConfigJSON variable
var ConfigJSON models.Config

//ValidateURL function for constructing and validating AM URL
func ValidateURL(baseURL, apiURL string) string {

	cmpltURL := baseURL + apiURL

	u, err := url.ParseRequestURI(cmpltURL)
	if err != nil {
		log.Fatalf("AlertManager API URL is not valid, error:%v", err)
	}

	return u.String()
}

//ParseConfig Function for parsing the config File
func ParseConfig(configFile string, verbose int) {

	//Defaults in case no config file is provided
	ConfigJSON.CMSMONURL = "https://cms-monitoring.cern.ch"
	ConfigJSON.GetAlertsAPI = "/api/v1/alerts?active=true&silenced=false&inhibited=false&unprocessed=false"
	ConfigJSON.PostAlertsAPI = "/api/v1/alerts"
	ConfigJSON.PostSilenceAPI = "/api/v1/silences"
	ConfigJSON.HTTPTimeout = 3 //3 secs timeout for HTTP requests
	ConfigJSON.Interval = 10   // 10 sec interval for the service
	ConfigJSON.Verbose = verbose

	ConfigJSON.UniqueLabel = "alertname"
	ConfigJSON.Comment = "maintenance"
	ConfigJSON.CreatedBy = "admin"

	ConfigJSON.SeverityLabel = "severity"

	ConfigJSON.SsbKeywordLabel = "shortDescription"
	ConfigJSON.DefaultSSBSeverityLevel = "info"

	ConfigJSON.GGUSKeywordLabel = "Priority"
	ConfigJSON.DefaultGGUSSeverityLevel = "ticket"

	ConfigJSON.DefaultSeverityLevel = "info"

	if stats, err := os.Stat(configFile); err == nil {
		if ConfigJSON.Verbose > 1 {
			log.Printf("FileInfo: %s\n", stats)
		}
		jsonFile, e := os.Open(configFile)
		if e != nil {
			log.Fatalf("Config File not found, error: %s", e)
		}
		defer jsonFile.Close()
		decoder := json.NewDecoder(jsonFile)
		err := decoder.Decode(&ConfigJSON)
		if err != nil {
			log.Fatalf("Config JSON File can't be loaded, error: %s", err)
		} else if ConfigJSON.Verbose > 0 {
			log.Printf("Load config from %s\n", configFile)
		}
	} else {
		log.Fatalf("%s: Config File doesn't exist, error: %v", configFile, err)
	}

	if ConfigJSON.Verbose > 0 {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}

	if ConfigJSON.Verbose > 1 {
		log.Printf("Configuration:\n%+v\n", ConfigJSON)
	}

}
