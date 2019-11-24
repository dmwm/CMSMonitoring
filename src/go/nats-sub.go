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
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
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

func printMsg(m *nats.Msg, i int) {
	log.Printf("[#%d] [%s] '%s'", i, m.Subject, string(m.Data))
}
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

func main() {
	var urls = flag.String("s", "cms-nats.cern.ch", "The nats server URLs (separated by comma)")
	var creds = flag.String("creds", "", "User NSC credentials")
	var cmsAuth = flag.String("cmsAuth", "", "User cms auth file")
	var showTime = flag.Bool("t", false, "Display timestamps")
	var showHelp = flag.Bool("h", false, "Show help message")
	var attrs = flag.String("attrs", "", "comma separated list of attributes to show, e.g. campaign,dataset")
	var sep = flag.String("sep", "   ", "message attribute separator, default 3 spaces")
	var raw = flag.Bool("raw", false, "Show raw messages")

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

	nc.Subscribe(subj, func(msg *nats.Msg) {
		i += 1
		if *raw {
			printMsg(msg, i)
		} else {
			printCMSMsg(msg, attributes, *sep)
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
		log.Printf("Reconnected [%s]", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		log.Fatalf("Exiting: %v", nc.LastError())
	}))
	return opts
}
