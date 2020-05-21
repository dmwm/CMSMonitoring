package main

import (
	"bytes"
	"encoding/json"
	"flag"
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
		Alertname string `json:"alertname"`
		Severity  string `json:"severity"`
		Tag       string `json:"tag"`
	} `json:"labels"`
	Annotations struct {
		Date             string `json:"date"`
		Description      string `json:"description"`
		FeName           string `json:"feName"`
		MonitState       string `json:"monitState"`
		MonitState1      string `json:"monitState1"`
		SeName           string `json:"seName"`
		ShortDescription string `json:"shortDescription"`
		SsbNumber        string `json:"ssbNumber"`
		SysCreatedBy     string `json:"sysCreatedBy"`
		SysModCount      string `json:"sysModCount"`
		SysUpdatedBy     string `json:"sysUpdatedBy"`
		Type             string `json:"type"`
		UpdateTimestamp  string `json:"updateTimestamp"`
	} `json:"annotations"`
	StartsAt string `json:"startsAt"`
	EndsAt   string `json:"endsAt"`
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

//function for converting SSB JSON Data into JSON Data required by AlertManager APIs.
func (data *ssb) convertData() []byte {

	var temp amJSON
	var finalData []amJSON

	for _, each := range data.Results[0].Series[0].Values {

		begin := int64(each[1].(float64))
		end := int64(each[4].(float64))
		updatets := int64(each[15].(float64))

		_beginRFC3339 := time.Unix(0, begin*int64(time.Millisecond)).UTC().Format(time.RFC3339)
		_endRFC3339 := time.Unix(0, end*int64(time.Millisecond)).UTC().Format(time.RFC3339)
		_updatetsRFC3339 := time.Unix(0, updatets*int64(time.Millisecond)).UTC().Format(time.RFC3339)

		temp.Labels.Alertname = each[2].(string)
		temp.Labels.Severity = severity
		temp.Labels.Tag = tag

		temp.Annotations.Date = each[0].(string)
		temp.Annotations.Description = each[2].(string)
		temp.Annotations.FeName = each[5].(string)
		temp.Annotations.MonitState = each[6].(string)
		temp.Annotations.MonitState1 = each[7].(string)
		temp.Annotations.SeName = each[8].(string)
		temp.Annotations.ShortDescription = each[9].(string)
		temp.Annotations.SsbNumber = each[10].(string)
		temp.Annotations.SysCreatedBy = each[11].(string)
		temp.Annotations.SysModCount = each[12].(string)
		temp.Annotations.SysUpdatedBy = each[13].(string)
		temp.Annotations.Type = each[14].(string)
		temp.Annotations.UpdateTimestamp = _updatetsRFC3339

		temp.StartsAt = _beginRFC3339
		temp.EndsAt = _endRFC3339

		finalData = append(finalData, temp)

	}

	jsonStr, err := json.Marshal(finalData)

	if err != nil {
		log.Printf("Unable to convert JSON Data, error: %v\n", err)
	}

	return jsonStr

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

//function containing all logics for alerting.
func alert(inp string) {

	jsonData := fetchJSON(inp)
	var data ssb
	data.parseJSON(jsonData)
	jsonStrAM := data.convertData() //JSON data in AlertManager APIs format.
	post(jsonStrAM)

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
