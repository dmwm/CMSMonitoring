package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dmwm/CMSMonitoring/src/go/intelligence/models"
	"io"
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

// ConfigJSON variable
var ConfigJSON models.Config

// SilenceMapVals struct for storing ifAvail bool and silenceID in IfSilencedMap
type SilenceMapVals struct {
	IfAvail   int
	SilenceID string
}

// ChangeCounters variable for storing counters for logging before and after running intelligence module
var ChangeCounters models.ChangeCounters

// IfSilencedMap - variable for storing ongoing silences
var IfSilencedMap map[string]SilenceMapVals

// ExtAlertsMap - map for storing existing Alerts in AlertManager
var ExtAlertsMap map[string]int

// ExtSuppressedAlertsMap - map for storing existing suppressed Alerts in AlertManager
var ExtSuppressedAlertsMap map[string]models.AmJSON

// FirstRunSinceRestart - store information if the it's the first start of the service after restart or not
var FirstRunSinceRestart bool

// DataReadWriteLock variable for solving Concurrent Read/Write on Map issue.
var DataReadWriteLock sync.RWMutex

// DCache - variable for DashboardsCache
var DCache DashboardsCache

// DashboardsCache - a cache for storing dashboards and Expiration time for updating the cache
type DashboardsCache struct {
	Dashboards map[float64]models.AllDashboardsFetched
	Expiration time.Time
}

// UpdateDashboardCache - function for updating the dashboards cache on expiration
func (dCache *DashboardsCache) UpdateDashboardCache() {

	if !FirstRunSinceRestart && dCache.Expiration.After(time.Now()) {
		return
	}

	DataReadWriteLock.Lock()
	defer DataReadWriteLock.Unlock()
	dCache.Dashboards = make(map[float64]models.AllDashboardsFetched)

	for _, tag := range ConfigJSON.AnnotationDashboard.Tags {
		tmp := findDashboards(tag)

		for _, each := range tmp {
			dCache.Dashboards[each.ID] = each
		}
	}
	if ConfigJSON.Server.Verbose > 0 {
		log.Println("updated dashboard cache with", len(dCache.Dashboards), "maps")
		if ConfigJSON.Server.Verbose > 1 {
			for _, d := range dCache.Dashboards {
				log.Println(d.String())
			}
		}
	}

	dCache.Expiration = time.Now().Add(ConfigJSON.AnnotationDashboard.DashboardsCacheExpiration * time.Hour)
}

// ValidateURL - function for constructing and validating AM URL
func ValidateURL(baseURL, apiURL string) string {

	cmpltURL := baseURL + apiURL

	u, err := url.ParseRequestURI(cmpltURL)
	if err != nil {
		log.Fatalf("AlertManager API URL is not valid, error:%v", err)
	}

	return u.String()
}

// ParseConfig - Function for parsing the config File
func ParseConfig(configFile string, verbose int) {

	//Defaults in case no config file is provided
	ConfigJSON.Server.CMSMONURL = "https://cms-monitoring.cern.ch"
	ConfigJSON.Server.GetAlertsAPI = "/api/v1/alerts?active=true&silenced=false&inhibited=false&unprocessed=false"
	ConfigJSON.Server.GetSuppressedAlertsAPI = "/api/v1/alerts?active=false&silenced=true"
	ConfigJSON.Server.PostAlertsAPI = "/api/v1/alerts"
	ConfigJSON.Server.GetSilencesAPI = "/api/v1/silences"
	ConfigJSON.Server.PostSilenceAPI = "/api/v1/silences"
	ConfigJSON.Server.DeleteSilenceAPI = "/api/v1/silence"
	ConfigJSON.Server.HTTPTimeout = 3 // timeout in sec for HTTP requests
	ConfigJSON.Server.Interval = 10   // interval in sec for the service
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
	ConfigJSON.Alerts.DurationThreshold = 24 // duration threshold (in hours) used by filter pipeline
	ConfigJSON.Alerts.FilterKeywords = []string{}

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

	// Custom verbose value overriden
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

// findDashboard - helper function to find dashboard info
// The following block of code was taken from
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/monit.go#L604
func findDashboards(tag string) []models.AllDashboardsFetched {
	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", ConfigJSON.AnnotationDashboard.Token)
	headers = append(headers, []string{"Authorization", bearer})
	headers = append(headers, []string{"Accept", "application/json"})
	// example: /api/search?query=Production%20Overview&starred=true&tag=prod
	v := url.Values{}
	v.Set("tag", strings.Trim(tag, " "))
	apiURL := fmt.Sprintf("%s%s?%s", ConfigJSON.AnnotationDashboard.URL, ConfigJSON.AnnotationDashboard.DashboardSearchAPI, v.Encode())

	resp := HttpCall("GET", apiURL, headers, nil)
	defer resp.Body.Close()

	// Deserialize the response into a map.
	var data []models.AllDashboardsFetched
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		d, _ := io.ReadAll(resp.Body)
		log.Printf("Error parsing the response body: %s, %+v\n", err, string(d))
	}
	return data
}

// GetSilences - function for get request on /api/v1/silences alertmanager endpoint for fetching all silences.
func GetSilences() (models.AllSilences, error) {
	var data models.AllSilences
	apiURL := ValidateURL(ConfigJSON.Server.CMSMONURL, ConfigJSON.Server.GetSilencesAPI)

	var headers [][]string
	headers = append(headers, []string{"Accept-Encoding", "identify"})
	headers = append(headers, []string{"Accept", "application/json"})
	resp := HttpCall("GET", apiURL, headers, nil)
	defer resp.Body.Close()

	byteValue, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Unable to read JSON Data from AlertManager Silence GET API, error: %v\n", err)
		return data, err
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

// GetAlerts - function for get request on /api/v1/alerts alertmanager endpoint for fetching alerts.
func GetAlerts(getAlertsAPI string, updateMapChoice bool) (models.AmData, error) {

	var data models.AmData
	apiURL := ValidateURL(ConfigJSON.Server.CMSMONURL, getAlertsAPI)

	var headers [][]string
	headers = append(headers, []string{"Accept-Encoding", "identify"})
	headers = append(headers, []string{"Accept", "application/json"})
	resp := HttpCall("GET", apiURL, headers, nil)
	defer resp.Body.Close()

	byteValue, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Unable to read JSON Data from AlertManager GET API, error: %v\n", err)
		return data, err
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
		DataReadWriteLock.Lock()
		defer DataReadWriteLock.Unlock()
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

// PostAlert - function for making post request on /api/v1/alerts alertmanager endpoint for creating alerts.
func PostAlert(data models.AmJSON) error {
	apiURL := ValidateURL(ConfigJSON.Server.CMSMONURL, ConfigJSON.Server.PostAlertsAPI)
	var finalData []models.AmJSON
	finalData = append(finalData, data)

	jsonStr, err := json.Marshal(finalData)
	if err != nil {
		log.Printf("Unable to convert JSON Data, error: %v\n", err)
		return err
	}
	var headers [][]string
	headers = append(headers, []string{"Content-Type", "application/json"})
	resp := HttpCall("POST", apiURL, headers, bytes.NewBuffer(jsonStr))
	defer resp.Body.Close()

	if ConfigJSON.Server.Verbose > 1 {
		log.Println("Pushed Alerts: ", string(jsonStr))
	}
	return nil
}

// Get helper function to safely get value from the dict
func Get(dict map[string]interface{}, key string) (string, bool) {
	DataReadWriteLock.RLock()
	defer DataReadWriteLock.RUnlock()
	val, ok := dict[key]
	if ok {
		return val.(string), ok
	}
	return "", false
}

// Set helper function to safely set value in dict
func Set(dict map[string]interface{}, key string, value string) {
	DataReadWriteLock.Lock()
	defer DataReadWriteLock.Unlock()
	dict[key] = value
}

// HttpCall helper function to make http call
func HttpCall(method, apiURL string, headers [][]string, buf *bytes.Buffer) *http.Response {
	var req *http.Request
	var err error
	if buf != nil {
		// POST request
		req, err = http.NewRequest(method, apiURL, buf)
	} else {
		// GET, DELETE requests
		req, err = http.NewRequest(method, apiURL, nil)
	}
	if err != nil {
		log.Printf("Unable to make request to %s, error: %s", apiURL, err)
	}
	for _, v := range headers {
		if len(v) == 2 {
			req.Header.Set(v[0], v[1])
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
	if ConfigJSON.Server.Verbose > 2 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("response: ", string(dump))
		}
	}
	log.Println(method, apiURL, resp.Status)
	return resp
}
