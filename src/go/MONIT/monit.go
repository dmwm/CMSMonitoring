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
)

type Record struct {
	Record map[string]interface{}
}

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

func queryIDB(base, dbid, dbname, query string, headers [][]string, verbose int) []Record {
	rurl := fmt.Sprintf("%s/api/datasources/proxy/%s/query?db=%s&q=%s", base, dbid, dbname, query)
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
	var data []Record
	defer resp.Body.Close()
	// Deserialize the response into a map.
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	return data
}

func queryES(base, dbid, dbname, query string, headers [][]string, verbose int) []Record {
	// Method to query ES DB
	// https://www.elastic.co/guide/en/elasticsearch/reference/5.5/search-multi-search.html
	rurl := fmt.Sprintf("%s/api/datasources/proxy/%s/_msearch", base, dbid)
	q := fmt.Sprintf("{\"search_type\": \"query_then_fetch\", \"index\": [\"%s_*\"], \"ignore_unavailable\": true}\n%s\n", dbname, query)
	if verbose > 0 {
		log.Println(rurl, q)
	}
	/*
		send: 'GET /api/datasources/proxy/7617/_msearch HTTP/1.1\r\nHost: monit-grafana.cern.ch\r\nConnection: keep-alive\r\nAccept-Encoding: gzip, deflate\r\nAccept: application/json\r\nUser-Agent: python-requests/2.23.0\r\nContent-type: application/x-ndjson\r\nAuthorization: Bearer eyJrIjoiZWRnWXc1bUZWS0kwbWExN011TGNTN2I2S1JpZFFtTWYiLCJuIjoiY21zLXNzYiIsImlkIjoxMX0=\r\nContent-Length: 181\r\n\r\n{"search_type": "query_then_fetch", "index": ["monit_prod_wmagent_*"], "ignore_unavailable": true}\n{"query": {"match": {"data.payload.status": "Available"}}, "from": 0, "size": 10}\n'
	*/
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
	req.Header.Add("Accept", "application/json")
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
	var data []Record
	// Deserialize the response into a map.
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	return data
}

func queryURL(rurl string, headers [][]string, verbose int) []Record {
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
	req.Header.Add("Accept", "application/json")
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
	var data []Record
	// Deserialize the response into a map.
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	return data
}

func run(rurl, token, dbid, dbname, query string, idx, limit, verbose int) []Record {
	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", token)
	h := []string{"Authorization", bearer}
	headers = append(headers, h)
	h = []string{"Connection", "keep-alive"}
	headers = append(headers, h)
	h = []string{"Accept-Encoding", "gzip, deflate"}
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
	var dbid string
	flag.StringVar(&dbid, "dbid", "", "MONIT db identified")
	var dbname string
	flag.StringVar(&dbname, "dbname", "", "MONIT dbname")
	var query string
	flag.StringVar(&query, "query", "", "query string or query json file")
	var idx int
	flag.IntVar(&idx, "idx", 0, "verbosity level")
	var limit int
	flag.IntVar(&limit, "limit", 0, "verbosity level")
	flag.Parse()

	if verbose > 0 {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}
	t := read(token)
	q := read(query)
	if dbid == "" && url == defaultUrl {
		log.Fatalf("Please provide valid dbid")
	}
	if dbname == "" && url == defaultUrl {
		log.Fatalf("Please provide valid dbname")
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
	}
	run(url, t, dbid, dbname, q, idx, limit, verbose)
}
