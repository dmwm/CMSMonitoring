package main

// File       : annotationManager.go
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Tue, 1 Sep 2020 14:08:00 GMT
// Description: CERN MONIT infrastructure Annotation Manager CLI Tool

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

// -------VARIABLES-------

// MAX timeStamp // Saturday, May 24, 3000 3:43:26 PM
var maxtstmp int64 = 32516091806

var microSec int64 = 1000000

var milliSec int64 = 1000

// token name
var token string

// text for annotating
var annotation string

// annotation ID for deleting single annotation
var annotationID int

// limit for fetching number of annotations at a time
var grafanaLimit int

// tags list seperated by comma
var tags string

// Action to be performed. [list, create, delete, deleteall, update] Default: list
var action string

// time range
var trange string

// boolean for generating default config
var generateConfig *bool

// Config filepath
var configFilePath string

// config variable
var configJSON config

// -------VARIABLES-------

// -------STRUCTS---------

// Grafana Dashboard Annotation API Data struct
type annotationData struct {
	ID          int      `json:"id"`          // Annotation Id
	DashboardID int      `json:"dashboardId"` // Dashboard ID
	Created     int64    `json:"created"`     // Timestamp when the annotation created
	Updated     int64    `json:"updated"`     // Timestamp when the annotation last updated
	Time        int64    `json:"time"`        // Start Time of the annotation
	TimeEnd     int64    `json:"timeEnd"`     // End Time of the annotation
	Text        string   `json:"text"`        // Text field of the annotation
	Tags        []string `json:"tags"`        // List of tags attached with the annotation
}

type config struct {
	GrafanaBaseURL      string   `json:"grafanaBaseURL"`      // Grafana Annotation Base URL
	FindDashboardAPI    string   `json:"findDashboardAPI"`    // API endpoint for all dashboards with given tags
	ListAnnotationsAPI  string   `json:"listAnnotationsAPI"`  // API endpoint for fetching all annotations
	CreateAnnotationAPI string   `json:"createAnnotationAPI"` // API endpoint for creating an annotation
	UpdateAnnotationAPI string   `json:"updateAnnotationAPI"` // API endpoint for updating an annotation
	DeleteAnnotationAPI string   `json:"deleteAnnotationAPI"` // API endpoint for deleting an annotation
	Columns             []string `json:"columns"`             // column names for annotations info
	HTTPTimeout         int      `json:"httpTimeout"`         // http timeout to connect to AM
	Verbose             int      `json:"verbose"`             // verbosity level
	Token               string   `json:"token"`               // CERN SSO token to use
}

// -------STRUCTS---------

// function for get request grafana annotations endpoint for fetching annotations.
func getAnnotations(data *[]annotationData, tags []string) {

	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", configJSON.Token)
	h := []string{"Authorization", bearer}
	headers = append(headers, h)
	h = []string{"Accept", "application/json"}
	headers = append(headers, h)
	// Ex: /api/annotations?from=1598366184000&to=1598366184000&tags=cmsweb&tags=cmsmon-int
	v := url.Values{}

	if trange != "" {
		tms := parseTimes(false) // False because Listing annotations with past time offset

		if len(tms) == 2 {
			v.Add("from", strings.Trim(strconv.FormatInt(tms[0], 10), " "))
			v.Add("to", strings.Trim(strconv.FormatInt(tms[1], 10), " "))
		} else if len(tms) == 1 {
			v.Add("from", strings.Trim(strconv.FormatInt(tms[0], 10), " "))
			v.Add("to", strings.Trim(strconv.FormatInt(time.Now().UTC().Unix()*int64(milliSec), 10), " "))
		}
	}

	for _, tag := range tags {
		v.Add("tags", strings.Trim(tag, " "))
	}

	v.Add("limit", fmt.Sprintf("%d", grafanaLimit))

	apiURL := fmt.Sprintf("%s%s?%s", configJSON.GrafanaBaseURL, configJSON.ListAnnotationsAPI, v.Encode())

	if configJSON.Verbose > 0 {
		log.Println(apiURL)
	}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Fatalf("Unable to make request to %s, error: %s", apiURL, err)
	}
	for _, v := range headers {
		if len(v) == 2 {
			req.Header.Add(v[0], v[1])
		}
	}

	timeout := time.Duration(configJSON.HTTPTimeout) * time.Second
	client := &http.Client{Timeout: timeout}

	if configJSON.Verbose > 1 {
		log.Println("URL", apiURL)
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("Request: ", string(dump))
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Response Error, error: %v\n", err)
	}
	defer resp.Body.Close()

	byteValue, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Fatalf("Unable to read JSON Data from Grafana Annotation GET API, error: %v\n", err)
	}

	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		if configJSON.Verbose > 0 {
			log.Println(string(byteValue))
		}
		log.Fatalf("Unable to parse JSON Data from Grafana Annotation GET API, error: %v\n", err)
	}

	if configJSON.Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}
}

// function for delete request on grafana annotations endpoint for deleting an annotation.
func deleteAnnotationHelper(annotationID int) {

	apiurl := fmt.Sprintf("%s%s%d", configJSON.GrafanaBaseURL, configJSON.DeleteAnnotationAPI, annotationID)

	req, err := http.NewRequest("DELETE", apiurl, nil)
	if err != nil {
		log.Fatalf("Request Error, error: %v\n", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", configJSON.Token))

	timeout := time.Duration(configJSON.HTTPTimeout) * time.Second
	client := &http.Client{Timeout: timeout}

	if configJSON.Verbose > 1 {
		log.Println("DELETE", apiurl)
	} else if configJSON.Verbose > 2 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("Request: ", string(dump))
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Response Error, error: %v\n", err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Http Response Code Error, status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	if configJSON.Verbose > 2 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}
}

//
// The following block of code was taken from
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/alert.go#L204

// Helper function for converting time difference in a meaningful manner
func diff(a, b time.Time) (array []int) {
	if a.Location() != b.Location() {
		b = b.In(a.Location())
	}
	if a.After(b) {
		a, b = b, a
	}
	y1, M1, d1 := a.Date()
	y2, M2, d2 := b.Date()

	h1, m1, s1 := a.Clock()
	h2, m2, s2 := b.Clock()

	var year = int(y2 - y1)
	var month = int(M2 - M1)
	var day = int(d2 - d1)
	var hour = int(h2 - h1)
	var min = int(m2 - m1)
	var sec = int(s2 - s1)

	// Normalize negative values
	if sec < 0 {
		sec += 60
		min--
	}
	if min < 0 {
		min += 60
		hour--
	}
	if hour < 0 {
		hour += 24
		day--
	}
	if day < 0 {
		// days in month:
		t := time.Date(y1, M1, 32, 0, 0, 0, 0, time.UTC)
		day += 32 - t.Day()
		month--
	}
	if month < 0 {
		month += 12
		year--
	}

	array = append(array, year)
	array = append(array, month)
	array = append(array, day)
	array = append(array, hour)
	array = append(array, min)
	array = append(array, sec)

	return
}

//
// The following block of code was taken from
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/alert.go#L259
// Helper function for time difference between two time.Time objects
func timeDiffHelper(timeList []int) (dif string) {
	for ind := range timeList {
		if timeList[ind] > 0 {
			switch ind {
			case 0:
				dif += strconv.Itoa(timeList[ind]) + "Y "
				break
			case 1:
				dif += strconv.Itoa(timeList[ind]) + "M "
				break
			case 2:
				dif += strconv.Itoa(timeList[ind]) + "D "
				break
			case 3:
				dif += strconv.Itoa(timeList[ind]) + "h "
				break
			case 4:
				dif += strconv.Itoa(timeList[ind]) + "m "
				break
			case 5:
				dif += strconv.Itoa(timeList[ind]) + "s "
				break
			default:
				break
			}
		}
	}

	return
}

//
// The following block of code was taken from
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/alert.go#L291
// Function for time difference between two time.Time objects
func timeDiff(t1 time.Time, t2 time.Time, duration int) string {
	if t1.After(t2) {
		timeList := diff(t1, t2)
		return timeDiffHelper(timeList) + "AGO"
	}

	timeList := diff(t2, t1)
	if duration == 1 {
		return timeDiffHelper(timeList)
	}
	return "IN " + timeDiffHelper(timeList)

}

//
// The following block of code was taken from (with few changes done)
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/alert.go#L426
// Function for printing annotations in Plain text format
func tabulate(data []annotationData) {

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 0, '\t', 0)
	defer w.Flush()

	fmt.Fprintf(w, "\n ")
	for _, each := range configJSON.Columns {
		fmt.Fprintf(w, "%s\t", each)
	}
	fmt.Fprintf(w, "\n")

	for _, each := range data {

		fmt.Fprintf(w, " %d\t%d\t%s\t%s\t%s",
			each.ID,
			each.DashboardID,
			strings.Split(each.Text, "\n<a")[0],
			each.Tags,
			timeDiff(time.Now(), time.Unix(0, each.Time*int64(microSec)), 0), // times should be in microseconds of Unix since epoch
		)
		if each.TimeEnd == maxtstmp {
			fmt.Fprintf(w, "\t%s", "Undefined")
			fmt.Fprintf(w, "\t%s\n", "Undefined")
		} else {
			fmt.Fprintf(w, "\t%s", timeDiff(time.Now(), time.Unix(0, each.TimeEnd*int64(microSec)), 0))                                // times should be in microseconds of Unix since epoch
			fmt.Fprintf(w, "\t%s\n", timeDiff(time.Unix(0, each.Time*int64(microSec)), time.Unix(0, each.TimeEnd*int64(microSec)), 1)) // times should be in microseconds of Unix since epoch
		}
	}
}

// helper function for listing annotations
func listAnnotation() []annotationData {
	var aData []annotationData
	tagList := parseTags(tags)
	getAnnotations(&aData, tagList)

	if len(aData) == 0 {
		log.Printf("NO ANNOTATION FOUND\n")
		return nil
	}

	tabulate(aData)
	return aData
}

//
// The following block of code was taken from (with few changes done)
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/monit.go#L603

// helper function to find dashboard info
func findDashboard() []map[string]interface{} {
	tagsList := parseTags(tags)
	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", configJSON.Token)
	h := []string{"Authorization", bearer}
	headers = append(headers, h)
	h = []string{"Accept", "application/json"}
	headers = append(headers, h)
	// example: /api/search?query=Production%20Overview&starred=true&tag=prod
	v := url.Values{}
	for _, tag := range tagsList {
		v.Add("tag", strings.Trim(tag, " "))
	}
	rurl := fmt.Sprintf("%s%s?%s", configJSON.GrafanaBaseURL, configJSON.FindDashboardAPI, v.Encode())
	if configJSON.Verbose > 0 {
		log.Println(rurl)
	}

	req, err := http.NewRequest("GET", rurl, nil)
	if err != nil {
		log.Fatalf("Unable to make request to %s, error: %s", rurl, err)
	}
	for _, v := range headers {
		if len(v) == 2 {
			req.Header.Add(v[0], v[1])
		}
	}
	if configJSON.Verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("request: ", string(dump))
		}
	}
	timeout := time.Duration(configJSON.HTTPTimeout) * time.Second
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Unable to get response from %s, error: %s", rurl, err)
	}
	if configJSON.Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("response: ", string(dump))
		}
	}
	defer resp.Body.Close()
	var data []map[string]interface{}
	// Deserialize the response into a map.
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	return data

}

//
// The following block of code was taken from (with few changes done)
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/monit.go#L657

// helper function to add annotation
func addAnnotation(data []byte) {
	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", configJSON.Token)
	h := []string{"Authorization", bearer}
	headers = append(headers, h)
	h = []string{"Content-Type", "application/json"}
	headers = append(headers, h)

	rurl := fmt.Sprintf("%s%s", configJSON.GrafanaBaseURL, configJSON.CreateAnnotationAPI)
	if configJSON.Verbose > 0 {
		log.Println(rurl)
	}
	req, err := http.NewRequest("POST", rurl, bytes.NewBuffer(data))
	if err != nil {
		log.Fatalf("Unable to make request to %s, error: %s", rurl, err)
	}
	for _, v := range headers {
		if len(v) == 2 {
			req.Header.Add(v[0], v[1])
		}
	}
	if configJSON.Verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("request: ", string(dump))
		}
	}
	timeout := time.Duration(configJSON.HTTPTimeout) * time.Second
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Unable to get response from %s, error: %s", rurl, err)
	}
	if configJSON.Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("response:", string(dump))
		}
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Unable to read JSON Data from Grafana Annotation POST API, error: %v\n", err)
	}

	if configJSON.Verbose > 1 {
		log.Println("response Status:", resp.Status)
		log.Println("response Headers:", resp.Header)
		log.Println("response Body:", string(body))
	}
}

//
// The following block of code was taken from (with few changes done)
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/monit.go#L813
func createAnnotation() {

	if annotation != "" {
		if tags != "" {
			dashboards := findDashboard()
			timeRanges := parseTimes(true) // True because we are creating annotation and time ranges should be in future
			if trange == "" {
				log.Fatalf("Unable to create Annotation. Please provide Time Range!\n")
			}
			if configJSON.Verbose > 0 {
				log.Println("timeRanges", timeRanges)
			}
			for _, r := range dashboards {
				rec := make(map[string]interface{})
				rec["dashboardId"] = r["id"]
				rec["time"] = timeRanges[0]
				rec["timeEnd"] = timeRanges[1]
				rec["tags"] = parseTags(tags)
				rec["text"] = annotation
				data, err := json.Marshal(rec)
				if err != nil {
					log.Fatalf("Unable to marshal the data %+v, error %v\n", rec, err)
				}
				if configJSON.Verbose > 0 {
					log.Printf("Add annotation: %+v", rec)
				}
				addAnnotation(data)
			}

			log.Printf("Annotation Created Successfully\n")

		} else {
			log.Fatalf("Can't Create annotation. Please provide tags!\n")
		}
	} else {
		log.Fatalf("Can't Create annotation without any text!\n")
	}
}

// helper function for updating an annotation which makes PUT request
func updateAnnotationHelper(annotationID int, data []byte) {

	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", configJSON.Token)
	h := []string{"Authorization", bearer}
	headers = append(headers, h)
	h = []string{"Content-Type", "application/json"}
	headers = append(headers, h)
	h = []string{"Accept", "application/json"}
	headers = append(headers, h)

	apiurl := fmt.Sprintf("%s%s%d", configJSON.GrafanaBaseURL, configJSON.DeleteAnnotationAPI, annotationID)
	if configJSON.Verbose > 0 {
		log.Println(apiurl)
	}
	req, err := http.NewRequest("PUT", apiurl, bytes.NewBuffer(data))
	if err != nil {
		log.Fatalf("Unable to make request to %s, error: %s", apiurl, err)
	}
	for _, v := range headers {
		if len(v) == 2 {
			req.Header.Add(v[0], v[1])
		}
	}
	if configJSON.Verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("request: ", string(dump))
		}
	}
	timeout := time.Duration(configJSON.HTTPTimeout) * time.Second
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Unable to get response from %s, error: %s", apiurl, err)
	}
	if configJSON.Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("response:", string(dump))
		}
	}

	if resp.StatusCode != 200 {
		log.Fatalf("HTTP Respose error, code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Unable to read JSON Data from Grafana Annotation PUT API, error: %v\n", err)
	}

	if configJSON.Verbose > 1 {
		log.Println("response Status:", resp.Status)
		log.Println("response Headers:", resp.Header)
		log.Println("response Body:", string(body))
	}
}

// function which contains the logic of updating an annotation
func updateAnnotation() {
	updatedData := make(map[string]interface{})

	if trange != "" {
		tms := parseTimes(true)
		if len(tms) == 2 {
			updatedData["time"] = tms[0]
			updatedData["timeEnd"] = tms[1]
		}
	}

	if annotation != "" {
		updatedData["text"] = annotation
	}

	if tags != "" {
		updatedData["tags"] = parseTags(tags)
	}

	data, err := json.Marshal(updatedData)
	if err != nil {
		log.Fatalf("Unable to parse Data for Updation, update failed !, error: %v", err)
	}

	updateAnnotationHelper(annotationID, data)

	log.Printf("Annotation with id:%d has been updated successfully!\n", annotationID)

	if configJSON.Verbose > 1 {
		log.Printf("Annotation Update Data :%v\n", string(data))
	}
}

// function which contains the logic of deleting an annotation
func deleteOneAnnotation() {
	deleteAnnotationHelper(annotationID)
	log.Printf("Annotation with id: %d has been deleted successfully !\n", annotationID)
}

// function which contains the logic of deleting multiple annotations
func deleteAllAnnotations() {
	noOfDeletedAnnotation := 0
	aData := listAnnotation()
	if aData != nil {
		log.Printf("DELETING ALL OF THE ALERTS SHOWN ABOVE!!\n")
		for _, each := range aData {
			deleteAnnotationHelper(each.ID)
			if configJSON.Verbose > 2 {
				log.Printf("Annotation Deleted for:\n%+v\n", each)
			}
			noOfDeletedAnnotation++
		}
		log.Printf("Successfully Deleted %d annotations!!\n", noOfDeletedAnnotation)
	} else {
		log.Fatalf("Unable to delete with null data!\n")
	}
}

// Function running all logics
func run() {

	switch action {
	case "list":
		listAnnotation()

	case "create":
		createAnnotation()

	case "update":
		updateAnnotation()

	case "delete":
		deleteOneAnnotation()

	case "deleteall":
		deleteAllAnnotations()

	default:
		listAnnotation()
	}
}

// helper function for parsing Configs
func openConfigFile(configFilePath string) {
	jsonFile, e := os.Open(configFilePath)
	if e != nil {
		if configJSON.Verbose > 0 {
			log.Printf("Config File not found at %s, error: %s\n", configFilePath, e)
		} else {
			log.Printf("Config File Missing at %s. Using Defaults\n", configFilePath)
		}
		return
	}
	defer jsonFile.Close()
	decoder := json.NewDecoder(jsonFile)
	err := decoder.Decode(&configJSON)
	if err != nil {
		log.Printf("Config JSON File can't be loaded, error: %s", err)
		return
	} else if configJSON.Verbose > 0 {
		log.Printf("Load config from %s\n", configFilePath)
	}
}

// function for parsing Configs
func parseConfig(verbose int) {

	configFilePath = os.Getenv("CONFIG_PATH") // CONFIG_PATH Environment Variable storing config filepath.
	defaultConfigFilePath := os.Getenv("HOME") + "/.annotationManagerConfig.json"

	// Defaults in case no config file is provided
	configJSON.GrafanaBaseURL = "https://monit-grafana.cern.ch"
	configJSON.ListAnnotationsAPI = "/api/annotations/"
	configJSON.CreateAnnotationAPI = "/api/annotations/"
	configJSON.UpdateAnnotationAPI = "/api/annotations/"
	configJSON.DeleteAnnotationAPI = "/api/annotations/"
	configJSON.FindDashboardAPI = "/api/search"
	configJSON.Columns = []string{"ID", "DASID", "TEXT", "TAGS", "STARTS", "ENDS", "DURATION"}
	configJSON.Verbose = verbose
	configJSON.HTTPTimeout = 5 // 5 seconds timeout for http

	if *generateConfig {
		config, err := json.MarshalIndent(configJSON, "", " ")
		if err != nil {
			log.Fatalf("Default Config Value can't be parsed from configJSON struct, error: %s", err)
		}
		filePath := defaultConfigFilePath
		if len(flag.Args()) > 0 {
			filePath = flag.Args()[0]
		}

		err = ioutil.WriteFile(filePath, config, 0644)
		if err != nil {
			log.Fatalf("Failed to generate Config File, error: %s", err)
		}
		log.Printf("A new configuration file %s was generated.\n", filePath)
		return
	}

	if configFilePath != "" {
		openConfigFile(configFilePath)
	} else if defaultConfigFilePath != "" {
		log.Printf("$CONFIG_PATH is not set. Using config file at %s\n", defaultConfigFilePath)
		openConfigFile(defaultConfigFilePath)
	}

	// we we were given verbose from command line we should overwrite its value in config
	if verbose > 0 {
		configJSON.Verbose = verbose
	}

	if configJSON.Verbose > 0 {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}

	// if we were given the token we will use it
	if token != "" {

		if _, err := os.Stat(token); err == nil {
			tokenData, ioErr := ioutil.ReadFile(token)
			if ioErr != nil {
				log.Fatalf("Unable to read token file: %s, error: %v\n", token, ioErr)
			}
			configJSON.Token = string(tokenData)
		} else {
			configJSON.Token = token
		}
	}

	if configJSON.Verbose > 1 {
		log.Printf("Configuration:\n%+v\n", configJSON)
	}
}

//
// The following block of code was taken from
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/monit.go#L550

// helper function to parse comma separated tags string
func parseTags(itags string) []string {
	var tags []string
	for _, tag := range strings.Split(itags, ",") {
		tags = append(tags, strings.Trim(tag, " "))
	}
	return tags
}

//
// The following block of code was taken from (with few changes done)
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/monit.go#L559

// helper function to parse given time-range separated by '-'
func parseTimes(ifCreatingAnnotation bool) []int64 {
	var times []int64

	// additional line added in the adopted code for date support.
	if strings.Contains(trange, "--") {
		tms := strings.Split(trange, "--")

		startTime, err := time.Parse(time.RFC3339, tms[0])
		if err != nil {
			log.Fatalf("Unable to parse time, error: %s", err)
		}

		endTime, err := time.Parse(time.RFC3339, tms[1])
		if err != nil {
			log.Fatalf("Unable to parse time, error: %s", err)
		}

		times = append(times, startTime.Unix()*int64(milliSec))
		times = append(times, endTime.Unix()*int64(milliSec)) // times should be in milliseconds of Unix since epoch
		return times
	}

	for _, v := range strings.Split(trange, "-") {
		v = strings.Trim(v, " ")
		var t int64
		if v == "now" {
			t = time.Now().UTC().Unix()
		} else if len(v) == 10 { // unix since epoch
			value, err := strconv.Atoi(v)
			if err != nil {
				log.Fatalf("Unable to parse given time value: %v\n", v)
			}
			t = int64(value)
		} else {
			var value string
			var offset int64
			if strings.HasSuffix(v, "s") || strings.HasSuffix(v, "sec") || strings.HasSuffix(v, "secs") {
				value = strings.Split(v, "s")[0]
				offset = 1
			} else if strings.HasSuffix(v, "m") || strings.HasSuffix(v, "min") || strings.HasSuffix(v, "mins") {
				value = strings.Split(v, "m")[0]
				offset = 60
			} else if strings.HasSuffix(v, "h") || strings.HasSuffix(v, "hr") || strings.HasSuffix(v, "hrs") {
				value = strings.Split(v, "h")[0]
				offset = 60 * 60
			} else if strings.HasSuffix(v, "d") || strings.HasSuffix(v, "day") || strings.HasSuffix(v, "days") {
				value = strings.Split(v, "d")[0]
				offset = 60 * 60 * 24
			} else {
				log.Fatalf("Unable to parse given time value: %v\n", v)
			}
			if v, e := strconv.Atoi(value); e == nil {
				if ifCreatingAnnotation { // When we create annotation we think in future time frame while when we list annotation the offset would be in past.
					t = time.Now().UTC().Add(time.Duration((int64(v) * offset)) * time.Second).Unix() // FUTURE OFFSET
				} else {
					t = time.Now().UTC().Add(time.Duration((int64(v) * offset)) * -1 * time.Second).Unix() // PAST OFFSET
				}
			}
		}
		if t == 0 {
			log.Fatalf("Unable to parse given time value: %v\n", v)
		}
		times = append(times, t*milliSec) // times should be in milliseconds of Unix since epoch
	}
	return times
}

func main() {

	flag.StringVar(&annotation, "annotation", "", "Annotation text")
	flag.IntVar(&grafanaLimit, "grafana-limit", 100, "Limit for fetching number of annotations at a time.")
	flag.StringVar(&token, "token", "", "Authentication token to use (Optional-can be stored in config file)")
	flag.StringVar(&tags, "tags", "", "List of tags seperated by comma")
	flag.StringVar(&action, "action", "", "Action to be performed. [list, create, delete, deleteall, update] Default: list")
	flag.StringVar(&trange, "trange", "", "Time Range for filtering annotations.")
	flag.IntVar(&annotationID, "annotationID", 0, "Annotation ID required in making delete api call for deleting an annotation")
	generateConfig = flag.Bool("generateConfig", false, "Flag for generating default config")
	var verbose int
	flag.IntVar(&verbose, "verbose", 0, "verbosity level, can be overwritten in config (Optional-can be stored in config file)")

	flag.Usage = func() {
		configPath := os.Getenv("HOME") + "/.annotationManagerConfig.json"
		fmt.Println("Usage: annotationManager [options]")
		flag.PrintDefaults()
		fmt.Println("\nEnvironments:")
		fmt.Printf("\tCONFIG_PATH:\t Config to use, default (%s)\n", configPath)
		fmt.Println("\nExamples:")
		fmt.Println("\tALTHOUGH ACTION SHOULD BE (-action=list) FOR GETTING ALL ANNOTATIONS BUT ACTION FLAG CAN BE DROPPED AS LIST IS DEFAULT ACTION!")
		fmt.Println("\n\tGet all annotations with specified tags:")
		fmt.Println("\t    annotationManager -action=list -tags=das,cmsmon-int")
		fmt.Println("\n\tUse (d/day/days) for days, (h/hr/hrs) for hours, (m/min/mins) for minutes and (s/sec/secs) for seconds.")
		fmt.Println("\n\tGet all annotations with specified tags on time period range (from last 7days to now)")
		fmt.Println("\t    annotationManager -action=list -tags=das,cmsmon-int -trange=7d-now")
		fmt.Println("\n\tGet all annotations with specified tags on time period range (from last 14days to last 7days)")
		fmt.Println("\t    annotationManager -action=list -tags=das,cmsmon-int -trange=14d-7d")
		fmt.Println("\n\tGet all annotations with specified tags on time period range (using dates seperated by '--')")
		fmt.Println("\t    annotationManager -action=list -tags=das,cmsmon-int -trange=2020-09-01T19:49:50.206Z--2020-09-01T19:49:50.206Z")
		fmt.Println("\n\tDelete all annotations once you find appropriate annotations to delete from the above commands.")
		fmt.Println("\t    annotationManager -action=deleteall -tags=das,cmsmon-int -trange=7d-now")
		fmt.Println("\t    annotationManager -action=deleteall -tags=das,cmsmon-int -trange=14d-7d")
		fmt.Println("\t    annotationManager -action=deleteall -tags=das,cmsmon-int -trange=2020-09-01T19:49:50.206Z--2020-09-01T19:49:50.206Z")
		fmt.Println("\n\tDelete an annotation with specific annotation id (/api/annotations/:id)")
		fmt.Println("\t    annotationManager -action=delete -annotationID=761323")
		fmt.Println("\n\tCreate an annotation on dashboards with specific tags with time ranging from now to 2m")
		fmt.Println("\t    annotationManager -action=create -annotation=\"some message\" -tags=das,cmsweb -trange=now-2m")
		fmt.Println("\n\tUpdate an annotation with specific annotation id with new annotation and tags. (/api/annotations/:id)")
		fmt.Println("\t(TAGS AND ANNOTATION ARE REQUIRED FLAGS OR OLD DATA WILL BE REPLACED WITH NULL VALUES. If you don't want to change annotation or tag, please send the old values.)")
		fmt.Println("\t    annotationManager -action=update -annotationID=761323 -annotation=\"some new message\" -tags=newTag1,newTag2")
		fmt.Println("\n\tUpdate an annotation with specific annotation id with new start and end Time. (/api/annotations/:id)")
		fmt.Println("\t(TAGS AND ANNOTATION ARE REQUIRED FLAGS OR OLD DATA WILL BE REPLACED WITH NULL VALUES. If you don't want to change annotation or tag, please send the old values.)")
		fmt.Println("\t    annotationManager -action=update -annotationID=761323 -annotation=\"old message\" -tags=oldTag1,oldTag2 -trange=now-5m")
	}

	flag.Parse()
	parseConfig(verbose)
	if !*generateConfig {
		run()
	}
}
