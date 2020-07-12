package utils

import (
	"encoding/json"
	"fmt"
	"go/intelligence/models"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CMS MONIT infrastructure Intelligence Module

//ConfigJSON variable
var ConfigJSON models.Config

//IfSilencedMap for storing ongoing silences
var IfSilencedMap map[string]int

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
	ConfigJSON.Server.CMSMONURL = "https://cms-monitoring.cern.ch"
	ConfigJSON.Server.GetAlertsAPI = "/api/v1/alerts?active=true&silenced=false&inhibited=false&unprocessed=false"
	ConfigJSON.Server.PostAlertsAPI = "/api/v1/alerts"
	ConfigJSON.Server.GetSilencesAPI = "/api/v1/silences"
	ConfigJSON.Server.PostSilenceAPI = "/api/v1/silences"
	ConfigJSON.Server.HTTPTimeout = 3 //3 secs timeout for HTTP requests
	ConfigJSON.Server.Interval = 10   // 10 sec interval for the service
	ConfigJSON.Server.Verbose = verbose

	ConfigJSON.Silence.Comment = "maintenance"
	ConfigJSON.Silence.CreatedBy = "admin"
	ConfigJSON.Silence.ActiveStatus = "active"

	ConfigJSON.Alerts.UniqueLabel = "alertname"
	ConfigJSON.Alerts.SeverityLabel = "severity"
	ConfigJSON.Alerts.ServiceLabel = "service"
	ConfigJSON.Alerts.DefaultSeverityLevel = "info"

	if stats, err := os.Stat(configFile); err == nil {
		if ConfigJSON.Server.Verbose > 1 {
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
		} else if ConfigJSON.Server.Verbose > 0 {
			log.Printf("Load config from %s\n", configFile)
		}
	} else {
		log.Fatalf("%s: Config File doesn't exist, error: %v", configFile, err)
	}

	if ConfigJSON.Server.Verbose > 0 {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}

	if ConfigJSON.Server.Verbose > 1 {
		log.Printf("Configuration:\n%+v\n", ConfigJSON)
	}

}

//FindDashboard helper function to find dashboard info
// The following block of code was taken from
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/monit.go#L604
func FindDashboard() []map[string]interface{} {
	tags := ParseTags()
	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", ConfigJSON.AnnotationDashboard.Token)
	h := []string{"Authorization", bearer}
	headers = append(headers, h)
	h = []string{"Accept", "application/json"}
	headers = append(headers, h)
	// example: /api/search?query=Production%20Overview&starred=true&tag=prod
	v := url.Values{}
	for _, tag := range tags {
		v.Set("tag", strings.Trim(tag, " "))
	}
	apiURL := fmt.Sprintf("%s%s?%s", ConfigJSON.AnnotationDashboard.URL, ConfigJSON.AnnotationDashboard.DashboardSearchAPI, v.Encode())

	if ConfigJSON.Server.Verbose > 0 {
		log.Println(apiURL)
	}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("Unable to make request to %s, error: %s", apiURL, err)
	}
	for _, v := range headers {
		if len(v) == 2 {
			req.Header.Add(v[0], v[1])
		}
	}
	if ConfigJSON.Server.Verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("request: ", string(dump))
		}
	}

	timeout := time.Duration(ConfigJSON.Server.HTTPTimeout) * time.Second
	client := &http.Client{Timeout: timeout}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to get response from %s, error: %s", apiURL, err)
	}
	if ConfigJSON.Server.Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("response: ", string(dump))
		}
	}
	defer resp.Body.Close()
	var data []map[string]interface{}
	// Deserialize the response into a map.
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("Error parsing the response body: %s", err)
	}
	return data

}

//ParseTags helper function to parse comma separated tags string
// The following block of code was taken from
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/monit.go#L551
func ParseTags() []string {
	var tags []string
	for _, tag := range strings.Split(ConfigJSON.AnnotationDashboard.Tags, ",") {
		tags = append(tags, strings.Trim(tag, " "))
	}
	return tags
}
