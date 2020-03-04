package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// example of result json for query action
// {"status":"success","data":{"resultType":"matrix","result":[{"metric":Record, "values":[[int64,int]]]}}
// example of result json for query export
// {"metric":Record, "values":[int,int,...], "timestamps":[int64,int64,...]}

// QueryResult
type QueryResult struct {
	Status string     `json:"status"`
	Data   DataRecord `json:"data"`
}

// DataRecord
type DataRecord struct {
	ResultType string        `json:"resultType"`
	Result     []QueryRecord `json:"result"`
}

// QueryRecord represents VictoriaMetrics query record
type QueryRecord struct {
	Metric Record          `json:"metric"` // VM record metric
	Values [][]interface{} `json:"values"` // VM values (number of records)
}

// ExportRecord represents VictoriaMetrics export record
type ExportRecord struct {
	Metric     Record  `json:"metric"`     // VM record metric
	Values     []int64 `json:"values"`     // VM values (number of records)
	TimeStamps []int64 `json:"timestamps"` // VM timestamps
}

// Record represents DBS record
type Record struct {
	Metric      string `json:"__name__"`
	Dataset     string `json:"dataset"`
	DatasetType string `json:"dataset_type"`
	TimeStamps  []int64
}

// tFormat helper function to convert given time into Unix timestamp
func tFormat(ts int64) string {
	// YYYYMMDD, always use 2006 as year 01 for month and 02 for date since it is predefined int Go parser
	const layout = "20060102"
	t := time.Unix(ts, 0)
	return t.In(time.UTC).Format(layout)
}

func (r *Record) String() string {
	var s string
	for _, t := range r.TimeStamps {
		if s == "" {
			s = fmt.Sprintf("%s %s %s", time.Unix(t, 0), r.Dataset, r.DatasetType)
		} else {
			s = fmt.Sprintf("%s\n%s %s %s", s, t, time.Unix(t, 0), r.Dataset, r.DatasetType)
		}
	}
	return s
}

func parseQueryResults(r io.Reader) []Record {
	var records []Record
	var qResult QueryResult
	data, err := ioutil.ReadAll(r)
	if err != nil {
		log.Println(err)
	}
	err = json.Unmarshal(data, &qResult)
	if err != nil {
		log.Println(err)
		log.Println(string(data))
	}
	if qResult.Status == "success" {
		for _, r := range qResult.Data.Result {
			var tstamps []int64
			for _, vals := range r.Values {
				// each value in vals list is [tstamp, "number"]
				tstamps = append(tstamps, int64(vals[0].(float64)))
			}
			rec := r.Metric
			rec.TimeStamps = tstamps
			records = append(records, rec)
		}
	}
	return records
}

func parseExportResults(r io.Reader) []Record {
	var eRecord ExportRecord
	var records []Record
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		data := scanner.Text()
		err := json.Unmarshal([]byte(data), &eRecord)
		if err != nil {
			log.Println(err)
			log.Println(data)
		} else {
			rec := eRecord.Metric
			var tstamps []int64
			for _, t := range eRecord.TimeStamps {
				tstamps = append(tstamps, int64(t/1000))
			}
			rec.TimeStamps = tstamps
			records = append(records, rec)
		}
	}
	return records
}

// helper function to fetch data from VM
func fetch(action, rurl, dtype string, start, end int64, verbose bool) []Record {
	client := http.Client{}
	if action == "query" {
		rurl = fmt.Sprintf("%s?query=cms.dbs&dataset_type=%s&start=%d", rurl, dtype, start)
		if end != 0 {
			rurl = fmt.Sprintf("%s&end=%d", rurl, end)
		}
	} else if action == "export" {
		query := fmt.Sprintf("{__name__=\"cms.dbs\",dataset_type=\"%s\"}", dtype)
		rurl = fmt.Sprintf("%s?match=%s", rurl, url.QueryEscape(query))
	}
	if verbose {
		fmt.Println(rurl)
	}
	req, err := http.NewRequest("GET", rurl, nil)
	if err != nil {
		log.Println(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	if action == "query" {
		return parseQueryResults(resp.Body)
	} else if action == "export" {
		return parseExportResults(resp.Body)
	}
	return []Record{}
}

func main() {
	var rurl string
	flag.StringVar(&rurl, "rurl", "", "DAS monitoring url")
	var action string
	flag.StringVar(&action, "action", "query", "Action: query or export")
	var dataset string
	flag.StringVar(&dataset, "dataset", "", "DAS dataset to look at")
	var datasetType string
	flag.StringVar(&datasetType, "datasetType", "VALID", "DAS datasetType to use")
	var start int64
	flag.Int64Var(&start, "start", 0, "timestamp since you want to start fetching the data")
	var end int64
	flag.Int64Var(&end, "end", 0, "timestamp since you want to end fetching the data")
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "verbose mode")
	flag.Parse()
	if rurl == "" {
		rurl = "http://cms-monitoring.cern.ch:30428"
	}
	if action == "query" {
		rurl = fmt.Sprintf("%s/api/v1/query_range", rurl)
	} else if action == "export" {
		rurl = fmt.Sprintf("%s/api/v1/export", rurl)
	} else {
		log.Fatalf("Wrong action: %s", action)
	}
	// log time, filename, and line number
	log.SetFlags(log.Ltime | log.Lshortfile)
	for _, r := range fetch(action, rurl, datasetType, start, end, verbose) {
		if r.DatasetType == datasetType {
			if dataset == "" { // match all dataset
				fmt.Println(r.String())
			} else { // match pattern
				if strings.Contains(r.Dataset, dataset) {
					fmt.Println(r.String())
				}
			}
		}
	}
}
