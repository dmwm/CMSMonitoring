package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"
)

// File       : intelligence.go
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Thu, 18 June 2020 13:02:19 GMT
// Description: CERN MONIT infrastructure Intelligence Module

//--------MAPS----------
//Map for storing alertData with instance as key.
var silenceMap map[string]amJSON

//--------MAPS----------

//-------STRUCTS---------
//AlertManager API acceptable JSON Data for GGUS Data
type amJSON struct {
	Labels      map[string]interface{} `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`
	StartsAt    time.Time              `json:"startsAt"`
	EndsAt      time.Time              `json:"endsAt"`
}

type amData struct {
	Data []amJSON
}

//Alert CLI tool data struct (Tabular)
type alertData struct {
	Name     string
	Service  string
	Tag      string
	Severity string
	StartsAt time.Time
	EndsAt   time.Time
}

//Array of alerts for alert CLI Tool (Tabular)
var allAlertData []alertData

type matchers struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type silenceData struct {
	Matchers  []matchers `json:"matchers"`
	StartsAt  time.Time  `json:"startsAt"`
	EndsAt    time.Time  `json:"endsAt"`
	CreatedBy string     `json:"createdBy"`
	Comment   string     `json:"comment"`
}

type config struct {
	CMSMONURL      string        `json:"cmsmonURL"`
	GetAlertsAPI   string        `json:"getAlertsAPI"`
	PostSilenceAPI string        `json:"postSilenceAPI"`
	HttpTimeout    int           `json:"httpTimeout"`
	Interval       time.Duration `json:"interval"`
	CreatedBy      string        `json:"createdBy"`
	SeverityFilter string        `json:"severityFilter"`
	SearchingLabel string        `json:"searchingLabel"`
	UniqueLabel    string        `json:"uniqueLabel"`
	Comment        string        `json:"comment"`
	Verbose        int           `json:"verbose"`
}

var configJSON config

//-------STRUCTS---------

//function for get request on /api/v1/alerts alertmanager endpoint for fetching alerts.
func get(data interface{}) {

	//GET API for fetching only GGUS alerts.
	apiurl := configJSON.CMSMONURL + configJSON.GetAlertsAPI

	req, err := http.NewRequest("GET", apiurl, nil)
	req.Header.Add("Accept-Encoding", "identity")
	req.Header.Add("Accept", "application/json")

	timeout := time.Duration(configJSON.HttpTimeout) * time.Second
	client := &http.Client{Timeout: timeout}

	if configJSON.Verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("Request: ", string(dump))
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	byteValue, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Unable to read JSON Data from AlertManager GET API, error: %v\n", err)
		return
	}

	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		if configJSON.Verbose > 0 {
			log.Println(string(byteValue))
		}
		log.Fatalf("Unable to parse JSON Data from AlertManager GET API, error: %v\n", err)
	}

	if configJSON.Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}

}

func silence(data amJSON, mData amJSON) {
	// GET API for fetching only GGUS alerts.
	apiurl := configJSON.CMSMONURL + configJSON.PostSilenceAPI

	var sData silenceData
	sData.StartsAt = mData.StartsAt //Start Time equal to maintenance alert	   //So that when maintenance alerts vanishes, ongoing alerts comes back alive.
	sData.EndsAt = mData.EndsAt     //End Time equal to maintenance alert	//So that when maintenance alerts vanishes, ongoing alerts comes back alive.
	sData.CreatedBy = configJSON.CreatedBy
	sData.Comment = configJSON.Comment
	var m matchers

	m.Name = configJSON.UniqueLabel
	for k, v := range data.Labels {
		if k == configJSON.UniqueLabel {
			m.Value = v.(string)
		}
	}

	sData.Matchers = append(sData.Matchers, m)
	jsonStr, err := json.Marshal(sData)

	if err != nil {
		log.Printf("Unable to convert JSON Data, error: %v\n", err)
	}

	req, err := http.NewRequest("POST", apiurl, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	timeout := time.Duration(configJSON.HttpTimeout) * time.Second
	client := &http.Client{Timeout: timeout}

	if configJSON.Verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("Request: ", string(dump))
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if configJSON.Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}

	if configJSON.Verbose > 1 {
		log.Printf("Silence Data:\n%+v\n", string(jsonStr))
	}
}

//Function for silencing maintenance false alerts
func silenceMaintenance(filteredAlerts amData) {

	if configJSON.Verbose > 1 {
		if len(filteredAlerts.Data) == 0 {
			log.Printf("No Maintenance Alert Found")
			return
		}
	}

	ifAnyAlert := false

	for _, each := range filteredAlerts.Data {
		for k, v := range each.Labels {
			if k == configJSON.SearchingLabel {
				for _, ins := range strings.Split(v.(string), ",") {
					if val, ok := silenceMap[ins]; ok {
						ifAnyAlert = true
						silence(val, each)
					}
				}
			}
		}
	}

	if configJSON.Verbose > 1 {
		if ifAnyAlert == false {
			log.Printf("No Alert found for Silencing")
		}
	}
}

//Function for filtering maintenance alerts
func filterMaintenance(amdata amData) amData {
	silenceMap = make(map[string]amJSON)
	var maintenanceData amData

	for _, each := range amdata.Data {
		for k, v := range each.Labels {
			if k == "severity" && v == configJSON.SeverityFilter {
				maintenanceData.Data = append(maintenanceData.Data, each)
			}
			if k == configJSON.SearchingLabel {
				if len(strings.Split(v.(string), ",")) == 1 {
					silenceMap[v.(string)] = each
				}
			}
		}
	}
	return maintenanceData
}

//Function running all logics
func run() {

	var amdata amData
	get(&amdata)

	filtered := filterMaintenance(amdata)
	silenceMaintenance(filtered)

}

func parseConfig(verbose int) {

	configFile := os.Getenv("INTELLIGENCE_CONFIG_PATH") //INTELLIGENCE_CONFIG_PATH Environment Variable storing config filepath.

	//Defaults in case no config file is provided
	configJSON.CMSMONURL = "https://cms-monitoring.cern.ch"
	configJSON.GetAlertsAPI = "/api/v1/alerts?active=true&silenced=false&inhibited=false&unprocessed=false"
	configJSON.PostSilenceAPI = "/api/v1/silences"
	configJSON.SeverityFilter = "maintenance"
	configJSON.SearchingLabel = "instance"
	configJSON.UniqueLabel = "alertname"
	configJSON.Comment = "maintenance"
	configJSON.CreatedBy = "admin"
	configJSON.HttpTimeout = 3 //3 secs timeout for HTTP requests
	configJSON.Interval = 10   // 10 sec interval for the service
	configJSON.Verbose = verbose

	if configFile != "" {
		jsonFile, e := os.Open(configFile)
		if e != nil {
			log.Fatalf("Config File not found, error: %s", e)
		}
		defer jsonFile.Close()
		decoder := json.NewDecoder(jsonFile)
		err := decoder.Decode(&configJSON)
		if err != nil {
			log.Fatalf("Config JSON File can't be loaded, error: %s", err)
		} else if configJSON.Verbose > 0 {
			log.Printf("Load config from %s\n", configFile)
		}
	}

	if configJSON.Verbose > 0 {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}

	if configJSON.Verbose > 1 {
		log.Printf("Configuration:\n%+v\n", configJSON)
	}

}

func runInfinite() {
	for true {
		run()
		time.Sleep(configJSON.Interval * time.Second)
	}
}

func main() {

	var verbose int
	flag.IntVar(&verbose, "verbose", 0, "Verbosity Level, can be overwritten in config")

	flag.Usage = func() {
		fmt.Println("Usage: intelligence [options]")
		flag.PrintDefaults()
		fmt.Println("\nEnvironments:")
		fmt.Println("\tINTELLIGENCE_CONFIG_PATH:\t Config Filepath")
	}

	flag.Parse()
	parseConfig(verbose)
	runInfinite()
}
