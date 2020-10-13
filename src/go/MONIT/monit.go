package main

// File       : monit.go
// Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
// Created    : Fri, 03 Apr 2020 10:48:20 GMT
// Description: client for CERN MONIT infrastructure

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-stomp/stomp"
)

// Record represents MONIT return record {"response":...}
type Record map[string]interface{}

// DSRecord represents record we write out
type DSRecord struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Database string `json:"database"`
}

// DataSources provides list of MONIT datasources
var DataSources []DSRecord

// StompConfig represents stomp configuration. The configuration should contains
// either Login/Password or Key/Cert pair
type StompConfig struct {
	Producer string `json:"producer"`       // stomp producer name
	URI      string `json:"host_and_ports"` // stomp URI host:port
	Login    string `json:"username"`       // stomp user name (optional)
	Password string `json:"password"`       // stomp password (optional)
	Key      string `json:"key"`            // stomp user key (optional)
	Cert     string `json:"cert"`           // stomp user certification (optional)
	Topic    string `json:"topic"`          // stomp topic path
}

// helper function to provide string representation of Stomp Config
func (c *StompConfig) String() string {
	return fmt.Sprintf("<StompConfig uri=%s producer=%s topic=%s>", c.URI, c.Producer, c.Topic)
}

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
func datasources(rurl, t string, verbose int) []DSRecord {
	uri := fmt.Sprintf("%s/api/datasources", rurl)
	req, err := http.NewRequest("GET", uri, nil)
	token := read(t)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-type", "application/x-ndjson")
	req.Header.Set("Accept", "application/json")
	if verbose > 0 {
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
	if verbose > 0 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("response:", string(dump))
		}
	}
	var records []DSRecord
	defer resp.Body.Close()
	// Deserialize the response into a map.
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	return records
}

// helper function to find MONIT datasource id
func findDataSource(pat string) (int, string, string) {
	for _, r := range DataSources {
		db := r.Database
		if strings.Contains(db, "YYYY-MM-DD") {
			db = strings.Replace(db, "YYYY-MM-DD", "", -1)
		}
		did := r.Id
		dtype := r.Type
		dbname := r.Name
		if db == pat || dbname == pat || (strings.Contains(pat, "*") && strings.Contains(db, pat)) {
			return did, db, dtype
		}
	}
	return 0, "", ""
}

// helper function to send data to MONIT via StompAMQ and TLS
func sendDataToStompTLS(config StompConfig, data []byte, verbose int) {
	contentType := "application/json"
	x509cert, err := tls.LoadX509KeyPair(config.Cert, config.Key)
	if err != nil {
		log.Fatalf("failed to parse user certificates: %v", err)
	}
	certs := []tls.Certificate{x509cert}
	conf := &tls.Config{Certificates: certs, InsecureSkipVerify: true}
	conn, err := tls.Dial("tcp", config.URI, conf)
	defer conn.Close()
	if err != nil {
		log.Fatalf("Unable to dial to %s, error %v", config.URI, err)
	}
	if verbose > 0 {
		log.Printf("connected to StompAMQ server %s", config.URI)
	}
	stompConn, err := stomp.Connect(conn)
	if err != nil {
		log.Fatalf("Unable to connect to %s, error %v", config.URI, err)
	}
	defer stompConn.Disconnect()
	if stompConn != nil {
		err = stompConn.Send(config.Topic, contentType, data)
		if err != nil {
			log.Printf("unable to send data to %s, error %v", config.Topic, err)
		}
		if verbose > 0 {
			log.Println("Send data to MONIT", string(data))
		}
	}
}

// helper function to send data to MONIT via StompAMQ (plain or TLS access is supported)
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
			log.Println("Send data to MONIT", err, string(data))
		}
	}
}

// helper function to query InfluxDB
func queryIDB(base string, dbid int, dbname, query string, headers [][]string, verbose int) Record {
	rurl := fmt.Sprintf("%s/api/datasources/proxy/%d/query?db=%s&q=%s", base, dbid, dbname, url.QueryEscape(query))
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

// helper function to query ElasticSearch
func queryES(base string, dbid int, dbname, query, esapi string, headers [][]string, verbose int) Record {
	// Method to query ES DB
	// https://www.elastic.co/guide/en/elasticsearch/reference/5.5/search-multi-search.html
	rurl := fmt.Sprintf("%s/api/datasources/proxy/%d/_msearch", base, dbid)
	if esapi != "_msearch" && esapi != "" {
		if strings.HasPrefix(esapi, "/") {
			rurl = fmt.Sprintf("%s/api/datasources/proxy/%d%s", base, dbid, esapi)
		} else {
			rurl = fmt.Sprintf("%s/api/datasources/proxy/%d/%s", base, dbid, esapi)
		}
	}
	dbname = strings.Replace(dbname, "[", "", -1)
	dbname = strings.Replace(dbname, "]", "", -1)
	dbname = strings.Replace(dbname, "_*", "", -1)
	dbname = strings.Replace(dbname, "*", "", -1)
	q := fmt.Sprintf("{\"search_type\": \"query_then_fetch\", \"index\": [\"%s*\"], \"ignore_unavailable\": true}\n%s\n", dbname, query)
	if esapi != "_msearch" {
		// if we provided with concrete API to use, e.g. _count
		// we don't need to wrap the query
		q = query
	}
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
	if esapi != "_msearch" {
		// for concrete API use application/json content type
		req.Header.Add("Content-type", "application/json")
	} else {
		// for _msearch (default) API we use specific json content type
		req.Header.Add("Content-type", "application/x-ndjson")
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

// helper function to query given URL
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

// helper function to run ES/InfluxDB or MONIT query
func run(rurl, token string, dbid int, dbname, query, esapi string, idx, limit, verbose int) Record {
	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", token)
	h := []string{"Authorization", bearer}
	headers = append(headers, h)
	h = []string{"Accept", "application/json"}
	headers = append(headers, h)

	q := strings.ToLower(query)
	if strings.Contains(q, "elasticsearch") {
		return queryES(rurl, dbid, dbname, query, esapi, headers, verbose)
	} else if strings.Contains(q, "query") {
		return queryES(rurl, dbid, dbname, query, esapi, headers, verbose)
	} else if strings.Contains(q, "select") {
		return queryIDB(rurl, dbid, dbname, query, headers, verbose)
	} else if strings.Contains(q, "show") {
		return queryIDB(rurl, dbid, dbname, query, headers, verbose)
	}
	// perform query for given url
	if verbose > 0 {
		log.Println("Run query for given URL:", rurl)
	}
	return queryURL(rurl, headers, verbose)
}

// helper function to read single JSON or list of JSON records from
// given file name
func readRecords(fname string, verbose int) []Record {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Fatalf("Unable to read, file: %s, error: %v\n", fname, err)
	}
	var out []Record
	// let's probe if our data is a single JSON record
	var rec Record
	err = json.Unmarshal(data, &rec)
	if err == nil {
		out = append(out, rec)
	} else {
		// let's probe if our data is a list of JSON records
		var records []Record
		err = json.Unmarshal(data, &records)
		if err == nil {
			out = records
		}
	}
	if verbose > 0 {
		for _, r := range out {
			raw, err := json.Marshal(r)
			if err == nil {
				log.Println(string(raw))
			} else {
				log.Printf("unable to marshal input data, %v, error %v", r, err)
			}
		}
	}
	return out
}

// helper function to parse stomp credentials, return StompConfig structure
func parseStompConfig(creds string) StompConfig {
	raw, err := ioutil.ReadFile(creds)
	if err != nil {
		log.Fatalf("Unable to read, file: %s, error: %v\n", creds, err)
	}
	var config StompConfig
	err = json.Unmarshal(raw, &config)
	if err != nil {
		log.Fatalf("Unable to parse, file: %s, error: %v\n", creds, err)
	}
	return config
}

// helper function to inject given data record into MONIT with given credentials file
func injectData(config StompConfig, data []byte, verbose int) {
	if config.Key != "" && config.Cert != "" {
		if verbose > 0 {
			log.Println("Use TLS method to conenct to Stomp endpoint")
		}
		sendDataToStompTLS(config, data, verbose)
	} else if config.Login != "" && config.Password != "" {
		if verbose > 0 {
			log.Println("Use Login/Password method to connect to Stomp endpoint")
		}
		sendDataToStomp(config, data, verbose)
	} else {
		log.Fatalf("Provided configuration does not contain user credentials")
	}
}

// helper function to inject list of records into MONIT infrastructure with
// given credentials file name
func injectRecords(config StompConfig, records []Record, verbose int, inject bool) {
	if !inject {
		if verbose > 0 {
			for _, r := range records {
				raw, err := json.Marshal(r)
				if err == nil {
					log.Println(string(raw))
				}
			}
		}
		return
	}
	if config.Topic == "" {
		if verbose > 0 {
			log.Printf("Empty StompConfig %s\n", config.String())
			return
		}
	}
	for _, r := range records {
		raw, err := json.Marshal(r)
		if err == nil {
			injectData(config, raw, verbose)
		}
	}
}

// helper function to group ES Index
func groupESIndex(name string) string {
	s := strings.Replace(name, "monit_prod_", "", 1)
	s = strings.Split(s, "_raw_")[0]
	return s
}

// helper funtion to parse stats meta-data
func parseStats(data map[string]interface{}, verbose int) []Record {
	indices := data["indices"].(map[string]interface{})
	cmsIndexes := []string{}
	for _, d := range DataSources {
		db := d.Database
		arr := strings.Split(db, "]")
		for _, idx := range arr {
			idx = strings.Replace(idx, "[", "", -1)
			idx = strings.Replace(idx, "*", "", -1)
			idx = strings.Trim(idx, " ")
			if idx != "" {
				cmsIndexes = append(cmsIndexes, idx)
			}
		}
	}
	if verbose > 0 {
		log.Println("CMS indexes", cmsIndexes)
	}
	var out []Record
	for k, v := range indices {
		for _, idx := range cmsIndexes {
			if strings.HasPrefix(k, idx) {
				r := v.(map[string]interface{})
				t := r["total"].(map[string]interface{})
				s := t["store"].(map[string]interface{})
				size := s["size_in_bytes"].(float64)
				fmt.Printf("%s %d\n", k, int64(size))
				rec := make(Record)
				rec["name"] = k
				rec["size"] = int64(size)
				rec["type"] = "elasticsearch"
				rec["location"] = ""
				rec["group"] = groupESIndex(k)
				out = append(out, rec)
			}
		}
	}
	return out
}

// helper function to read given file with hdfs paths and dump their sizes
func hdfsDump(fname string, verbose int) []Record {
	var out []Record
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Fatalf("Unable to read, file: %s, error: %v\n", fname, err)
	}
	var rec Record
	err = json.Unmarshal(data, &rec)
	if err != nil {
		log.Printf("Unable to parse, data: %s, error: %v\n", string(data), err)
		return out
	}
	for k, p := range rec {
		path := p.(string)
		size, err := hdfsSize(path, verbose)
		if err == nil {
			rec := make(Record)
			rec["name"] = k
			rec["size"] = int64(size)
			rec["type"] = "hdfs"
			rec["location"] = path
			rec["group"] = ""
			fmt.Printf("%s %d\n", path, int64(size))
			out = append(out, rec)
		}
	}
	return out
}

// helper function to print out total size of given HDFS path
func hdfsSize(path string, verbose int) (float64, error) {
	out, err := exec.Command("hadoop", "fs", "-du", "-s", "-h", path).Output()
	if err != nil {
		log.Println("Fail to execute hadoop fs -du -s -h", path, err)
		return -1, err
	}
	// hadoop fs -du -h -s /cms/wmarchive
	// 3.0 T  9.1 T  /cms/wmarchive
	arr := strings.Split(string(out), " ")
	size, err := strconv.ParseFloat(arr[0], 10)
	if err != nil {
		log.Printf("Fail to parse hadoop output, %s, %v\n", out, err)
		return -1, err
	}
	metric := strings.Trim(arr[1], " ")
	switch metric {
	case "K":
		size = size * 1024
	case "M":
		size = size * 1024 * 1024
	case "G":
		size = size * 1024 * 1024 * 1024
	case "T":
		size = size * 1024 * 1024 * 1024
	case "P":
		size = size * 1024 * 1024 * 1024 * 1024
	case "E":
		size = size * 1024 * 1024 * 1024 * 1024 * 1024
	case "Z":
		size = size * 1024 * 1024 * 1024 * 1024 * 1024 * 1024
	case "Y":
		size = size * 1024 * 1024 * 1024 * 1024 * 1024 * 1024 * 1024
	default:
		if verbose > 0 {
			log.Println("Unable to find proper size", out)
		}
	}
	return size, nil
}

// helper function to parse comma separated tags string
func parseTags(itags string) []string {
	var tags []string
	for _, tag := range strings.Split(itags, ",") {
		tags = append(tags, strings.Trim(tag, " "))
	}
	return tags
}

// helper function to parse given time-range separated by '-'
func parseTimes(trange string) []int64 {
	var times []int64
	for _, v := range strings.Split(trange, "-") {
		v = strings.Trim(v, " ")
		var t int64
		if v == "now" {
			t = time.Now().Unix()
		} else if len(v) == 10 { // unix since epoch
			value, err := strconv.Atoi(v)
			if err != nil {
				log.Fatalf("Unable to parse given time value: %v\n", v)
			}
			t = int64(value)
		} else {
			var value string
			var offset int64
			if strings.HasSuffix(v, "s") || strings.HasSuffix(v, "sec") {
				value = strings.Split(v, "s")[0]
				offset = 1
			} else if strings.HasSuffix(v, "m") || strings.HasSuffix(v, "min") {
				value = strings.Split(v, "m")[0]
				offset = 60
			} else if strings.HasSuffix(v, "h") || strings.HasSuffix(v, "hour") {
				value = strings.Split(v, "h")[0]
				offset = 60 * 60
			} else if strings.HasSuffix(v, "d") || strings.HasSuffix(v, "day") || strings.HasSuffix(v, "days") {
				value = strings.Split(v, "d")[0]
				offset = 60 * 60 * 24
			} else {
				log.Fatalf("Unable to parse given time value: %v\n", v)
			}
			if v, e := strconv.Atoi(value); e == nil {
				t = time.Now().Unix() + int64(v)*offset
			}
		}
		if t == 0 {
			log.Fatalf("Unable to parse given time value: %v\n", v)
		}
		times = append(times, t*1000) // times should be in milliseconds of Unix since epoch
	}
	return times
}

// helper function to find dashboard info
func findDashboard(base, token, itags string, verbose int) []map[string]interface{} {
	tags := parseTags(itags)
	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", token)
	h := []string{"Authorization", bearer}
	headers = append(headers, h)
	h = []string{"Accept", "application/json"}
	headers = append(headers, h)
	// example: /api/search?query=Production%20Overview&starred=true&tag=prod
	v := url.Values{}
	for _, tag := range tags {
		v.Add("tag", strings.Trim(tag, " "))
	}
	rurl := fmt.Sprintf("%s/api/search?%s", base, v.Encode())
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
	var data []map[string]interface{}
	// Deserialize the response into a map.
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	return data

}

// helper function to add annotation
func addAnnotation(base, token string, data []byte, verbose int) {
	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", token)
	h := []string{"Authorization", bearer}
	headers = append(headers, h)
	h = []string{"Content-Type", "application/json"}
	headers = append(headers, h)

	rurl := fmt.Sprintf("%s/api/annotations", base)
	if verbose > 0 {
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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Unable to read JSON Data POST API, error: %v\n", err)
	}

	defer resp.Body.Close()

	log.Println("response Status:", resp.Status)
	log.Println("response Headers:", resp.Header)
	log.Println("response Body:", string(body))
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
	var esapi string
	flag.StringVar(&esapi, "esapi", "_msearch", "MONIT ES API end-point")
	var annotation string
	flag.StringVar(&annotation, "annotation", "", "annotation json file")
	var tags string
	flag.StringVar(&tags, "tags", "", "comma separated dashboard tags")
	var trange string
	flag.StringVar(&trange, "trange", "", "dash separated time range, supported formats: seconds since epoch, <YYYYMMDD>, <now>, or Xs, Xm, Xh, Xd (where X is a value and suffixes s,m,h,d refer to seconds, minute, hour, day)")
	var query string
	flag.StringVar(&query, "query", "", "query string or query json file")
	var input string
	flag.StringVar(&input, "input", "", "specify input file (should contain either single JSON dict or list of dicts)")
	var inject bool
	flag.BoolVar(&inject, "inject", false, "inject data to MONIT")
	var creds string
	flag.StringVar(&creds, "creds", "", "json document with MONIT credentials")
	var hdfs string
	flag.StringVar(&hdfs, "hdfs", "", "json document with HDFS paths, see doc/hdfs/hdfs.json")
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
		fmt.Println("   # read data from input file")
		fmt.Println("   monit -input=doc.json -verbose 1")
		fmt.Println("")
		fmt.Println("   # inject data from given file into MONIT")
		fmt.Println("   monit -input=doc.json -creds=creds.json -verbose 1")
		fmt.Println("")
		fmt.Println("   # look-up data from MONIT")
		fmt.Println("   monit -token token -query=query.json -dbname=monit_prod_wmagent")
		fmt.Println("")
		fmt.Println("   # look-up data from ES MONIT using specific ES API end-point")
		fmt.Println("   monit -token token -query=query.json -dbname=monit_prod_wmagent -esapi=_count")
		fmt.Println("")
		fmt.Println("   # provide stats for all cms ES indicies")
		fmt.Println("   monit -token token -query=\"stats\"")
		fmt.Println("")
		fmt.Println("   # provide stats for all cms ES and hdfs data and inject them to MONIT")
		fmt.Println("   monit -token token -query=\"stats\" -hdfs hdfs.json -creds=creds.json -inject")
		fmt.Println("")
		fmt.Println("   # look-up all available intervention in MONIT InfluxDB for last 2 hours")
		fmt.Println("   monit -token token -query=\"select * from outages where time > now() - 2h limit 1\" -dbname=monit_production_ssb_otgs -dbid=9474")
		fmt.Println("")
		fmt.Println("   # provide annotation to MONIT")
		fmt.Println("   monit -token admin.token -annotation=a.json")
		fmt.Println("   monit -token admin.token -annotation=\"some message\" -tags=cmsweb,das -trange=now-10m")
		fmt.Println("")
		fmt.Println("   # find dashboard information for given tags")
		fmt.Println("   monit -token token -tags=\"cmsweb,das\"")
		fmt.Println("")
		fmt.Println("   # look-up all available datasources in MONIT")
		fmt.Println("   monit -datasources -token token")
	}
	flag.Parse()

	if verbose > 0 {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}
	var e error
	DataSources = datasources(defaultUrl, token, verbose)
	if listDataSources {
		data, err := json.MarshalIndent(DataSources, "", "\t")
		if err == nil {
			fmt.Println(string(data))
			os.Exit(0)
		}
	}
	// parse Stomp Configuration
	var stompConfig StompConfig
	if creds != "" {
		stompConfig = parseStompConfig(creds)
		if verbose > 0 {
			log.Println("StompConfig:", stompConfig.String())
		}
	}
	if input != "" {
		records := readRecords(input, verbose)
		injectRecords(stompConfig, records, verbose, inject)
		return
	}
	t := read(token)
	if annotation != "" {
		var data []byte
		if _, err := os.Stat(annotation); err == nil {
			data, err = ioutil.ReadFile(annotation)
			if err != nil {
				log.Fatalf("Unable to read, file: %s, error: %v\n", annotation, err)
			}
			addAnnotation(url, t, data, verbose)
		} else {
			// find dashboards info
			dashboards := findDashboard(url, t, tags, verbose)
			timeRanges := parseTimes(trange)
			if verbose > 0 {
				log.Println("timeRanges", timeRanges)
			}
			for _, r := range dashboards {
				// create new annotation data
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
				if verbose > 0 {
					log.Printf("Add annotation: %+v", rec)
				}
				addAnnotation(url, t, data, verbose)
			}
		}
		return
	}
	q := read(query)
	if tags != "" {
		data := findDashboard(url, t, tags, verbose)
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			fmt.Println("error:", err)
		}
		fmt.Print(string(b))
		return
	}
	dbID := dbid
	var database, dbtype string
	if strings.Contains(q, "stats") {
		// per our discussion with MONIT team
		// https://cern.service-now.com/service-portal?id=ticket&table=u_request_fulfillment&n=RQF1647972
		// we only need to use some dbid for _stats ES API, here I use
		// 9573, the id of monit_prod_cms-es-size_raw_elasticsearch collection
		dbid = 9573
		url = fmt.Sprintf("%s/api/datasources/proxy/%d/_stats/store", url, dbid)
	} else {
		if dbname == "" && url == defaultUrl {
			log.Fatalf("Please provide valid dbname")
		}
		dbid, database, dbtype = findDataSource(dbname)
		if dbid == 0 {
			log.Fatalf("No valid dbid found for %s", dbname)
		}
	}
	// if user gave explicit dbid we should use it
	if dbID > 0 {
		dbid = dbID
	}
	if token == "" {
		log.Fatalf("Please provide valid token")
	}
	if verbose > 1 {
		log.Println("url     ", url)
		log.Println("token   ", t)
		log.Println("query   ", q)
		log.Println("dbname  ", dbname)
		log.Println("dbid    ", dbid)
		log.Println("database", database)
		log.Println("dbtype  ", dbtype)
		log.Println("esapi   ", esapi)
		log.Println("hdfs    ", hdfs)
	}
	data := run(url, t, dbid, database, q, esapi, idx, limit, verbose)
	if strings.Contains(q, "stats") {
		records := parseStats(data, verbose)
		injectRecords(stompConfig, records, verbose, inject)
		// obtain HDFS records if requested
		if hdfs != "" {
			records = hdfsDump(hdfs, verbose)
			injectRecords(stompConfig, records, verbose, inject)
		}
		return
	}
	d, e := json.Marshal(data)
	if e == nil {
		fmt.Println(string(d))
	} else {
		log.Fatal(e)
	}
}
