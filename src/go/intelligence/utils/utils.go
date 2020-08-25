package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/intelligence/models"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CMS MONIT infrastructure Intelligence Module

//ConfigJSON variable
var ConfigJSON models.Config

//SilenceMapVals struct for storing ifAvail bool and silenceID in IfSilencedMap
type SilenceMapVals struct {
	IfAvail   int
	SilenceID string
}

//ChangeCounters variable for storing counters for logging before and after running intelligence module
var ChangeCounters models.ChangeCounters

//IfSilencedMap - variable for storing ongoing silences
var IfSilencedMap map[string]SilenceMapVals

//ExtAlertsMap - map for storing existing Alerts in AlertManager
var ExtAlertsMap map[string]int

//ExtSuppressedAlertsMap - map for storing existing suppressed Alerts in AlertManager
var ExtSuppressedAlertsMap map[string]models.AmJSON

//FirstRunSinceRestart - boolean variable for storing the information if the it's the first start of the service after restart or not
var FirstRunSinceRestart bool

//DataReadWriteLock - Mutex Lock for Concurrent Read/Write on Alert Data
var DataReadWriteLock sync.RWMutex

//SuppressedAlertsDataReadWriteLock - Mutex Lock for Concurrent Read/Write on Suppressed Alert Data
var SuppressedAlertsDataReadWriteLock sync.RWMutex

//DashboardsCache - a cache for storing dashboards and Expiration time for updating the cache
type DashboardsCache struct {
	Dashboards map[float64]models.AllDashboardsFetched
	Expiration time.Time
}

//UpdateDashboardCache - function for updating the dashboards cache on expiration
func (dCache *DashboardsCache) UpdateDashboardCache() {

	if !FirstRunSinceRestart && dCache.Expiration.After(time.Now()) {
		return
	}

	dCache.Dashboards = make(map[float64]models.AllDashboardsFetched)

	for _, tag := range ConfigJSON.AnnotationDashboard.Tags {
		tmp := findDashboards(tag)

		for _, each := range tmp {
			dCache.Dashboards[each.ID] = each
		}
	}

	dCache.Expiration = time.Now().Add(ConfigJSON.AnnotationDashboard.DashboardsCacheExpiration * time.Hour)
}

//DCache - variable for DashboardsCache
var DCache DashboardsCache

//ValidateURL - function for constructing and validating AM URL
func ValidateURL(baseURL, apiURL string) string {

	cmpltURL := baseURL + apiURL

	u, err := url.ParseRequestURI(cmpltURL)
	if err != nil {
		log.Fatalf("AlertManager API URL is not valid, error:%v", err)
	}

	return u.String()
}

//ParseConfig - Function for parsing the config File
func ParseConfig(configFile string, verbose int) {

	//Defaults in case no config file is provided
	ConfigJSON.Server.CMSMONURL = "https://cms-monitoring.cern.ch"
	ConfigJSON.Server.GetAlertsAPI = "/api/v1/alerts?active=true&silenced=false&inhibited=false&unprocessed=false"
	ConfigJSON.Server.GetSuppressedAlertsAPI = "/api/v1/alerts?active=false&silenced=true"
	ConfigJSON.Server.PostAlertsAPI = "/api/v1/alerts"
	ConfigJSON.Server.GetSilencesAPI = "/api/v1/silences"
	ConfigJSON.Server.PostSilenceAPI = "/api/v1/silences"
	ConfigJSON.Server.DeleteSilenceAPI = "/api/v1/silence"
	ConfigJSON.Server.HTTPTimeout = 3 //3 secs timeout for HTTP requests
	ConfigJSON.Server.Interval = 10   // 10 sec interval for the service
	ConfigJSON.Server.Verbose = verbose

	ConfigJSON.AnnotationDashboard.URL = "https://monit-grafana.cern.ch"
	ConfigJSON.AnnotationDashboard.DashboardSearchAPI = "/api/search"
	ConfigJSON.AnnotationDashboard.AnnotationAPI = "/api/annotations"
	ConfigJSON.AnnotationDashboard.DashboardsCacheExpiration = 1
	ConfigJSON.AnnotationDashboard.IntelligenceModuleTag = "cmsmon-int"

	ConfigJSON.Alerts.UniqueLabel = "alertname"
	ConfigJSON.Alerts.SeverityLabel = "severity"
	ConfigJSON.Alerts.ServiceLabel = "service"
	ConfigJSON.Alerts.DefaultSeverityLevel = "info"

	ConfigJSON.Silence.Comment = "maintenance"
	ConfigJSON.Silence.CreatedBy = "admin"

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

	//Custom verbose value overriden
	if verbose > 0 {
		ConfigJSON.Server.Verbose = verbose
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

//findDashboard - helper function to find dashboard info
// The following block of code was taken from
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/monit.go#L604
func findDashboards(tag string) []models.AllDashboardsFetched {
	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", ConfigJSON.AnnotationDashboard.Token)
	h := []string{"Authorization", bearer}
	headers = append(headers, h)
	h = []string{"Accept", "application/json"}
	headers = append(headers, h)
	// example: /api/search?query=Production%20Overview&starred=true&tag=prod
	v := url.Values{}
	v.Set("tag", strings.Trim(tag, " "))
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
	var data []models.AllDashboardsFetched
	// Deserialize the response into a map.
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		d, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Error parsing the response body: %s, %+v\n", err, string(d))
	}
	return data
}

//GetSilences - function for get request on /api/v1/silences alertmanager endpoint for fetching all silences.
func GetSilences() (models.AllSilences, error) {
	var data models.AllSilences
	apiurl := ValidateURL(ConfigJSON.Server.CMSMONURL, ConfigJSON.Server.GetSilencesAPI) //GET API for fetching all AM Silences.

	req, err := http.NewRequest("GET", apiurl, nil)
	if err != nil {
		log.Printf("Request Error, error: %v\n", err)
		return data, err
	}
	req.Header.Add("Accept-Encoding", "identity")
	req.Header.Add("Accept", "application/json")

	timeout := time.Duration(ConfigJSON.Server.HTTPTimeout) * time.Second
	client := &http.Client{Timeout: timeout}

	if ConfigJSON.Server.Verbose > 1 {
		log.Println("GET", apiurl)
	} else if ConfigJSON.Server.Verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("Request: ", string(dump))
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Response Error, error: %v\n", err)
		return data, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Http Response Code Error, status code: %d", resp.StatusCode)
		return data, errors.New("Respose Error")
	}

	defer resp.Body.Close()

	byteValue, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Unable to read JSON Data from AlertManager Silence GET API, error: %v\n", err)
		return data, err
	}

	if ConfigJSON.Server.Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}

	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		if ConfigJSON.Server.Verbose > 0 {
			log.Println(string(byteValue))
		}
		log.Printf("Unable to parse JSON Data from AlertManager Silence GET API, error: %v\n", err)
		return data, err
	}

	return data, nil
}

//GetAlerts - function for get request on /api/v1/alerts alertmanager endpoint for fetching alerts.
func GetAlerts(getAlertsAPI string, updateMapChoice bool) (models.AmData, error) {

	var data models.AmData
	apiurl := ValidateURL(ConfigJSON.Server.CMSMONURL, getAlertsAPI) //GET API for fetching all AM alerts.

	req, err := http.NewRequest("GET", apiurl, nil)
	if err != nil {
		log.Printf("Request Error, error: %v\n", err)
		return data, err
	}
	req.Header.Add("Accept-Encoding", "identity")
	req.Header.Add("Accept", "application/json")

	timeout := time.Duration(ConfigJSON.Server.HTTPTimeout) * time.Second
	client := &http.Client{Timeout: timeout}

	if ConfigJSON.Server.Verbose > 1 {
		log.Println("GET", apiurl)
	} else if ConfigJSON.Server.Verbose > 2 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("Request: ", string(dump))
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Response Error, error: %v\n", err)
		return data, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Http Response Code Error, status code: %d", resp.StatusCode)
		return data, errors.New("Respose Error")
	}

	defer resp.Body.Close()

	byteValue, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Unable to read JSON Data from AlertManager GET API, error: %v\n", err)
		return data, err
	}

	if ConfigJSON.Server.Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}

	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		if ConfigJSON.Server.Verbose > 0 {
			log.Println(string(byteValue))
		}
		log.Printf("Unable to parse JSON Data from AlertManager GET API, error: %v\n", err)
		return data, err
	}

	if updateMapChoice == true {
		DataReadWriteLock.Lock()
		defer DataReadWriteLock.Unlock()
		ExtAlertsMap = make(map[string]int)
		for _, eachAlert := range data.Data {
			for k, v := range eachAlert.Labels {
				if k == ConfigJSON.Alerts.UniqueLabel {
					if val, ok := v.(string); ok {
						ExtAlertsMap[val] = 1
					}
				}
			}
		}
	} else {
		SuppressedAlertsDataReadWriteLock.Lock()
		defer SuppressedAlertsDataReadWriteLock.Unlock()
		ExtSuppressedAlertsMap = make(map[string]models.AmJSON)
		for _, eachAlert := range data.Data {
			for k, v := range eachAlert.Labels {
				if k == ConfigJSON.Alerts.UniqueLabel {
					if val, ok := v.(string); ok {
						ExtSuppressedAlertsMap[val] = eachAlert
					}
				}
			}
		}
	}

	return data, nil
}

//PostAlert - function for making post request on /api/v1/alerts alertmanager endpoint for creating alerts.
func PostAlert(data models.AmJSON) error {
	apiurl := ValidateURL(ConfigJSON.Server.CMSMONURL, ConfigJSON.Server.PostAlertsAPI)
	var finalData []models.AmJSON
	finalData = append(finalData, data)

	jsonStr, err := json.Marshal(finalData)
	if err != nil {
		log.Printf("Unable to convert JSON Data, error: %v\n", err)
		return err
	}

	req, err := http.NewRequest("POST", apiurl, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Printf("Request Error, error: %v\n", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	timeout := time.Duration(ConfigJSON.Server.HTTPTimeout) * time.Second
	client := &http.Client{Timeout: timeout}

	if ConfigJSON.Server.Verbose > 1 {
		log.Println("POST", apiurl)
	} else if ConfigJSON.Server.Verbose > 2 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("Request: ", string(dump))
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Response Error, error: %v\n", err)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Http Response Code Error, status code: %d", resp.StatusCode)
		return errors.New("Http Response Code Error")
	}

	defer resp.Body.Close()

	if ConfigJSON.Server.Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}

	if ConfigJSON.Server.Verbose > 1 {
		log.Println("Pushed Alerts: ", string(jsonStr))
	}
	return nil
}
