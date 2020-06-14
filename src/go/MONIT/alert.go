package main

// File       : alert.go
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Tue, 9 June 2020 11:04:10 GMT
// Description: CERN MONIT infrastructure Alert CLI Tool

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

//-------VARIABLES-------
//URL for AlertManager
var cmsmonURL string

//MAX timeStamp //Saturday, May 24, 3000 3:43:26 PM
var maxtstmp int64 = 32516091806

//alertname
var name string

//service name
var service string

//tag name
var tag string

//severity level
var severity string

//boolean for JSON output
var jsonOutput *bool

//boolean for detailed output for an alert
var details *bool

//Sort Label
var sortLabel string

//Severity Level Config filepath
var severityConfig string

//verbose defines verbosity level
var verbose int

//-------VARIABLES-------

//-------MAPS-------

//Map for storing filtered alerts
var filteredAlerts map[string]int

//Map for storing Alert details against their name
var alertDetails map[string]amJSON

//-------MAPS-------

//-------STRUCTS---------
//AlertManager API acceptable JSON Data for GGUS Data
type amJSON struct {
	Labels      map[string]interface{} `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`
	StartsAt    time.Time              `json:"startsAt"`
	EndsAt      time.Time              `json:"endsAt"`
}

type amData struct {
	Data []amJSON
}

//Alert CLI tool data struct (Tabular)
type alertData struct {
	Name     string
	Service  string
	Tag      string
	Severity string
	StartsAt time.Time
	EndsAt   time.Time
}

//Array of alerts for alert CLI Tool (Tabular)
var allAlertData []alertData

//Alert CLI tool data struct (JSON)
type alertDataJSON struct {
	Name     string
	Service  string
	Tag      string
	Severity string
	Starts   string
	Ends     string
	Duration string
}

type severityLevels struct {
	SeverityLevels map[string]int `json:"severity"`
}

var sLevel severityLevels

//-------STRUCTS---------

//function for get request on /api/v1/alerts alertmanager endpoint for fetching alerts.
func get(data interface{}) {

	//GET API for fetching only GGUS alerts.
	apiurl := cmsmonURL + "/api/v1/alerts?active=true&silenced=false&inhibited=false&unprocessed=false"

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
		log.Printf("Unable to read %s JSON Data from AlertManager GET API, error: %v\n", service, err)
		return
	}

	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		log.Printf("Unable to parse %s JSON Data from AlertManager GET API, error: %v\n", service, err)
		return
	}

	if verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("Response: ", string(dump))
		}
	}

}

//function for merging all alerts from various services at one place
func mergeData(amdata amData) {
	alertDetails = make(map[string]amJSON)

	for _, each := range amdata.Data {
		var temp alertData

		for key, value := range each.Labels {
			switch key {
			case "alertname":
				alertDetails[value.(string)] = each
				temp.Name = value.(string)

			case "severity":
				temp.Severity = value.(string)

			case "service":
				temp.Service = value.(string)

			case "tag":
				temp.Tag = value.(string)
			}

		}
		temp.StartsAt = each.StartsAt
		temp.EndsAt = each.EndsAt
		allAlertData = append(allAlertData, temp)
	}
}

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

//Helper function for Filtering
func filterHelper(each alertData) int {

	if service == "" && severity == "" && tag == "" {
		return 1
	} else if service == "" && severity == "" && tag == each.Tag {
		return 1
	} else if service == "" && severity == each.Severity && tag == "" {
		return 1
	} else if service == "" && severity == each.Severity && tag == each.Tag {
		return 1
	} else if service == each.Service && severity == "" && tag == "" {
		return 1
	} else if service == each.Service && severity == "" && tag == each.Tag {
		return 1
	} else if service == each.Service && severity == each.Severity && tag == "" {
		return 1
	} else if service == each.Service && severity == each.Severity && tag == each.Tag {
		return 1
	} else {
		return 0
	}
}

//Function for Filtering
func filter() {
	filteredAlerts = make(map[string]int)
	for _, each := range allAlertData {
		if filterHelper(each) == 0 {
			continue
		}
		filteredAlerts[each.Name] = 1
	}
}

//Sorting Logic
type durationSorter []alertData

func (d durationSorter) Len() int      { return len(d) }
func (d durationSorter) Swap(i, j int) { d[i], d[j] = d[j], d[i] }
func (d durationSorter) Less(i, j int) bool {
	return d[i].EndsAt.Sub(d[i].StartsAt) <= d[j].EndsAt.Sub(d[j].StartsAt)
}

type severitySorter []alertData

func (s severitySorter) Len() int      { return len(s) }
func (s severitySorter) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s severitySorter) Less(i, j int) bool {
	return sLevel.SeverityLevels[s[i].Severity] < sLevel.SeverityLevels[s[j].Severity]
}

type startAtSorter []alertData

func (s startAtSorter) Len() int      { return len(s) }
func (s startAtSorter) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s startAtSorter) Less(i, j int) bool {
	return s[j].StartsAt.After(s[i].StartsAt)
}

type endsAtSorter []alertData

func (e endsAtSorter) Len() int      { return len(e) }
func (e endsAtSorter) Swap(i, j int) { e[i], e[j] = e[j], e[i] }
func (e endsAtSorter) Less(i, j int) bool {
	return e[j].EndsAt.After(e[i].EndsAt)
}

//Function for sorting alerts based on a passed label
func sortAlert() {

	switch strings.ToLower(sortLabel) {
	case "severity":
		sort.Sort(severitySorter(allAlertData))
	case "starts":
		sort.Sort(startAtSorter(allAlertData))
	case "ends":
		sort.Sort(endsAtSorter(allAlertData))
	case "duration":
		sort.Sort(durationSorter(allAlertData))
	default:
		return
	}
}

//Function for printing alerts in JSON format
func jsonPrint() {

	var filteredData []alertDataJSON
	var temp alertDataJSON

	for _, each := range allAlertData {
		if filteredAlerts[each.Name] == 1 {
			temp.Name = each.Name
			temp.Service = each.Service
			temp.Severity = each.Service
			temp.Tag = each.Tag
			temp.Starts = timeDiff(time.Now(), each.StartsAt, 0)
			if each.EndsAt == time.Unix(maxtstmp, 0).UTC() {
				temp.Ends = "Undefined"
				temp.Duration = "Undefined"
			} else {
				temp.Ends = timeDiff(time.Now(), each.EndsAt, 0)
				temp.Duration = timeDiff(each.StartsAt, each.EndsAt, 1)
			}

			filteredData = append(filteredData, temp)
		}
	}

	b, err := json.Marshal(filteredData)

	if err != nil {
		log.Printf("Unable to convert Filtered JSON Data, error: %v\n", err)
		return
	}

	fmt.Println(string(b))
}

//Function for printing alerts in Plain text format
func tabulate() {

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 0, '\t', 0)
	defer w.Flush()

	fmt.Fprintf(w, "\n %s\t\t%s\t\t%s\t\t%s\t\t%s\t\t%s\t\t%s\n", "NAME", "SERVICE", "TAG", "SEVERITY", "STARTS", "ENDS", "DURATION")

	for _, each := range allAlertData {
		if filteredAlerts[each.Name] == 1 {
			fmt.Fprintf(w, " %s\t\t%s\t\t%s\t\t%s\t\t%s",
				each.Name,
				each.Service,
				each.Tag,
				each.Severity,
				timeDiff(time.Now(), each.StartsAt, 0),
			)
			if each.EndsAt == time.Unix(maxtstmp, 0).UTC() {
				fmt.Fprintf(w, "\t\t%s", "Undefined")
				fmt.Fprintf(w, "\t\t%s\n", "Undefined")
			} else {
				fmt.Fprintf(w, "\t\t%s", timeDiff(time.Now(), each.EndsAt, 0))
				fmt.Fprintf(w, "\t\t%s\n", timeDiff(each.StartsAt, each.EndsAt, 1))
			}

		}
	}
}

//Function for printing alert's details in JSON format
func jsonPrintDetails() {

	b, err := json.Marshal(alertDetails[name])

	if err != nil {
		log.Printf("Unable to convert Detailed JSON Data, error: %v\n", err)
		return
	}

	fmt.Println(string(b))
}

//Function for printing alert's details in Plain text format
func detailPrint() {

	if alert, ok := alertDetails[name]; ok { //use of two different loops for LABELS as use of map[string]interface{} in amJSON causes unordered LABELS.
		for key, value := range alert.Labels { // i.e. "severity", "tag", "service" could arrive before "alertname"
			if key == "alertname" {
				fmt.Printf("NAME: %s\n", value)
				break
			}
		}
		fmt.Printf("LABELS\n")
		for key, value := range alert.Labels {
			switch key {
			case "service", "tag", "severity":
				fmt.Printf("\t%s: %s\n", key, value)
			}
		}
		fmt.Printf("ANNOTATIONS\n")
		for key, value := range alert.Annotations {
			fmt.Printf("\t%s: %s\n", key, value)
		}
	} else {
		fmt.Printf("%s alert not found\n", name)
	}

}

//Function running all logics
func run() {

	//Default Severity Levels in case no config file is provided
	if severityConfig == "" {
		sLevel.SeverityLevels = make(map[string]int)
		sLevel.SeverityLevels["info"] = 0
		sLevel.SeverityLevels["warning"] = 1
		sLevel.SeverityLevels["medium"] = 2
	} else {
		jsonFile, e := os.Open(severityConfig)
		if e != nil {
			fmt.Println("Severity Config File not found, error:", e)
		}
		defer jsonFile.Close()
		decoder := json.NewDecoder(jsonFile)
		err := decoder.Decode(&sLevel)
		if err != nil {
			fmt.Println("Severity Config JSON File can't be loaded, error:", err)
		}
	}

	fmt.Println(sLevel)

	var amdata amData
	get(&amdata)
	mergeData(amdata)
	sortAlert()
	filter()

	if *jsonOutput {
		if *details {
			jsonPrintDetails()
		} else {
			jsonPrint()
		}
	} else {
		if *details {
			detailPrint()
		} else {
			tabulate()
		}
	}
}

func main() {

	flag.StringVar(&cmsmonURL, "url", os.Getenv("CMSMON_URL"), "CMS Monit URL")
	flag.StringVar(&name, "name", "", "Alert Service Name (GGUS/SSB)")
	flag.StringVar(&severity, "severity", "", "Severity Level of alerts")
	flag.StringVar(&tag, "tag", "", "Tag for alerts")
	flag.StringVar(&service, "service", "", "Service Name")
	jsonOutput = flag.Bool("json", false, "Output in JSON format")
	flag.StringVar(&sortLabel, "sort", "", "Sort data on a specific Label")
	details = flag.Bool("details", false, "Detailed output for an alert")
	flag.StringVar(&severityConfig, "sconfig", os.Getenv("SEVERITY_CONFIG"), "Severity Level Config filepath")
	flag.IntVar(&verbose, "verbose", 0, "verbosity level")

	flag.Parse()

	run()
}
