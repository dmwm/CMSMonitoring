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
	"strconv"
	"strings"
	"time"
)

// File       : ggus_alerting.go
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Thu, 21 May 2020 21:07:32 GMT
// Description: GGUS Alerting Module for CERN MONIT infrastructure

//URLs for AlertManager Instances
var alertManagerURLs string

//List of URLs for AlertManager Instances
var alertManagerURLList []string

// severity of alerts
var severity string

//tag name
var tag string

//service name
var service string

//Required VO attribute
var vo string

//verbose defines verbosity level
var verbose int

//MAX timeStamp //Saturday, May 24, 3000 3:43:26 PM
var maxtstmp int64 = 32516091806

//Map for storing Existing Tickets
var exstTkt map[string]int

//GGUS Data Struct
type ggus []struct {
	TicketID        int    `json:"TicketID"`
	Type            string `json:"Type"`
	VO              string `json:"VO"`
	Site            string `json:"Site"`
	Priority        string `json:"Priority"`
	ResponsibleUnit string `json:"ResponsibleUnit"`
	Status          string `json:"Status"`
	LastUpdate      string `json:"LastUpdate"`
	Subject         string `json:"Subject"`
	Scope           string `json:"Scope"`
}

//AlertManager API acceptable JSON Data for GGUS Data
type amJSON struct {
	Labels struct {
		Alertname string `json:"alertname"`
		Severity  string `json:"severity"`
		Service   string `json:"service"`
		Tag       string `json:"tag"`
		Priority  string `json:"Priority"`
		Scope     string `json:"Scope"`
		Site      string `json:"Site"`
		VO        string `json:"VO"`
		Type      string `json:"Type"`
	} `json:"labels"`
	Annotations struct {
		Priority        string `json:"Priority"`
		ResponsibleUnit string `json:"ResponsibleUnit"`
		Scope           string `json:"Scope"`
		Site            string `json:"Site"`
		Status          string `json:"Status"`
		Subject         string `json:"Subject"`
		TicketID        string `json:"TicketID"`
		Type            string `json:"Type"`
		VO              string `json:"VO"`
		URL             string `json:"URL"`
	} `json:"annotations"`
	StartsAt time.Time `json:"startsAt"`
	EndsAt   time.Time `json:"endsAt"`
}

//AlertManager GET API acceptable JSON Data struct for GGUS data
type ggusData struct {
	Data []amJSON
}

//function for parsing JSON data from GGUS Data
func (data *ggus) parseJSON(jsondata []byte) {
	err := json.Unmarshal(jsondata, &data)
	if err != nil {
		log.Printf("Unable to parse GGUS JSON data, error: %v\n", err)
	}
}

//function for fetching JSON data from GGUS Data
func fetchJSON(filename string) []byte {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Unable to open JSON file, error: %v\n", err)
	}

	defer file.Close()

	jsonData, err := ioutil.ReadAll(file)
	if err != nil {
		log.Printf("Unable to read JSON file, error: %v\n", err)
	}

	if verbose > 1 {
		log.Println("GGUS Data: " + string(jsonData))
	}

	return jsonData

}

//function for converting SSB JSON Data into JSON Data required by AlertManager APIs.
func convertData(data ggus) []byte {

	var temp amJSON
	var finalData []amJSON

	exstTkt = make(map[string]int)

	for _, each := range data {

		// Ignoring all tickets whose VO is not "cms"
		if each.VO != vo {
			continue
		}

		exstTkt[strconv.Itoa(each.TicketID)] = 1

		begin, errInt := strconv.ParseInt(each.LastUpdate, 10, 64)
		if errInt != nil {
			log.Printf("Unable to convert LastUpdate string to int64, error: %v\n", errInt)
		}

		_beginRFC3339 := time.Unix(begin, 0).UTC()

		temp.Labels.Alertname = "ggus-" + strconv.Itoa(each.TicketID)
		temp.Labels.Severity = severity
		temp.Labels.Service = service
		temp.Labels.Tag = tag
		temp.Labels.Type = each.Type
		temp.Labels.VO = each.VO
		temp.Labels.Site = each.Site
		temp.Labels.Priority = each.Priority
		temp.Labels.Scope = each.Scope

		temp.Annotations.TicketID = strconv.Itoa(each.TicketID)
		temp.Annotations.Type = each.Type
		temp.Annotations.VO = each.VO
		temp.Annotations.Site = each.Site
		temp.Annotations.Priority = each.Priority
		temp.Annotations.ResponsibleUnit = each.ResponsibleUnit
		temp.Annotations.Status = each.Status
		temp.Annotations.Subject = each.Subject
		temp.Annotations.Scope = each.Scope
		temp.Annotations.URL = "https://ggus.eu/?mode=ticket_info&ticket_id=" + strconv.Itoa(each.TicketID)

		temp.StartsAt = _beginRFC3339
		temp.EndsAt = time.Unix(maxtstmp, 0).UTC()

		if verbose > 0 {
			log.Println("adding", temp.Labels.Alertname, temp.Labels.Severity, temp.Labels.Service, temp.Labels.Tag, temp.Annotations.URL)
		}
		finalData = append(finalData, temp)

	}

	jsonStr, err := json.Marshal(finalData)

	if err != nil {
		log.Printf("Unable to convert JSON Data, error: %v\n", err)
	}

	return jsonStr

}

//helper function for parsing list of alertmanager urls separated by comma
func parseURLs(urls string) []string {
	var urlList []string
	for _, url := range strings.Split(urls, ",") {
		urlList = append(urlList, strings.Trim(url, " "))
	}
	return urlList
}

//function for get request on /api/v1/alerts alertmanager endpoint for fetching alerts.
func get(alertManagerURL string) *ggusData {

	var data *ggusData

	//GET API for fetching only GGUS alerts.
	apiurl := alertManagerURL + "/api/v1/alerts?active=true&silenced=false&inhibited=false&unprocessed=false&filter=tag=\"GGUS\""

	req, err := http.NewRequest("GET", apiurl, nil)
	req.Header.Add("Accept-Encoding", "identity")
	req.Header.Add("Accept", "application/json")

	client := &http.Client{}

	if verbose > 1 {
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
		return data
	}

	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		log.Printf("Unable to parse JSON Data from AlertManager GET API, error: %v\n", err)
		return data
	}

	if verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}

	return data

}

//function for making post request on /api/v1/alerts alertmanager endpoint for creating alerts.
func post(jsonStr []byte, alertManagerURL string) {
	apiurl := alertManagerURL + "/api/v1/alerts"

	req, err := http.NewRequest("POST", apiurl, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	if verbose > 1 {
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

	if verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}
}

//Function to end alerts for tickets which are resolved
func deleteAlerts(alertManagerURL string) {
	amData := get(alertManagerURL)
	var temp amJSON
	var finalData []amJSON

	for _, each := range amData.Data {

		// If tickets which are in Existing Ticket Map then no need to send End Time, We Skip.
		if exstTkt[each.Annotations.TicketID] == 1 {
			continue
		}

		temp.Labels.Alertname = each.Labels.Alertname
		temp.Labels.Severity = each.Labels.Severity
		temp.Labels.Service = each.Labels.Service
		temp.Labels.Tag = each.Labels.Tag
		temp.Labels.Type = each.Labels.Type
		temp.Labels.VO = each.Labels.VO
		temp.Labels.Site = each.Labels.Site
		temp.Labels.Priority = each.Labels.Priority
		temp.Labels.Scope = each.Labels.Scope

		temp.Annotations.TicketID = each.Annotations.TicketID
		temp.Annotations.Type = each.Annotations.Type
		temp.Annotations.VO = each.Annotations.VO
		temp.Annotations.Site = each.Annotations.Site
		temp.Annotations.Priority = each.Annotations.Priority
		temp.Annotations.ResponsibleUnit = each.Annotations.ResponsibleUnit
		temp.Annotations.Status = each.Annotations.Status
		temp.Annotations.Subject = each.Annotations.Subject
		temp.Annotations.Scope = each.Annotations.Scope
		temp.Annotations.URL = each.Annotations.URL

		temp.StartsAt = each.StartsAt
		temp.EndsAt = time.Now().UTC()

		//Checking if Start Time is Afer Time Right now, maybe false alert added by mistake. **Corner Case to be handled**
		if each.StartsAt.Before(temp.EndsAt) == false {
			temp.StartsAt = time.Now().UTC()
			temp.EndsAt = time.Now().UTC()
		}

		finalData = append(finalData, temp)
	}

	jsonStr, err := json.Marshal(finalData)
	if err != nil {
		log.Printf("Unable to convert JSON Data, error: %v\n", err)
	}

	if verbose > 1 {
		fmt.Println("Deleted Alerts: ", string(jsonStr))
	}

	post(jsonStr, alertManagerURL)
}

//Machine Learning Logic for Categorizing GGUS Tickets
func categorizeTicket() {
	// To be Implemented
}

//function containing all logics for alerting.
func alert(inp string, dryRun bool) {

	alertManagerURLList = parseURLs(alertManagerURLs)
	jsonData := fetchJSON(inp)
	var data ggus
	data.parseJSON(jsonData)
	jsonStrAM := convertData(data) //JSON data in AlertManager APIs format.
	if dryRun {
		fmt.Println(string(jsonStrAM))
		return
	}

	for _, alertManagerURL := range alertManagerURLList {
		post(jsonStrAM, alertManagerURL)
		deleteAlerts(alertManagerURL)
	}

}

func main() {

	var inp string
	severity = "ticket" // TODO: replace with severity level from ticket annotation
	tag = "monitoring"
	service = "GGUS"
	var dryRun bool

	flag.StringVar(&inp, "input", "", "input filename")
	flag.StringVar(&vo, "vo", "cms", "Required VO attribute in GGUS Ticket")
	flag.StringVar(&alertManagerURLs, "urls", "", "list of alertmanager URLs seperated by commas")
	flag.IntVar(&verbose, "verbose", 0, "verbosity level")
	flag.BoolVar(&dryRun, "dryRun", false, "dry run mode, fetch data but do not post it to AM")
	flag.Parse()

	if inp == "" {
		log.Fatalf("Input filename missing. Exiting....")
	}

	if alertManagerURLs == "" {
		log.Fatalf("AlertManager URLs missing. Exiting....")
	}

	if verbose > 0 {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}

	alert(inp, dryRun)
}
