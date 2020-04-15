package main

// File       : monit.go
// Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
// Created    : Fri, 03 Apr 2020 10:48:20 GMT
// Description: client for CERN MONIT infrastructure

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/go-stomp/stomp"
)

// Record represents MONIT return record {"response":...}
type Record map[string]interface{}
type DSRecord map[string]Record

// DataSources keeps global map of MONIT datasources
var DataSources DSRecord

// helper function to either read file content or return given string
func read(r string) string {
	if _, err := os.Stat(r); err == nil {
		b, e := ioutil.ReadFile(r)
		if e != nil {
			log.Fatalf("Unable to read data from file: %s, error: %s", r, e)
		}
		return strings.Replace(string(b), "\n", "", -1)
	}
	return r
}

// return CMS Monitoring datasources
func datasources() (DSRecord, error) {
	rurl := "https://raw.githubusercontent.com/dmwm/CMSMonitoring/master/static/datasources.json"
	resp, err := http.Get(rurl)
	if err != nil {
		log.Printf("Unable to fetch datasources from %s, error %v\n", rurl, err)
		return nil, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Unable to read, url: %s, error: %v\n", rurl, err)
		return nil, err
	}
	var rec DSRecord
	err = json.Unmarshal(data, &rec)
	if err != nil {
		log.Printf("Unable to parse, url: %s, error: %v\n", rurl, err)
		return nil, err
	}
	return rec, nil
}

// helper function to find MONIT datasource id
func findDataSource(pat string) (int, string, string) {
	for src, d := range DataSources {
		if v, ok := d["database"]; ok {
			db := v.(string)
			if db == pat || src == pat || (strings.Contains(pat, "*") && strings.Contains(db, pat)) {
				if did, ok := d["id"]; ok {
					t, _ := d["type"]
					return int(did.(float64)), v.(string), t.(string)
				}
			}
		}
	}
	return 0, "", ""
}

// StompConfig represents stomp configuration
type StompConfig struct {
	Producer string `json:"producer"`
	URI      string `json:"host_and_ports"`
	Login    string `json:"username"`
	Password string `json:"password"`
	Topic    string `json:"topic"`
}

func (c *StompConfig) String() string {
	return fmt.Sprintf("<StompConfig uri=%s producer=%s topic=%s>", c.URI, c.Producer, c.Topic)
}

func sendDataToStomp(config StompConfig, data []byte, verbose int) {
	contentType := "application/json"
	conn, err := stomp.Dial("tcp",
		config.URI,
		stomp.ConnOpt.Login(config.Login, config.Password))
	defer conn.Disconnect()
	if err != nil {
		log.Fatalf("Unable to connect to %s, error %v", config.URI, err)
	}
	if verbose > 0 {
		log.Printf("connected to StompAMQ server %s", config.URI)
	}
	if conn != nil {
		err = conn.Send(config.Topic, contentType, data)
		if err != nil {
			log.Printf("unable to send data to %s, error %v", config.Topic, err)
		}
		if verbose > 0 {
			log.Println("Send data to MONIT", err, conn, string(data))
		}
	}
}

// helper function to update doc in MONIT
func injectToMonit(creds, fname string, verbose int) {
	data, err := ioutil.ReadFile(creds)
	if err != nil {
		log.Fatalf("Unable to read, file: %s, error: %v\n", creds, err)
	}
	var config StompConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Unable to parse, file: %s, error: %v\n", creds, err)
	}
	if verbose > 0 {
		log.Println("StompConfig:", config.String())
	}
	data, err = ioutil.ReadFile(fname)
	if err != nil {
		log.Fatalf("Unable to read, file: %s, error: %v\n", fname, err)
	}
	sendDataToStomp(config, data, verbose)
}

// helper function to query InfluxDB
func queryIDB(base string, dbid int, dbname, query string, headers [][]string, verbose int) Record {
	rurl := fmt.Sprintf("%s/api/datasources/proxy/%d/query?db=%s&q=%s", base, dbid, dbname, query)
	if verbose > 0 {
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
	if verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("request: ", string(dump))
		}
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Unable to get response from %s, error: %s", rurl, err)
	}
	if verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("response:", string(dump))
		}
	}
	var data Record
	defer resp.Body.Close()
	// Deserialize the response into a map.
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	return data
}

func queryES(base string, dbid int, dbname, query string, headers [][]string, verbose int) Record {
	// Method to query ES DB
	// https://www.elastic.co/guide/en/elasticsearch/reference/5.5/search-multi-search.html
	rurl := fmt.Sprintf("%s/api/datasources/proxy/%d/_msearch", base, dbid)
	dbname = strings.Replace(dbname, "[", "", -1)
	dbname = strings.Replace(dbname, "]", "", -1)
	dbname = strings.Replace(dbname, "_*", "", -1)
	dbname = strings.Replace(dbname, "*", "", -1)
	q := fmt.Sprintf("{\"search_type\": \"query_then_fetch\", \"index\": [\"%s*\"], \"ignore_unavailable\": true}\n%s\n", dbname, query)
	if verbose > 0 {
		log.Println(rurl, q)
	}
	req, err := http.NewRequest("GET", rurl, strings.NewReader(q))
	if err != nil {
		log.Fatalf("Unable to make request to %s, error: %s", rurl, err)
	}
	for _, v := range headers {
		if len(v) == 2 {
			req.Header.Add(v[0], v[1])
		}
	}
	req.Header.Add("Content-type", "application/x-ndjson")
	if verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("request: ", string(dump))
		}
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Unable to get response from %s, error: %s", rurl, err)
	}
	if verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("response: ", string(dump))
		}
	}
	defer resp.Body.Close()
	var data Record
	// Deserialize the response into a map.
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	return data
}

func queryURL(rurl string, headers [][]string, verbose int) Record {
	if verbose > 0 {
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
	if verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("request: ", string(dump))
		}
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Unable to get response from %s, error: %s", rurl, err)
	}
	if verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("response: ", string(dump))
		}
	}
	defer resp.Body.Close()
	var data Record
	// Deserialize the response into a map.
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	return data
}

func run(rurl, token string, dbid int, dbname, query string, idx, limit, verbose int) Record {
	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", token)
	h := []string{"Authorization", bearer}
	headers = append(headers, h)
	h = []string{"Accept", "application/json"}
	headers = append(headers, h)

	q := strings.ToLower(query)
	if strings.Contains(q, "elasticsearch") {
		return queryES(rurl, dbid, dbname, query, headers, verbose)
	} else if strings.Contains(q, "query") {
		return queryES(rurl, dbid, dbname, query, headers, verbose)
	} else if strings.Contains(q, "select") {
		return queryIDB(rurl, dbid, dbname, query, headers, verbose)
	} else if strings.Contains(q, "show") {
		return queryIDB(rurl, dbid, dbname, query, headers, verbose)
	}
	// perform query for given url
	return queryURL(rurl, headers, verbose)
}

func main() {
	defaultUrl := "https://monit-grafana.cern.ch"
	var verbose int
	flag.IntVar(&verbose, "verbose", 0, "verbosity level")
	var url string
	flag.StringVar(&url, "url", defaultUrl, "MONIT URL")
	var token string
	flag.StringVar(&token, "token", "", "MONIT token or token file")
	var dbid int
	flag.IntVar(&dbid, "dbid", 0, "MONIT db identified")
	var dbname string
	flag.StringVar(&dbname, "dbname", "", "MONIT dbname")
	var query string
	flag.StringVar(&query, "query", "", "query string or query json file")
	var inject string
	flag.StringVar(&inject, "inject", "", "inject given json document to MONIT")
	var creds string
	flag.StringVar(&creds, "creds", "", "json document with MONIT credentials")
	var idx int
	flag.IntVar(&idx, "idx", 0, "verbosity level")
	var limit int
	flag.IntVar(&limit, "limit", 0, "verbosity level")
	var listDataSources bool
	flag.BoolVar(&listDataSources, "datasources", false, "List MONIT datasources")
	flag.Usage = func() {
		fmt.Println("Usage: monit [options]")
		flag.PrintDefaults()
		fmt.Println("Examples:")
		fmt.Println("   # inject data into MONIT")
		fmt.Println("   monit -creds=creds_udp.json -inject=cmssw.json -verbose 1")
		fmt.Println("")
		fmt.Println("   # look-up data from MONIT")
		fmt.Println("   monit -token token -query=query.json -dbname=monit_prod_wmagent")
		fmt.Println("")
		fmt.Println("   # look-up all available datasources in MONIT")
		fmt.Println("   monit -datasources")
	}
	flag.Parse()

	if verbose > 0 {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}
	var e error
	DataSources, e = datasources()
	if listDataSources {
		data, err := json.MarshalIndent(DataSources, "", "\t")
		if err == nil {
			fmt.Println(string(data))
			os.Exit(0)
		}
	}
	if inject != "" && creds != "" {
		injectToMonit(creds, inject, verbose)
		return
	}
	t := read(token)
	q := read(query)
	if dbname == "" && url == defaultUrl {
		log.Fatalf("Please provide valid dbname")
	}
	var database, dbtype string
	dbid, database, dbtype = findDataSource(dbname)
	if dbid == 0 {
		log.Fatalf("No valid dbid found for %s", dbname)
	}
	if token == "" {
		log.Fatalf("Please provide valid token")
	}
	if verbose > 1 {
		log.Println("url   ", url)
		log.Println("token ", t)
		log.Println("query ", q)
		log.Println("dbname", dbname)
		log.Println("dbid  ", dbid)
		log.Println("database", database)
		log.Println("dbtype  ", dbtype)
	}
	data := run(url, t, dbid, database, q, idx, limit, verbose)
	d, e := json.Marshal(data)
	if e == nil {
		fmt.Println(string(d))
	} else {
		log.Fatal(e)
	}
}
