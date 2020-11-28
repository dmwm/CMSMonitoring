package main

// File       : rumble_go.go
// Author     : Ceyhun Uzunoglu <cuzunogl AT gmail dot com>
// Description: client for rumble server

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
)

// helper function to either read file content or return given string. Second parameter is for isHdfs.
func read(r string) (string, bool) {
	// if query is in hdfs, use query-path parameter in Rumble
	// if not, read query from file or give query string itself
	if strings.HasPrefix(r, "hdfs://") {
		return r, true
	} else if _, err := os.Stat(r); err == nil {
		b, e := ioutil.ReadFile(r)
		if e != nil {
			log.Fatalf("Unable to read data from file: %s, error: %s", r, e)
		}
		return strings.Replace(string(b), "\n", "", -1), false
	}
	return r, false
}

func write(resp *http.Response, output string) {
	// if output is not given, print response to stdout
	if output == "" {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("RESPONSE: \n" + string(data))
	} else {
		outFile, err := os.Create(output)
		if err != nil {
			log.Fatal("ERROR: unable to create file:", err)
		}
		defer outFile.Close()
		io.Copy(outFile, resp.Body)
		log.Println("Response is written to file:", output)
	}
}

// helper function to send Rumble request
func sendRequest(server string, query string, output string, isHdfs bool, verbose int) {
	var req *http.Request
	var err error
	rurl := server
	if verbose > 0 {
		log.Println("Server:", rurl)
		log.Println("Query:\n" + query)
	}

	if !isHdfs {
		// if JSONiq query is not in hdfs file
		req, err = http.NewRequest("POST", rurl, strings.NewReader(query))
	} else {
		rurl := fmt.Sprintf("%s&overwrite=yes&query-path=%s", server, query)
		req, err = http.NewRequest("POST", rurl, nil)
	}

	if err != nil {
		log.Fatalf("Unable to make request to %s, error: %s", rurl, err)
	}

	err = nil // delete
	if verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("Request:\n" + string(dump))
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Unable to get response from %s, error: %s", rurl, err)
	}
	defer resp.Body.Close()

	log.Println("Response Status:", resp.Status)
	if verbose > 0 {
		log.Println("Response Headers:", resp.Header)
	}
	if resp.StatusCode == http.StatusOK {
		write(resp, output)
	}
}

func main() {
	defaultUrl := "https://cms-monitoring.cern.ch/"
	uri := defaultUrl + "jsoniq?materialization-cap=-1"
	var verbose int
	flag.IntVar(&verbose, "verbose", 0, "verbosity level")
	var server string
	flag.StringVar(&server, "server", uri, "Rumble server URL")
	var query string
	flag.StringVar(&query, "query", "", "query string or query JSONiq file")
	var output string
	flag.StringVar(&output, "output", "", "specify output file path, if not defined, print to stdout")
	flag.Usage = func() {
		fmt.Println("Usage: rumble_go [options]")
		flag.PrintDefaults()
		fmt.Println("Examples:")
		fmt.Println("   # simple rumble query, prints to stdout")
		fmt.Println("   rumble_go -query=\"1+1\"")
		fmt.Println("")
		fmt.Println("   # write result to file")
		fmt.Println("   rumble_go -query=\"1+1\" -output output.txt")
		fmt.Println("")
		fmt.Println("   # get query from hdfs or file and write result to file")
		fmt.Println("   rumble_go -query=\"./exmaple.jq\" -output output.txt")
		fmt.Println("   rumble_go -query=\"hdfs://anlaytix/rumble.jq\" -output output.txt")
		fmt.Println("")
	}
	flag.Parse()
	if verbose > 0 {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}
	q, isHdfs := read(query)
	sendRequest(server, q, output, isHdfs, verbose)
}
