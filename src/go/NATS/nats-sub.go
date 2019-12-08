// Copyright 2012-2019 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

// NOTE: Can test with demo servers.
// nats-sub -s demo.nats.io <subject>
// nats-sub -s demo.nats.io:4443 <subject> (TLS version)

func usage() {
	log.Printf("Usage: nats-sub [-s server] [-creds file] [-t] <subject>\n")
	flag.PrintDefaults()
}

func showUsageAndExit(exitcode int) {
	usage()
	os.Exit(exitcode)
}

// StringList implement sort for []string type
type StringList []string

// Len provides length of the []int type
func (s StringList) Len() int { return len(s) }

// Swap implements swap function for []int type
func (s StringList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Less implements less function for []int type
func (s StringList) Less(i, j int) bool { return s[i] < s[j] }

// inList helper function to find an element in a list
func inList(a string, list []string) bool {
	check := 0
	for _, b := range list {
		if b == a {
			check += 1
		}
	}
	if check != 0 {
		return true
	}
	return false
}

// print helper functions
func printMsg(m *nats.Msg, i int) {
	log.Printf("[#%d] [%s] '%s'", i, m.Subject, string(m.Data))
}
func printCMSMsg(m *nats.Msg, attributes []string, sep string) {
	msg := string(m.Data)
	var vals []string
	for _, v := range strings.Split(msg, sep) {
		arr := strings.Split(v, ":")
		if len(attributes) > 0 {
			if inList(arr[0], attributes) {
				vals = append(vals, arr[1])
			}
		} else { // show all attributes
			vals = append(vals, arr[1])
		}
	}
	log.Printf(strings.Join(vals, " "))
}

// decode NATS message and extract value of given key
func decodeMsg(m *nats.Msg, sep, key string) string {
	msg := string(m.Data)
	for _, v := range strings.Split(msg, sep) {
		arr := strings.Split(v, ":")
		if key == arr[0] {
			return arr[1]
		}
	}
	return ""
}

// helper function to send stats to pushgateway server
func sendToPushGateway(uri, job, inst string, statsDict map[string]int) {
	job = strings.Replace(job, ".", "_", -1)
	job = strings.Replace(job, ">", "metric", -1)
	url := fmt.Sprintf("%s/metrics/job/%s", uri, job)
	if inst != "" {
		url = fmt.Sprintf("%s/instance/%s", url, inst)
	}
	client := &http.Client{}
	for k, v := range statsDict {
		// message should end with '\n' when we send it to pushgateway
		msg := fmt.Sprintf("%s_%s %d\n", job, k, v)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(msg)))
		req.Header.Set("Content-Type", "application/octet-stream")
		if err == nil {
			go func() {
				resp, err := client.Do(req)
				defer resp.Body.Close()
				if err != nil {
					fmt.Println(url, msg, "response:", resp.Status)
					fmt.Println("response Headers:", resp.Header)
					body, _ := ioutil.ReadAll(resp.Body)
					fmt.Println("response Body:", string(body))
				}
			}()
		}
	}
}

// VMRecord represents VictoriaMetrics record
type VMRecord struct {
	Metric string            `json:"metric"`
	Value  int               `json:"value"`
	Tags   map[string]string `json:"tags"`
}

// helper function to convert message to VMRecord
func msg2VMRecord(msg, topic, sep string) VMRecord {
	var rec VMRecord
	metric := strings.Replace(topic, ".>", "", -1)
	metric = strings.Replace(metric, "*", "", -1)
	rec.Metric = metric
	rdict := make(map[string]string)
	for _, v := range strings.Split(msg, sep) {
		arr := strings.Split(v, ":")
		rdict[arr[0]] = arr[1]
	}
	rec.Tags = rdict
	rec.Value = 1
	return rec
}

// helper function to send stats to VictoriaMetrics server
func sendToVictoriaMetrics(vmUri string, m *nats.Msg, topic, sep string) {
	msg := string(m.Data)
	url := fmt.Sprintf("%s/api/put", vmUri)
	client := &http.Client{}
	rec := msg2VMRecord(msg, topic, sep)
	recBytes, err := json.Marshal(rec)
	if err != nil {
		log.Println("Unable to marshal VMRecord", rec, err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(recBytes))
	req.Header.Set("Content-Type", "application/json")
	if err == nil {
		go func() {
			resp, err := client.Do(req)
			defer resp.Body.Close()
			if err != nil {
				fmt.Println(url, msg, "response:", resp.Status)
				fmt.Println("response Headers:", resp.Header)
				body, _ := ioutil.ReadAll(resp.Body)
				fmt.Println("response Body:", string(body))
			}
		}()
	}
}

func main() {
	var urls = flag.String("s", "cms-nats.cern.ch", "The nats server URLs (separated by comma)")
	var creds = flag.String("creds", "", "User NSC credentials")
	var cmsAuth = flag.String("cmsAuth", "", "User cms auth file")
	var showTime = flag.Bool("t", false, "Display timestamps")
	var showHelp = flag.Bool("h", false, "Show help message")
	var attrs = flag.String("attrs", "", "comma separated list of attributes to show, e.g. campaign,dataset")
	var sep = flag.String("sep", "   ", "message attribute separator, default 3 spaces")
	var raw = flag.Bool("raw", false, "Show raw messages")
	var showStats = flag.Bool("showStats", false, "Show stats instead of messages")
	var statsBy = flag.String("statsBy", "", "Show stats by given attribute")
	var statsInterval = flag.Int("statsInterval", 10, "stats collect interval in seconds")
	var rootCAs = flag.String("rootCAs", "", "Comma separated list of CERN Root CAs files")
	var userKey = flag.String("userKey", "", "x509 user key file")
	var userCert = flag.String("userCert", "", "x509 user certificate")
	var gatewayUri = flag.String("gatewayUri", "", "URI of gateway server")
	var gatewayJob = flag.String("gatewayJob", "", "gateway job name")
	var gatewayInstance = flag.String("gatewayInstance", "", "gateway instance name")
	var vmUri = flag.String("vmUri", "", "VictoriaMetrics URI")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if *showHelp {
		showUsageAndExit(0)
	}

	args := flag.Args()
	if len(args) != 1 {
		showUsageAndExit(1)
	}

	// Connect Options.
	opts := []nats.Option{nats.Name("NATS Sample Subscriber")}
	opts = setupConnOptions(opts)

	// Use UserCredentials
	var auth string
	if *creds != "" {
		if _, err := os.Stat(*creds); err == nil {
			opts = append(opts, nats.UserCredentials(*creds))
		} else {
			auth = *creds
		}
	} else {
		var fname string
		if *cmsAuth == "" {
			for _, item := range os.Environ() {
				value := strings.Split(item, "=")
				if value[0] == "HOME" {
					fname = fmt.Sprintf("%s/.nats/cms-auth", value[1])
					break
				}
			}
		}
		if fname == "" {
			fname = *cmsAuth
		}
		file, err := os.Open(fname)
		if err != nil {
			fmt.Printf("Unable to open '%s' cms-auth file\n", fname)
			os.Exit(1)
		}
		defer file.Close()
		bytes, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Printf("Unable to read '%s' cms-auth file\n", fname)
			os.Exit(1)
		}
		auth = strings.Replace(string(bytes), "\n", "", -1)
	}
	var servers []string
	for _, v := range strings.Split(*urls, ",") {
		srv := fmt.Sprintf("nats://%s@%s:4222", auth, v)
		if auth == "" {
			srv = fmt.Sprintf("nats://%s:4222", v)
		}
		servers = append(servers, srv)
	}
	// handle user certificates
	if *userKey != "" && *userCert != "" {
		opts = append(opts, nats.ClientCert(*userCert, *userKey))
	}
	// handle root CAs
	if *rootCAs != "" {
		for _, v := range strings.Split(*rootCAs, ",") {
			f := strings.Trim(v, " ")
			opts = append(opts, nats.RootCAs(f))
		}
	}

	// Connect to NATS
	//nc, err := nats.Connect(*urls, opts...)
	nc, err := nats.Connect(strings.Join(servers, ","), opts...)
	if err != nil {
		log.Fatal(err)
	}

	subj, i := args[0], 0
	var attributes []string
	if *attrs != "" {
		attributes = strings.Split(*attrs, ",")
	}

	// start the counter for stats
	start := time.Now().Unix()
	counter := 0
	statsDict := make(map[string]int)

	nc.Subscribe(subj, func(msg *nats.Msg) {
		i += 1
		if time.Now().Unix()-start > int64(*statsInterval) {
			// collect statistics
			if *showStats {
				log.Printf("count(%s) %d\n", subj, counter)
			}
			if *statsBy != "" {
				var sKeys []string
				for k, _ := range statsDict {
					sKeys = append(sKeys, k)
				}
				sort.Sort(StringList(sKeys))
				for _, k := range sKeys {
					if v, ok := statsDict[k]; ok {
						fmt.Printf("%s: %d\n", k, v)
					}
				}
			}
			// send stats to pushgateway server
			if *gatewayUri != "" {
				job := *gatewayJob
				inst := *gatewayInstance
				if job == "" {
					job = subj
				}
				sendToPushGateway(*gatewayUri, job, inst, statsDict)
			}
			// reset start counter
			start = time.Now().Unix()
			counter = 0
			// reset stats dict
			statsDict = make(map[string]int)
		} else {
			// update the counter of docs on given topic
			counter += 1
			if *statsBy != "" {
				key := decodeMsg(msg, *sep, *statsBy)
				if v, ok := statsDict[key]; ok {
					statsDict[key] = v + 1
				} else {
					statsDict[key] = 1
				}
			}
		}
		if !*showStats {
			if *raw {
				printMsg(msg, i)
			} else {
				printCMSMsg(msg, attributes, *sep)
			}
			if *vmUri != "" {
				sendToVictoriaMetrics(*vmUri, msg, subj, *sep)
			}
		}
	})
	nc.Flush()

	if err := nc.LastError(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on [%s/%s]", *urls, subj)
	if *showTime {
		log.SetFlags(log.LstdFlags)
	}

	runtime.Goexit()
}

func setupConnOptions(opts []nats.Option) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectHandler(func(nc *nats.Conn) {
		log.Printf("Disconnected: will attempt reconnects for %.0fm", totalWait.Minutes())
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Printf("Reconnected")
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		log.Fatalf("Exiting: %v", nc.LastError())
	}))
	return opts
}
