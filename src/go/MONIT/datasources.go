package main

// File       : datasources.go
// Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
// Created    : Mon Oct  5 20:03:48 EDT 2020
// Description: fetch datasources info from CERN MONIT

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

// MonitRecord represents record we write out
type MonitRecord struct {
	Id       int    `json:"id"`
	Type     string `json:"type"`
	Database string `json:"database"`
}

// DSRecord represent a MONIT record
type DSRecord struct {
	Name string `json:"name"`
	MonitRecord
}

// helper function to get token
func getToken(r string) string {
	if _, err := os.Stat(r); err == nil {
		b, e := ioutil.ReadFile(r)
		if e != nil {
			log.Fatalf("Unable to read data from file: %s, error: %s", r, e)
		}
		return strings.Replace(string(b), "\n", "", -1)
	}
	return r
}

// helper function to get datasources
func datasources(rurl, t string, verbose int) map[string]MonitRecord {
	uri := fmt.Sprintf("%s/api/datasources", rurl)
	req, err := http.NewRequest("GET", uri, nil)
	token := getToken(t)
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
	orec := make(map[string]MonitRecord)
	for _, r := range records {
		o := MonitRecord{Id: r.Id, Type: r.Type, Database: r.Database}
		orec[r.Name] = o
	}
	return orec
}

func main() {
	var verbose int
	flag.IntVar(&verbose, "verbose", 0, "verbosity level")
	var rurl string
	flag.StringVar(&rurl, "url", "https://monit-grafana.cern.ch", "CERN MONIT URL")
	var token string
	flag.StringVar(&token, "token", "", "Token or token file")
	flag.Parse()
	rec := datasources(rurl, token, verbose)
	data, err := json.MarshalIndent(rec, "", "\t")
	if err != nil {
		log.Fatalf("unable to marshal data, error %v\n", err)
	}
	fmt.Println(string(data))
}
