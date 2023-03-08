package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"time"
)

// File       : intelligence.go
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Thu, 18 June 2020 13:02:19 GMT
// Description: CERN MONIT infrastructure Intelligence Module

// --------MAPS----------
// Map for storing alertData with instance as key.
var silenceMap map[string][]amJSON

//--------MAPS----------

// -------STRUCTS---------
// AlertManager API acceptable JSON Data for GGUS Data
type amJSON struct {
	Labels      map[string]interface{} `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`
	StartsAt    time.Time              `json:"startsAt"`
	EndsAt      time.Time              `json:"endsAt"`
}

type amData struct {
	Data []amJSON
}

// Alert CLI tool data struct (Tabular)
type alertData struct {
	Name     string
	Service  string
	Tag      string
	Severity string
	StartsAt time.Time
	EndsAt   time.Time
}

// Array of alerts for alert CLI Tool (Tabular)
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
	Seperator      string        `json:"seperator"`
	Comment        string        `json:"comment"`
	Verbose        int           `json:"verbose"`
}

var configJSON config

//-------STRUCTS---------

// function for constructing and validating AM URL
func construct(baseURL, apiURL string) string {

	cmpltURL := baseURL + apiURL

	u, err := url.ParseRequestURI(cmpltURL)
	if err != nil {
		log.Fatalf("AlertManager API URL is not valid, error:%v", err)
	}

	return u.String()
}

// function for get request on /api/v1/alerts alertmanager endpoint for fetching alerts.
func get(data interface{}) error {

	//GET API for fetching all AM alerts.
	apiurl := construct(configJSON.CMSMONURL, configJSON.GetAlertsAPI)

	req, reqErr := http.NewRequest("GET", apiurl, nil)
	if reqErr != nil {
		return reqErr
	}
	req.Header.Add("Accept-Encoding", "identity")
	req.Header.Add("Accept", "application/json")

	timeout := time.Duration(configJSON.HttpTimeout) * time.Second
	client := &http.Client{Timeout: timeout}

	if configJSON.Verbose > 1 {
		dump, dumpErr := httputil.DumpRequestOut(req, true)
		if dumpErr == nil {
			log.Println("Request: ", string(dump))
		}
	}

	resp, respErr := client.Do(req)
	if respErr != nil {
		return respErr
	} else {
		if resp.StatusCode != http.StatusOK {
			log.Printf("Http Response Code Error, status code: %d", resp.StatusCode)
			return errors.New("http Response Code Error")
		}
	}
	defer resp.Body.Close()

	byteValue, bvErr := io.ReadAll(resp.Body)

	if bvErr != nil {
		log.Printf("Unable to read JSON Data from AlertManager GET API, error: %v\n", bvErr)
		return bvErr
	}

	jsonErr := json.Unmarshal(byteValue, &data)
	if jsonErr != nil {
		if configJSON.Verbose > 0 {
			log.Println(string(byteValue))
		}
		log.Printf("Unable to parse JSON Data from AlertManager GET API, error: %v\n", jsonErr)
		return jsonErr
	}

	if configJSON.Verbose > 1 {
		dump, dumpErr := httputil.DumpResponse(resp, true)
		if dumpErr == nil {
			log.Println("Response: ", string(dump))
		}
	}

	return nil
}

// Function for post request on /api/v1/silences alertmanager endpoint for creating silences.
func silence(data amJSON, mData amJSON) error {
	// POST API for creating silences.
	apiurl := construct(configJSON.CMSMONURL, configJSON.PostSilenceAPI)

	var sData silenceData
	sData.StartsAt = mData.StartsAt //Start Time equal to maintenance alert	   //So that when maintenance alerts vanishes, ongoing alerts comes back alive.
	sData.EndsAt = mData.EndsAt     //End Time equal to maintenance alert	//So that when maintenance alerts vanishes, ongoing alerts comes back alive.
	sData.CreatedBy = configJSON.CreatedBy
	sData.Comment = configJSON.Comment
	var m matchers

	m.Name = configJSON.UniqueLabel
	for k, v := range data.Labels {
		if k == configJSON.UniqueLabel {
			if val, ok := v.(string); ok {
				m.Value = val
			}
		}
	}

	sData.Matchers = append(sData.Matchers, m)
	jsonStr, jsonErr := json.Marshal(sData)

	if jsonErr != nil {
		log.Printf("Unable to convert JSON Data, error: %v\n", jsonErr)
		return jsonErr
	}

	req, reqErr := http.NewRequest("POST", apiurl, bytes.NewBuffer(jsonStr))
	if reqErr != nil {
		return reqErr
	}
	req.Header.Set("Content-Type", "application/json")

	timeout := time.Duration(configJSON.HttpTimeout) * time.Second
	client := &http.Client{Timeout: timeout}

	if configJSON.Verbose > 1 {
		dump, dumpErr := httputil.DumpRequestOut(req, true)
		if dumpErr == nil {
			log.Println("Request: ", string(dump))
		}
	}

	resp, respErr := client.Do(req)
	if respErr != nil {
		return respErr
	} else {
		if resp.StatusCode != http.StatusOK {
			log.Printf("Http Response Code Error, status code: %d", resp.StatusCode)
			return errors.New("http Response Code Error")
		}
	}
	defer resp.Body.Close()

	if configJSON.Verbose > 1 {
		dump, dumpErr := httputil.DumpResponse(resp, true)
		if dumpErr == nil {
			log.Println("Response: ", string(dump))
		}
	}

	if configJSON.Verbose > 1 {
		log.Printf("Silence Data:\n%+v\n", string(jsonStr))
	}

	return nil
}

// Function for silencing maintenance false alerts
func silenceMaintenance(filteredAlerts amData) {

	if configJSON.Verbose > 1 {
		if len(filteredAlerts.Data) == 0 {
			log.Printf("No Maintenance Alert Found")
			return
		} else {
			log.Printf("Maintenance Alert Data:\n%v\n\n", filteredAlerts)
		}
	}

	ifAnyAlert := false

	for _, each := range filteredAlerts.Data {
		for k, v := range each.Labels {
			if k == configJSON.SearchingLabel {
				for _, ins := range regexp.MustCompile("["+configJSON.Seperator+"\\,\\s]+").Split(v.(string), -1) {
					if ins != "" {
						re, err := regexp.Compile(ins + ":\\d*|" + ins + "$")
						if err != nil {
							log.Fatalf("Regex didn't compile %v", err)
						}
						for silenceMapKey, silenceMapValues := range silenceMap {
							if re.Match([]byte(silenceMapKey)) {
								ifAnyAlert = true
								for _, silenceMapValue := range silenceMapValues {
									silenceErr := silence(silenceMapValue, each)
									if silenceErr != nil {
										log.Printf("Could not silence, error: %v\n", silenceErr)
										if configJSON.Verbose > 1 {
											log.Printf("Silence Data: %s\n ", silenceMapValue)
										}
									}
								}
							}
						}
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

// Function for filtering maintenance alerts
func filterMaintenance(amdata amData) amData {
	silenceMap = make(map[string][]amJSON)
	var maintenanceData amData

	for _, each := range amdata.Data {
		for k, v := range each.Labels {
			if k == "severity" && v == configJSON.SeverityFilter {
				maintenanceData.Data = append(maintenanceData.Data, each)
			} else {
				if k == configJSON.SearchingLabel {
					if len(regexp.MustCompile("["+configJSON.Seperator+"\\,\\s]+").Split(v.(string), -1)) == 1 {
						silenceMap[v.(string)] = append(silenceMap[v.(string)], each)
					}
				}
			}
		}
	}
	return maintenanceData
}

// Function running all logics
func run() {

	var amdata amData
	getErr := get(&amdata)

	if getErr != nil {
		log.Fatalf("Could not get alerts. %v", getErr)
	}

	filtered := filterMaintenance(amdata)
	silenceMaintenance(filtered)

}

// Function for parsing the config File
func parseConfig(configFile string, verbose int) {

	//Defaults in case no config file is provided
	configJSON.CMSMONURL = "https://cms-monitoring.cern.ch"
	configJSON.GetAlertsAPI = "/api/v1/alerts?active=true&silenced=false&inhibited=false&unprocessed=false"
	configJSON.PostSilenceAPI = "/api/v1/silences"
	configJSON.SeverityFilter = "maintenance"
	configJSON.SearchingLabel = "instance"
	configJSON.UniqueLabel = "alertname"
	configJSON.Seperator = " "
	configJSON.Comment = "maintenance"
	configJSON.CreatedBy = "admin"
	configJSON.HttpTimeout = 3 //3 secs timeout for HTTP requests
	configJSON.Interval = 10   // 10 sec interval for the service
	configJSON.Verbose = verbose

	if stats, err := os.Stat(configFile); err == nil {
		if configJSON.Verbose > 1 {
			log.Printf("FileInfo: %s\n", stats)
		}
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
	} else {
		log.Fatalf("%s: Config File doesn't exist, error: %v", configFile, err)
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

// Function for running the logic on a time interval
func runInfinite() {
	for true {
		run()
		time.Sleep(configJSON.Interval * time.Second)
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
	parseConfig(configFile, verbose)
	runInfinite()
}
