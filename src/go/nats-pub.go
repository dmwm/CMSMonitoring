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
	"strings"

	"github.com/nats-io/nats.go"
)

// NOTE: Can test with demo servers.
// nats-pub -s demo.nats.io <subject> <msg>
// nats-pub -s demo.nats.io:4443 <subject> <msg> (TLS version)

func usage() {
	log.Printf("Usage: nats-pub [-s server] [-creds file] <subject> <msg>\n")
	flag.PrintDefaults()
}

func showUsageAndExit(exitcode int) {
	usage()
	os.Exit(exitcode)
}

func main() {
	var urls = flag.String("s", "cms-nats.cern.ch", "The nats server URLs (separated by comma)")
	var creds = flag.String("creds", "", "User NSC credentials")
	var cmsAuth = flag.String("cmsAuth", "", "User cms auth file")
	var showHelp = flag.Bool("h", false, "Show help message")
	var rootCAs = flag.String("rootCAs", "", "Comma separated list of CERN Root CAs files")
	var userKey = flag.String("userKey", "", "x509 user key file")
	var userCert = flag.String("userCert", "", "x509 user certificate")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if *showHelp {
		showUsageAndExit(0)
	}

	args := flag.Args()
	if len(args) != 2 {
		showUsageAndExit(1)
	}

	// Connect Options.
	opts := []nats.Option{nats.Name("NATS Sample Publisher")}

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
	defer nc.Close()
	subj, msg := args[0], []byte(args[1])

	nc.Publish(subj, msg)
	nc.Flush()

	if err := nc.LastError(); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("Published on [%s/%s]: '%s'\n", *urls, subj, msg)
	}
}
