package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"
)

//URL for AlertManager
var alertManagerURL string

//Severity of alerts
var severity string

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

//function for parsing JSON data from CERN SSB Data
func (data *ssb) parseJSON(jsondata []byte) {
	json.Unmarshal(jsondata, &data)
}

//function for fetching JSON data from CERN SSB Data
func fetchJSON(filename string) []byte {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Unable to open JSON file, error: %v\n", err)
	}

	jsonData, err := ioutil.ReadAll(file)
	if err != nil {
		log.Printf("Unable to read JSON file, error: %v\n", err)
	}

	return jsonData

}

//helper function for amtool.
func amtool(alertname,
	start,
	end,
	date,
	description,
	fe_name,
	monit_state,
	monit_state_1,
	se_name,
	short_description,
	ssb_number,
	sys_created_by,
	sys_mod_count,
	sys_updated_by,
	a_type,
	update_timestamp string) {

	cmd := exec.Command("amtool", "alert", "add", alertname, "severity="+severity,
		"--annotation=date="+date,
		"--annotation=description="+description,
		"--annotation=fe_name="+fe_name,
		"--annotation=monit_state="+monit_state,
		"--annotation=monit_state_1="+monit_state_1,
		"--annotation=se_name="+se_name,
		"--annotation=short_description="+short_description,
		"--annotation=ssb_number="+ssb_number,
		"--annotation=sys_created_by="+sys_created_by,
		"--annotation=sys_mod_count="+sys_mod_count,
		"--annotation=sys_updated_by="+sys_updated_by,
		"--annotation=type="+a_type,
		"--annotation=update_timestamp="+update_timestamp,
		"--start", start, "--end", end, "--alertmanager.url", alertManagerURL)
	_, err := cmd.Output()

	if err != nil {
		log.Printf("Unable to run amtool command, error: %v\n", err)
		return
	}

}

//function for pushing each data from JSON file to AlertManager.
func (data *ssb) pushData() {

	for _, each := range data.Results[0].Series[0].Values {

		begin := int64(each[1].(float64))
		end := int64(each[4].(float64))
		updatets := int64(each[15].(float64))

		_beginRFC3339 := time.Unix(0, begin*int64(time.Millisecond)).UTC().Format(time.RFC3339)
		_endRFC3339 := time.Unix(0, end*int64(time.Millisecond)).UTC().Format(time.RFC3339)
		_updatetsRFC3339 := time.Unix(0, updatets*int64(time.Millisecond)).UTC().Format(time.RFC3339)

		alertname := each[2].(string)
		date := each[0].(string)
		description := each[2].(string)
		fe_name := each[5].(string)
		monit_state := each[6].(string)
		monit_state_1 := each[7].(string)
		se_name := each[8].(string)
		short_description := each[9].(string)
		ssb_number := each[10].(string)
		sys_created_by := each[11].(string)
		sys_mod_count := each[12].(string)
		sys_updated_by := each[13].(string)
		a_type := each[14].(string)

		amtool(alertname, _beginRFC3339, _endRFC3339, date,
			description, fe_name, monit_state, monit_state_1, se_name,
			short_description, ssb_number, sys_created_by, sys_mod_count,
			sys_updated_by, a_type, _updatetsRFC3339)
	}

}

func main() {

	var in string
	alertManagerURL = "http://localhost:9093"
	severity = "monitoring"

	flag.StringVar(&in, "input", "", "input filename")
	flag.Parse()

	if in == "" {
		log.Fatalf("Input filename missing. Exiting....")
	}

	jsonData := fetchJSON(in)
	var data ssb
	data.parseJSON(jsonData)
	data.pushData()

}
