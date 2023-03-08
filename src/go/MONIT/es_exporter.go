package main

// Author: Valentin Kuznetsov <vkuznet [AT] gmail {DOT} com>
// Example of cmsweb data-service exporter for prometheus.io

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	listeningAddress  = flag.String("port", ":18000", "port to expose metrics and web interface.")
	metricsEndpoint   = flag.String("endpoint", "/metrics", "Path under which to expose metrics.")
	dbname            = flag.String("dbname", "", "ES dbname to use")
	gte               = flag.String("gte", "now-15m", "gte condition to use")
	lte               = flag.String("lte", "now", "lte condition to use")
	token             = flag.String("token", "", "token or file with token")
	namespace         = flag.String("namespace", "http", "namespace for prometheus metrics")
	connectionTimeout = flag.Int("connectionTimeout", 3, "connection timeout for HTTP request")
	verbose           = flag.Int("verbose", 0, "verbosity level")
)

// ESResponse has the following structure
// {"took":2,"responses":[{"took":2,"timed_out":false,"_shards":{"total":33,"successful":33,"skipped":0,"failed":0},"hits":{"total":{"value":4362,"relation":"eq"},"max_score":null,"hits":[]},"status":200}]}

// Total structure of Hits
type Total struct {
	Value    int    `json:"value"`
	Relation string `json:"relation"`
}

// Hits structure of Response
type Hits struct {
	Total Total `json:"total"`
}

// Response structure of ESResponse
type Response struct {
	Took     int  `json:"tool"`
	Timedout bool `json:"timed_out"`
	Hits     Hits `json:"hits"`
	Status   int  `json:"status"`
}

// ESResponse structure
type ESResponse struct {
	Took      int        `json:"took"`
	Responses []Response `json:"responses"`
}

// DSRecordMaps represents data sources maps {ds1:{id: 1, ...}, ds2:{id:2, ...}}
type DSRecordMaps map[string]map[string]interface{}

// DSRecord represents record we write out
type DSRecord struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Database string `json:"database"`
}

// helper function to either read file content or return given string
func read(r string) string {
	if _, err := os.Stat(r); err == nil {
		b, e := os.ReadFile(r)
		if e != nil {
			log.Fatalf("Unable to read data from file: %s, error: %s", r, e)
		}
		return strings.Replace(string(b), "\n", "", -1)
	}
	return r
}

// HttpClient provides HTTP client
func HttpClient() *http.Client {
	//     return &http.Client{}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		IdleConnTimeout: time.Duration(1 * time.Second),
		DialContext: (&net.Dialer{
			Timeout: time.Duration(*connectionTimeout) * time.Second,
		}).DialContext,
		DisableKeepAlives: true,
	}
	timeout := time.Duration(*connectionTimeout) * time.Second
	return &http.Client{Transport: tr, Timeout: timeout}
}

func dbName(dbname string) string {
	dbname = strings.Replace(dbname, "[", "", -1)
	dbname = strings.Replace(dbname, "]", "", -1)
	dbname = strings.Replace(dbname, "_*", "", -1)
	dbname = strings.Replace(dbname, "*", "", -1)
	return dbname
}

// Exporter represents Prometheus exporter structure
type Exporter struct {
	URI            string
	DBName         string
	mutex          sync.Mutex
	scrapeFailures prometheus.Counter
	status         *prometheus.Desc
	total          *prometheus.Desc
}

func NewExporter(uri, dbname string) *Exporter {
	ns := fmt.Sprintf("es_%s", dbName(dbname))
	return &Exporter{
		URI:            uri,
		DBName:         dbname,
		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{}),
		status: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "", "status"),
			fmt.Sprintf("Current status of %s", dbName(dbname)),
			nil,
			nil),
		total: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "", "total"),
			fmt.Sprintf("Number of docs in %s", dbName(dbname)),
			nil,
			nil),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.status
	ch <- e.total
}

// Collect performs metrics collectio of exporter attributes
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()
	if err := e.collect(ch); err != nil {
		log.Printf("Error scraping: %s\n", err)
		e.scrapeFailures.Inc()
		e.scrapeFailures.Collect(ch)
	}
	return
}

// helper function which collects exporter attributes
func (e *Exporter) collect(ch chan<- prometheus.Metric) error {

	// adjust dbname
	dbname := dbName(e.DBName)

	// construct proper query to ES
	if *gte == "" {
		*gte = "now-15m"
	}
	if *lte == "" {
		*lte = "now"
	}
	query := fmt.Sprintf("{\"size\":0,\"track_total_hits\":true,\"query\":{\"bool\":{\"must\":[{\"range\":{\"metadata.timestamp\":{\"gte\":\"%s\",\"lt\":\"%s\"}}}]}}}", *gte, *lte)

	q := fmt.Sprintf("{\"search_type\": \"query_then_fetch\", \"index\": [\"%s*\"], \"ignore_unavailable\": true}\n%s\n", dbname, query)

	if *verbose > 0 {
		log.Println(e.URI, "\n[User Query]:\n"+q)
	}

	// construct HTTP request to ES
	req, err := http.NewRequest("GET", e.URI, strings.NewReader(q))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", read(*token)))
	req.Header.Set("Content-type", "application/x-ndjson")
	req.Header.Set("Accept", "application/json")

	if *verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("request: ", string(dump))
		}
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if *verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("[DEBUG] response:", string(dump))
		}
	}
	if err != nil || resp.StatusCode != 200 {
		ch <- prometheus.MustNewConstMetric(e.status, prometheus.CounterValue, float64(resp.StatusCode))
		ch <- prometheus.MustNewConstMetric(e.total, prometheus.CounterValue, 0)
		if *verbose > 0 {
			log.Printf("Faile to make HTTP request, url=%s, error=%v\n", e.URI, err)
		}
		return nil
	}
	// {"took":2,"responses":[{"took":2,"timed_out":false,"_shards":{"total":33,"successful":33,"skipped":0,"failed":0},"hits":{"total":{"value":4362,"relation":"eq"},"max_score":null,"hits":[]},"status":200}]}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if *verbose > 0 {
		log.Println("received data", string(data))
	}
	var rec ESResponse
	err = json.Unmarshal(data, &rec)
	if err == nil {
		if *verbose > 1 {
			log.Printf("record: %+v\n", rec)
		}
		for _, r := range rec.Responses {
			status := float64(r.Status)
			total := float64(r.Hits.Total.Value)
			ch <- prometheus.MustNewConstMetric(e.status, prometheus.CounterValue, status)
			ch <- prometheus.MustNewConstMetric(e.total, prometheus.CounterValue, total)
			return nil
		}
	} else {
		log.Fatal("fail to decode data", err)
	}
	return nil
}

// helper function to find MONIT datasource id and database name
func findDataSource(pat string, records []DSRecord) (int, string) {
	for _, r := range records {
		db := r.Database
		if strings.Contains(db, "YYYY-MM-DD") {
			db = strings.Replace(db, "YYYY-MM-DD", "", -1)
		}
		did := r.Id
		dbname := r.Name
		if db == pat || dbname == pat || (strings.Contains(pat, "*") && strings.Contains(db, pat)) {
			return did, db
		}
	}
	return 0, ""
}

// Read CMS Monitoring datasources from CMS Monitoring location
func readDS() []DSRecord {
	var records []DSRecord
	// read datasources from CMS Monitoring place
	dsurl := "https://raw.githubusercontent.com/dmwm/CMSMonitoring/master/static/datasources.json"
	req, err := http.NewRequest("GET", dsurl, nil)
	req.Header.Set("Content-type", "application/x-ndjson")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Unable to get response from %s, error: %s", dsurl, err)
	}
	dsRecordMaps := make(DSRecordMaps)
	defer resp.Body.Close()
	// Deserialize the response into a map.
	if err := json.NewDecoder(resp.Body).Decode(&dsRecordMaps); err != nil {
		log.Fatalf("Error happened while getting Grafana datasources! Please report this error to admins.")
	}
	// Convert map of data sources to DSRecord struct list
	for k, v := range dsRecordMaps {
		r := new(DSRecord)
		r.Type = fmt.Sprint(v["type"])
		r.Id = int(v["id"].(float64))
		r.Database = fmt.Sprint(v["database"])
		r.Name = fmt.Sprint(k)
		records = append(records, *r)
	}
	return records
}

// main function
func main() {
	flag.Parse()
	// log time, filename, and line number
	if *verbose > 0 {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}
	dbid, database := findDataSource(*dbname, readDS())
	uri := fmt.Sprintf("https://monit-grafana.cern.ch/api/datasources/proxy/%d/_msearch", dbid)

	exporter := NewExporter(uri, database)
	prometheus.MustRegister(exporter)

	log.Printf("Starting Server: %s\n", *listeningAddress)
	http.Handle(*metricsEndpoint, promhttp.Handler())
	log.Fatal(http.ListenAndServe(*listeningAddress, nil))
}
