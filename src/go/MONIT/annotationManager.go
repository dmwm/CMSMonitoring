package main

// File       : alert.go
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Tue, 1 Sep 2020 14:08:00 GMT
// Description: CERN MONIT infrastructure Alert CLI Tool

import (
	"encoding/json"
	"errors"
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

//-------VARIABLES-------

//MAX timeStamp //Saturday, May 24, 3000 3:43:26 PM
var maxtstmp int64 = 32516091806

var microSec int64 = 1000000

var milliSec int64 = 1000

//token name
var token string

//tags list seperated by comma
var tags string

//Action to be performed, eg. delete, update etc.
var action string

//time range
var trange string

//boolean for generating default config
var generateConfig *bool

//Config filepath
var configFilePath string

//-------VARIABLES-------

//-------STRUCTS---------

//Grafana Dashboard Annotation API Data struct
type annotationData struct {
	ID          int      `json:"id"`          //Annotation Id
	DashboardID int      `json:"dashboardId"` //Dashboard ID
	Created     int64    `json:"created"`     //Timestamp when the annotation created
	Updated     int64    `json:"updated"`     //Timestamp when the annotation last updated
	Time        int64    `json:"time"`        //Start Time of the annotation
	TimeEnd     int64    `json:"timeEnd"`     //End Time of the annotation
	Text        string   `json:"text"`        //Text field of the annotation
	Tags        []string `json:"tags"`        //List of tags attached with the annotation
}

type config struct {
	GrafanaBaseURL      string   `json:"grafanaBaseURL"`      // Grafana Annotation Base URL
	GetAnnotationsAPI   string   `json:"getAnnotationsAPI"`   //API endpoint for fetching all annotations
	DeleteAnnotationAPI string   `json:"deleteAnnotationAPI"` //API endpoint for deleting an annotation
	Columns             []string `json:"columns"`             // column names for annotations info
	HTTPTimeout         int      `json:"httpTimeout"`         // http timeout to connect to AM
	Verbose             int      `json:"verbose"`             // verbosity level
	Token               string   `json:"token"`               // CERN SSO token to use
}

//-------STRUCTS---------

var configJSON config

//function for get request grafana annotations endpoint for fetching annotations.
func getAnnotations(data *[]annotationData, tags []string) error {

	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", configJSON.Token)
	h := []string{"Authorization", bearer}
	headers = append(headers, h)
	h = []string{"Accept", "application/json"}
	headers = append(headers, h)
	// Ex: /api/annotations?from=1598366184000&to=1598366184000&tags=cmsweb&tags=cmsmon-int
	v := url.Values{}

	if trange != "" {
		tms := parseTimes()

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

	apiURL := fmt.Sprintf("%s%s?%s", configJSON.GrafanaBaseURL, configJSON.GetAnnotationsAPI, v.Encode())

	if configJSON.Verbose > 0 {
		log.Println(apiURL)
	}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("Unable to make request to %s, error: %s", apiURL, err)
		return err
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
		return err
	}
	defer resp.Body.Close()

	byteValue, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Unable to read JSON Data from Grafana Annotation GET API, error: %v\n", err)
		return err
	}

	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		if configJSON.Verbose > 0 {
			log.Println(string(byteValue))
		}
		log.Fatalf("Unable to parse JSON Data from Grafana Annotation GET API, error: %v\n", err)
		return err
	}

	if configJSON.Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}

	return nil
}

//function for delete request on grafana annotations endpoint for deleting an annotation.
func deleteAnnotation(aData annotationData) error {

	apiurl := fmt.Sprintf("%s%s%d", configJSON.GrafanaBaseURL, configJSON.DeleteAnnotationAPI, aData.ID)

	req, err := http.NewRequest("DELETE", apiurl, nil)
	if err != nil {
		log.Printf("Request Error, error: %v\n", err)
		return err
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
		log.Printf("Response Error, error: %v\n", err)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Http Response Code Error, status code: %d", resp.StatusCode)
		return errors.New("Http Response Code Error")
	}

	defer resp.Body.Close()

	if configJSON.Verbose > 2 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}

	if configJSON.Verbose > 2 {
		log.Printf("Annotation Deleted for:\n%+v\n", aData)
	}

	return nil
}

//
// The following block of code was taken from
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/alert.go#L204

//Helper function for converting time difference in a meaningful manner
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
//Helper function for time difference between two time.Time objects
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
//Function for time difference between two time.Time objects
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
//Function for printing annotations in Plain text format
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

//Function running all logics
func run() {

	var aData []annotationData
	tagList := parseTags(tags)
	err := getAnnotations(&aData, tagList)
	if err != nil {
		log.Fatalf("Unable to fetch annotations, error :%v", err)
	}
	tabulate(aData)

	switch action {
	case "delete":
		fmt.Println("DELETING ALL OF THE ALERTS SHOWN ABOVE!!")
		for _, each := range aData {
			deleteAnnotation(each)
			if err != nil {
				if configJSON.Verbose > 1 {
					log.Printf("Annotation Data: %v", each)
				}
				log.Fatalf("Unable to delete the annotation, error :%v", err)
			}
		}
	default:
		break
	}

}

//helper function for parsing Configs
func openConfigFile(configFilePath string) {
	jsonFile, e := os.Open(configFilePath)
	if e != nil {
		if configJSON.Verbose > 0 {
			log.Printf("Config File not found at %s, error: %s\n", configFilePath, e)
		} else {
			fmt.Printf("Config File Missing at %s. Using Defaults\n", configFilePath)
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

//function for parsing Configs
func parseConfig(verbose int) {

	configFilePath = os.Getenv("CONFIG_PATH") //CONFIG_PATH Environment Variable storing config filepath.
	defaultConfigFilePath := os.Getenv("HOME") + "/.annotationManagerConfig.json"

	//Defaults in case no config file is provided
	configJSON.GrafanaBaseURL = "https://monit-grafana.cern.ch"
	configJSON.GetAnnotationsAPI = "/api/annotations/"
	configJSON.DeleteAnnotationAPI = "/api/annotations/"
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
		fmt.Printf("A new configuration file %s was generated.\n", filePath)
		return
	}

	if configFilePath != "" {
		openConfigFile(configFilePath)
	} else if defaultConfigFilePath != "" {
		fmt.Printf("$CONFIG_PATH is not set. Using config file at %s\n", defaultConfigFilePath)
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
		configJSON.Token = token
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
func parseTimes() []int64 {
	var times []int64

	//additional line added in the adopted code for date support.
	if strings.Contains(trange, "--") {
		tms := strings.Split(trange, "--")

		startTime, err := time.Parse(time.RFC3339, tms[0])
		if err != nil {
			log.Printf("Unable to parse time, error: %s", err)
		}

		endTime, err := time.Parse(time.RFC3339, tms[1])
		if err != nil {
			log.Printf("Unable to parse time, error: %s", err)
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
				t = time.Now().UTC().Add(time.Duration((int64(v) * offset)) * -1 * time.Second).Unix()
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

	flag.StringVar(&token, "token", "", "Authentication token to use")
	flag.StringVar(&tags, "tags", "", "List of tags seperated by comma")
	flag.StringVar(&action, "action", "", "Action to be performed, eg. delete, update etc.")
	flag.StringVar(&trange, "trange", "", "Time Range for filtering annotations.")
	generateConfig = flag.Bool("generateConfig", false, "Flag for generating default config")
	var verbose int
	flag.IntVar(&verbose, "verbose", 0, "verbosity level, can be overwritten in config")

	flag.Usage = func() {
		configPath := os.Getenv("HOME") + "/.annotationManagerConfig.json"
		fmt.Println("Usage: annotationManager [options]")
		flag.PrintDefaults()
		fmt.Println("\nEnvironments:")
		fmt.Printf("\tCONFIG_PATH:\t Config to use, default (%s)\n", configPath)
		fmt.Println("\nExamples:")
		fmt.Println("\tGet all annotations with specified tags:")
		fmt.Println("\t    annotationManager -tags=das,cmsmon-int")
		fmt.Println("\n\tUse (d/day/days) for days, (h/hr/hrs) for hours, (m/min/mins) for minutes and (s/sec/secs) for seconds.")
		fmt.Println("\n\tGet all annotations with specified tags on time period range (from last 7days to now)")
		fmt.Println("\t    annotationManager -tags=das,cmsmon-int -trange=7d-now")
		fmt.Println("\n\tGet all annotations with specified tags on time period range (from last 14days to last 7days)")
		fmt.Println("\t    annotationManager -tags=das,cmsmon-int -trange=14d-7d")
		fmt.Println("\n\tGet all annotations with specified tags on time period range (using dates seperated by '--')")
		fmt.Println("\t    annotationManager -tags=das,cmsmon-int -trange=2020-09-01T19:49:50.206Z--2020-09-01T19:49:50.206Z")
		fmt.Println("\n\tDelete annotations once you find appropriate annotations to delete from the above commands.")
		fmt.Println("\t    annotationManager -tags=das,cmsmon-int -trange=7d-now -action=delete")
		fmt.Println("\t    annotationManager -tags=das,cmsmon-int -trange=14d-7d -action=delete")
		fmt.Println("\t    annotationManager -tags=das,cmsmon-int -trange=2020-09-01T19:49:50.206Z--2020-09-01T19:49:50.206Z -action=delete")
	}

	flag.Parse()
	parseConfig(verbose)
	if !*generateConfig {
		run()
	}
}
