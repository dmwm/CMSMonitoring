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
	"time"
)

// File       : ssb_alerting.go
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Thu, 16 May 2020 16:45:00 GMT
// Description: SSB Alerting Module for CERN MONIT infrastructure

//URL for AlertManager
var alertManagerURL string

//severity of alerts
var severity string

//SSB tag
var tag string

//verbose defines verbosity level
var verbose int

//MAX timeStamp //Saturday, May 24, 3000 3:43:26 PM
var maxtstmp int64 = 32516091806

//Map for storing Existing SSB Data
var exstSSBData map[string]int

//CERN SSB Data Struct
type ssb struct {
	Results []struct {
		Series []struct {
			Columns []string        `json:"columns"`
			Name    string          `json:"name"`
			Values  [][]interface{} `json:"values"`
		} `json:"series"`
		StatementID int `json:"statement_id"`
	} `json:"results"`
}

//AlertManager API acceptable JSON Data for CERN SSB Data
type amJSON struct {
	Labels struct {
		Alertname   string `json:"alertname"`
		Severity    string `json:"severity"`
		Tag         string `json:"tag"`
		Type        string `json:"type"`
		Description string `json:"description"`
		FeName      string `json:"feName"`
		SeName      string `json:"seName"`
	} `json:"labels"`
	Annotations struct {
		Date             string    `json:"date"`
		Description      string    `json:"description"`
		FeName           string    `json:"feName"`
		MonitState       string    `json:"monitState"`
		MonitState1      string    `json:"monitState1"`
		SeName           string    `json:"seName"`
		ShortDescription string    `json:"shortDescription"`
		SsbNumber        string    `json:"ssbNumber"`
		SysCreatedBy     string    `json:"sysCreatedBy"`
		SysModCount      string    `json:"sysModCount"`
		SysUpdatedBy     string    `json:"sysUpdatedBy"`
		Type             string    `json:"type"`
		UpdateTimestamp  time.Time `json:"updateTimestamp"`
	} `json:"annotations"`
	StartsAt time.Time `json:"startsAt"`
	EndsAt   time.Time `json:"endsAt"`
}

//AlertManager GET API acceptable JSON Data struct for SSB data
type ssbData struct {
	Data []amJSON
}

//function for parsing JSON data from CERN SSB Data
func (data *ssb) parseJSON(jsondata []byte) {
	err := json.Unmarshal(jsondata, &data)
	if err != nil {
		log.Printf("Unable to parse CERN SSB JSON data, error: %v\n", err)
	}
}

//function for fetching JSON data from CERN SSB Data
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
		log.Println("CERN SSB Data: " + string(jsonData))
	}

	return jsonData

}

//function for eliminating "none" or empty values ("null") in JSON
func nullValueChecker(target *string, data interface{}) {

	if b, ok := data.(string); ok {
		*target = b
	}
}

//function for converting SSB JSON Data into JSON Data required by AlertManager APIs.
func (data *ssb) convertData() []byte {

	var finalData []amJSON
	exstSSBData = make(map[string]int)

	for _, each := range data.Results[0].Series[0].Values {
		var temp amJSON
		var _beginTS, _endTS, _updateTS time.Time
		var ssbNum string
		nullValueChecker(&ssbNum, each[10])

		exstSSBData[ssbNum] = 1

		if b, ok := each[1].(float64); ok {
			_beginTS = time.Unix(0, int64(b)*int64(time.Millisecond)).UTC()
		}

		if e, ok := each[4].(float64); ok {
			_endTS = time.Unix(0, int64(e)*int64(time.Millisecond)).UTC()

		} else {
			_endTS = time.Unix(maxtstmp, 0).UTC() // If not given EndTime the alert will be open ending. Max TimeStamp value given.
		}

		if u, ok := each[15].(float64); ok {
			_updateTS = time.Unix(0, int64(u)*int64(time.Millisecond)).UTC()
		}

		temp.Labels.Alertname = "ssb-" + ssbNum //ssbNumber as an unique key for alertname
		temp.Labels.Severity = severity
		temp.Labels.Tag = tag
		nullValueChecker(&temp.Labels.Description, each[2])
		nullValueChecker(&temp.Labels.FeName, each[5])
		nullValueChecker(&temp.Labels.SeName, each[8])
		nullValueChecker(&temp.Labels.Type, each[14])

		nullValueChecker(&temp.Annotations.Date, each[0])
		nullValueChecker(&temp.Annotations.Description, each[2])
		nullValueChecker(&temp.Annotations.FeName, each[5])
		nullValueChecker(&temp.Annotations.MonitState, each[6])
		nullValueChecker(&temp.Annotations.MonitState1, each[7])
		nullValueChecker(&temp.Annotations.SeName, each[8])
		nullValueChecker(&temp.Annotations.ShortDescription, each[9])
		nullValueChecker(&temp.Annotations.SsbNumber, each[10])
		nullValueChecker(&temp.Annotations.SysCreatedBy, each[11])
		nullValueChecker(&temp.Annotations.SysModCount, each[12])
		nullValueChecker(&temp.Annotations.SysUpdatedBy, each[13])
		nullValueChecker(&temp.Annotations.Type, each[14])

		temp.Annotations.UpdateTimestamp = _updateTS

		temp.StartsAt = _beginTS
		temp.EndsAt = _endTS

		finalData = append(finalData, temp)

	}

	jsonStr, err := json.Marshal(finalData)

	if err != nil {
		log.Printf("Unable to convert JSON Data, error: %v\n", err)
	}

	return jsonStr

}

//function for get request on /api/v1/alerts alertmanager endpoint for fetching alerts.
func get() *ssbData {

	var data *ssbData

	//GET API for fetching only GGUS alerts.
	apiurl := alertManagerURL + "/api/v1/alerts?active=true&silenced=false&inhibited=false&unprocessed=false&filter=tag=\"SSB\""

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
func post(jsonStr []byte) {
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

//Function to end alerts for SSB data which had no EndTime and now they are resolved.
func deleteAlerts() {
	amData := get()
	var finalData []amJSON

	for _, each := range amData.Data {
		var temp amJSON

		// If SSB Data which are in Existing SSB Data Map then no need to send End Time, We Skip.
		if exstSSBData[each.Annotations.SsbNumber] == 1 {
			continue
		}

		temp.Labels.Alertname = each.Labels.Alertname
		temp.Labels.Severity = each.Labels.Severity
		temp.Labels.Tag = each.Labels.Tag
		temp.Labels.Description = each.Labels.Description
		temp.Labels.FeName = each.Labels.FeName
		temp.Labels.SeName = each.Labels.SeName
		temp.Labels.Type = each.Labels.Type

		temp.Annotations.Date = each.Annotations.Date
		temp.Annotations.Description = each.Annotations.Description
		temp.Annotations.FeName = each.Annotations.FeName
		temp.Annotations.MonitState = each.Annotations.MonitState
		temp.Annotations.MonitState1 = each.Annotations.MonitState1
		temp.Annotations.SeName = each.Annotations.SeName
		temp.Annotations.ShortDescription = each.Annotations.ShortDescription
		temp.Annotations.SsbNumber = each.Annotations.SsbNumber
		temp.Annotations.SysCreatedBy = each.Annotations.SysCreatedBy
		temp.Annotations.SysModCount = each.Annotations.SysModCount
		temp.Annotations.SysUpdatedBy = each.Annotations.SysUpdatedBy
		temp.Annotations.Type = each.Annotations.Type
		temp.Annotations.UpdateTimestamp = each.Annotations.UpdateTimestamp

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

	post(jsonStr)
}

//function containing all logics for alerting.
func alert(inp string) {

	jsonData := fetchJSON(inp)
	var data ssb
	data.parseJSON(jsonData)
	jsonStrAM := data.convertData() //JSON data in AlertManager APIs format.
	post(jsonStrAM)
	deleteAlerts()

}

func main() {

	var inp string
	severity = "monitoring"
	tag = "SSB"

	flag.StringVar(&inp, "input", "", "input filename")
	flag.StringVar(&alertManagerURL, "url", "", "alertmanager URL")
	flag.IntVar(&verbose, "verbose", 0, "verbosity level")
	flag.Parse()

	if inp == "" {
		log.Fatalf("Input filename missing. Exiting....")
	}

	if alertManagerURL == "" {
		log.Fatalf("AlertManager URL missing. Exiting....")
	}

	if verbose > 0 {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}

	alert(inp)

}
