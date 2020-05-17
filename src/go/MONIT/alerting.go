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

// File       : alerting.go
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Thu, 16 May 2020 16:45:00 GMT
// Description: Alerting Module for CERN MONIT infrastructure

//URL for AlertManager
var alertManagerURL string

//severity of alerts
var severity string

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
	} `json:"labels"`
	Annotations struct {
		Date              string `json:"date"`
		Description       string `json:"description"`
		Fe_name           string `json:"fe_name"`
		Monit_state       string `json:"monit_state"`
		Monit_state_1     string `json:"monit_state_1"`
		Se_name           string `json:"se_name"`
		Short_description string `json:"short_description"`
		Ssb_number        string `json:"ssb_number"`
		Sys_created_by    string `json:"sys_created_by"`
		Sys_mod_count     string `json:"sys_mod_count"`
		Sys_updated_by    string `json:"sys_updated_by"`
		Type              string `json:"type"`
		Update_timestamp  string `json:"update_timestamp"`
	} `json:"annotations"`
	Starts_at string `json:"startsAt"`
	Ends_at   string `json:"endsAt"`
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

		temp.Annotations.Date = each[0].(string)
		temp.Annotations.Description = each[2].(string)
		temp.Annotations.Fe_name = each[5].(string)
		temp.Annotations.Monit_state = each[6].(string)
		temp.Annotations.Monit_state_1 = each[7].(string)
		temp.Annotations.Se_name = each[8].(string)
		temp.Annotations.Short_description = each[9].(string)
		temp.Annotations.Ssb_number = each[10].(string)
		temp.Annotations.Sys_created_by = each[11].(string)
		temp.Annotations.Sys_mod_count = each[12].(string)
		temp.Annotations.Sys_updated_by = each[13].(string)
		temp.Annotations.Type = each[14].(string)
		temp.Annotations.Update_timestamp = _updatetsRFC3339

		temp.Starts_at = _beginRFC3339
		temp.Ends_at = _endRFC3339

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

	if verbose >= 1 {
		log.Println("Response Status:", resp.Status)
		log.Println("Response Headers:", resp.Header)
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Unable to read Response Body, error: %v\n", err)
		}
		log.Println("Response Body:", string(body))
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
