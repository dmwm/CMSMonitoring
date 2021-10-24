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
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/nats-io/nats.go"
)

const (
	topN = 5
	topS = 3
)

// global variables
var (
	urls           *string
	statsInterval  *int
	updateInterval *int
	grid           *ui.Grid
	natsData       *DataType
	exitCodes      map[string]interface{}
	barsData       [][]float64
	dataPoints     []float64
)

// Map is a basic data type we'll use
type Map map[string]int
type SiteCodeMap map[string]Map

// DataObject represent single data object
type DataObject struct {
	Key         string
	Value       int
	Description string
}

// DataObjectList implement sort for []Object type
type DataObjectList []DataObject

// Len provides length of the []DataObject type
func (s DataObjectList) Len() int { return len(s) }

// Swap implements swap function for []DataObject type
func (s DataObjectList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Less implements less function for []DataObject type
func (s DataObjectList) Less(i, j int) bool { return s[i].Value < s[j].Value }

type MsgData struct {
	Site string
	Code string
}

type DataType struct {
	SiteCodes          SiteCodeMap            // map of site codes
	Codes              Map                    // map of codes vs values for every code
	Sites              Map                    // map of sites vs values of total codes on each site
	ExitCodes          map[string]interface{} // map of exit code descriptions
	SitePatterns       []string               // list of site patterns to watch
	InitStatsWidget    *widgets.List
	MetaDataWidget     *widgets.List
	CodeListWidget     *widgets.List
	SiteListWidget     *widgets.List
	CodeSiteListWidget *widgets.List
	Lines              *widgets.SparklineGroup
	BarsWidget         *widgets.StackedBarChart
}

// Reset resets our data type
func (d *DataType) Reset() {
	d.SiteCodes = make(SiteCodeMap)
	d.Codes = make(Map)
	d.Sites = make(Map)
	d.BarsWidget.Labels = []string{}
	d.BarsWidget.Data = make([][]float64, topS) // topS sites
}

// Update calls all updates methods for our data type
func (d *DataType) Update(msg *nats.Msg, sep string, verbose int) {
	natsData.UpdateInitStats()
	natsData.UpdateMetaData()
	natsData.UpdateSiteCodeMap(msg, sep, verbose)
	natsData.UpdateCodes(msg, sep, verbose)
	natsData.UpdateSites(msg, sep, verbose)
	natsData.UpdateLines(msg, sep, verbose)
	//     natsData.UpdateBars(msg, sep, verbose)
	natsData.UpdateSiteCodes(msg, sep, verbose)
	if verbose > 1 {
		log.Println("codes", d.Codes)
		log.Println("sites", d.Sites)
	}
}

// UpdateInitStats updates service parameters
func (d *DataType) UpdateInitStats() {
	var siteList []string
	for s, _ := range d.Sites {
		siteList = append(siteList, s)
	}
	natsServers := strings.Split(*urls, ",")
	rows := []string{
		fmt.Sprintf("Update interval: %d", *updateInterval),
		fmt.Sprintf("Stats interval: %d", *statsInterval),
		fmt.Sprintf("NATS server:\n%v", natsServers),
	}
	d.InitStatsWidget.Rows = rows
}

// UpdateMetaData updates service parameters
func (d *DataType) UpdateMetaData() {
	var totSites, totCodes int
	for range d.Sites {
		totSites += 1
	}
	for range d.Codes {
		totCodes += 1
	}
	rows := []string{
		fmt.Sprintf("Processing Sites: %d", totSites),
		fmt.Sprintf("Total codes: %d", totCodes),
	}
	d.MetaDataWidget.Rows = rows
}

// UpdateSiteCodeMap updates site-codes map
func (d *DataType) UpdateSiteCodeMap(msg *nats.Msg, sep string, verbose int) {
	if d.SiteCodes == nil {
		d.SiteCodes = make(SiteCodeMap)
	}
	code := decodeMsg(msg, "exitCode", sep)
	// count only non-zero codes
	if code == "0" {
		return
	}
	site := decodeMsg(msg, "site", sep)
	if cmap, ok := d.SiteCodes[site]; ok {
		if v, ok := cmap[code]; ok {
			cmap[code] = v + 1
		} else {
			cmap[code] = 1
		}
		d.SiteCodes[site] = cmap
	} else {
		cmap = make(Map)
		cmap[code] = 1
		d.SiteCodes[site] = cmap
	}
	if verbose > 0 {
		log.Println("siteCodes", d.SiteCodes)
	}
}

// UpdateCodes updates codes info
func (d *DataType) UpdateCodes(msg *nats.Msg, sep string, verbose int) {
	key := decodeMsg(msg, "exitCode", sep)
	// update codes
	if v, ok := d.Codes[key]; ok {
		d.Codes[key] = v + 1
	} else {
		d.Codes[key] = 1
	}
	// prepare codes rows
	objects := []DataObject{}
	for k, v := range d.Codes {
		var desc string
		if exitCodes != nil {
			if d, ok := exitCodes[k]; ok {
				desc = fmt.Sprintf("%v", d)
			}
		}
		o := DataObject{Key: k, Value: v, Description: desc}
		objects = append(objects, o)
	}
	sort.Sort(sort.Reverse(DataObjectList(objects)))
	rows := []string{}
	var row string
	for _, o := range objects {
		row = fmt.Sprintf("%4d %5s %s", o.Value, o.Key, o.Description)
		rows = append(rows, row)
	}
	d.CodeListWidget.Rows = rows
}

// UpdateSites updates sites info
func (d *DataType) UpdateSites(msg *nats.Msg, sep string, verbose int) {
	key := decodeMsg(msg, "site", sep)
	if verbose > 1 {
		log.Println("msg", string(msg.Data))
	}
	// update sites
	if v, ok := d.Sites[key]; ok {
		d.Sites[key] = v + 1
	} else {
		d.Sites[key] = 1
	}

	// prepare sites rows
	objects := []DataObject{}
	for k, v := range d.Sites {
		o := DataObject{Key: k, Value: v}
		objects = append(objects, o)
	}
	sort.Sort(sort.Reverse(DataObjectList(objects)))
	rows := []string{}
	var row string
	for _, o := range objects {
		row = fmt.Sprintf("%4d %s", o.Value, o.Key)
		rows = append(rows, row)
	}
	d.SiteListWidget.Rows = rows
}

// UpdateSiteCodes updates site-codes info
func (d *DataType) UpdateSiteCodes(msg *nats.Msg, sep string, verbose int) {
	rows := []string{}
	for s, cmap := range d.SiteCodes {
		objects := []DataObject{}
		for k, v := range cmap {
			o := DataObject{Key: k, Value: v}
			objects = append(objects, o)
		}
		sort.Sort(sort.Reverse(DataObjectList(objects)))
		var codes string
		for _, o := range objects {
			codes = fmt.Sprintf("%s, %d of %s", codes, o.Value, o.Key)
		}
		row := fmt.Sprintf("%s %s", s, codes)
		rows = append(rows, row)
	}
	d.CodeSiteListWidget.Rows = rows
}

// UpdateLines updates Lines widget
func (d *DataType) UpdateLines(msg *nats.Msg, sep string, verbose int) {
	// update lines
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, line := range d.Lines.Sparklines {
		v := float64(r.Intn(50))
		if len(line.Data) == 100 {
			line.Data = line.Data[1:len(line.Data)]
		}
		line.Data = append(line.Data, v)
	}
}

/*
// UpdateBars updates Bars widget
func (d *DataType) UpdateBars(msg *nats.Msg, sep string, verbose int) {
	var siteList []string
	for s, _ := range d.Sites {
		siteList = append(siteList, s)
	}
	//     siteList := d.SitePatterns
	// loop over site-code map to determine list of sites with non-zero codes
	siteDict := make(Map)
//     codeDict := make(Map)
	for k, v := range d.SiteCodes {
		arr := strings.Split(k, "#")
		site := arr[0]
		code := arr[1]
		if code == "0" {
			continue
		}
		if v, ok := siteDict[site]; ok {
			siteDict[site] = v+1
		} else {
			siteDict[site] = 1
		}
//         if v, ok := codeDict[code]; ok {
//             codeDict[code] = v+1
//         } else {
//             codeDict[code] = 1
//         }
	}
	// loop over site dict to find out sites with largest number of non-zero codes
	siteObjects := []DataObjects
	for s, v := range siteDict {
		o := DataObect{Key: s, Value: v}
		siteObjects = append(siteObjects, o)
	}
	sort.Sort(sort.Reverse(DataObjectList(siteObjects)))
//     codeObjects := []DataObjects
//     for s, v := range codeDict {
//         o := DataObect{Key: s, Value: v}
//         codeObjects = append(codeObjects, o)
//     }
//     sort.Sort(sort.Reverse(DataObjectList(codeObjects)))
	// loop over site objects
	for i, s := range siteObjects {
		if i+1 == topS {
			break
		}
		// reset site data points
		for j := 0; j < len(dataPoints); j++ {
			dataPoints[j] = 0.
		}
		codeDict := make(Map)
		for sc, w := range d.SiteCodes {
			if strings.HasPrefix(sc, s) {
				for code, v := range codeDict {
					if v, ok := codeDict[code]; ok {
						codeDict[code] = v+1
					} else {
						codeDict[code] = w
					}
				}
				for c, v := range codeDict {
					o := DataObject{Key: c, Value: v}
					codeObjects = append(codeObjects, o)
				}
				sort.Sort(sort.Reverse(DataObjectList(codeObjects)))
			}
		}
		if len(codeObjects) >= topN {
			codeObjects
		}
		for sc, w := range d.SiteCodes {
		}
	}
				log.Println("bars", s, dataPoints)
				d.BarsWidget.Data[i] = dataPoints
	d.BarsWidget.Labels = siteObjects[:topS]
}
*/

// WidgetStackedBarChart creates new widget
func WidgetBars(title string) *widgets.StackedBarChart {
	sbc := widgets.NewStackedBarChart()
	sbc.Title = title
	sbc.Labels = []string{}
	sbc.Data = barsData
	sbc.SetRect(0, 0, 100, 10)
	sbc.BarWidth = 15
	return sbc
}

// WidgetList creates new List widget
func WidgetList(title string, rows []string) *widgets.List {
	table := widgets.NewList()
	table.Rows = rows
	table.Title = title
	table.SetRect(0, 0, 100, 10)
	return table
}

func WidgetSparklines(topN int, title string) *widgets.SparklineGroup {
	var lines []*widgets.Sparkline
	for i := 0; i < topN; i++ {
		line := widgets.NewSparkline()
		line.Data = []float64{}
		line.Title = fmt.Sprintf("%s-%d", title, i)
		if i == 0 {
			line.LineColor = ui.ColorBlack
		} else if i == 1 {
			line.LineColor = ui.ColorRed
		} else if i == 2 {
			line.LineColor = ui.ColorGreen
		} else if i == 3 {
			line.LineColor = ui.ColorYellow
		} else if i == 4 {
			line.LineColor = ui.ColorBlue
		} else {
			line.LineColor = ui.ColorMagenta
		}
		line.MaxVal = 50
		lines = append(lines, line)
	}
	spark := widgets.NewSparklineGroup(lines...)
	spark.Title = fmt.Sprintf("Top-%d %s", topN, title)
	spark.SetRect(0, 0, 100, 10)
	return spark
}

func setupGrid() {
	grid = ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewRow(0.2,
			ui.NewCol(0.3, natsData.InitStatsWidget),
			ui.NewCol(0.7, natsData.MetaDataWidget),
		),
		ui.NewRow(0.3, natsData.CodeListWidget),
		ui.NewRow(0.5,
			ui.NewCol(0.3, natsData.SiteListWidget),
			ui.NewCol(0.7, natsData.CodeSiteListWidget),
		),
	)
}

func eventLoop() {
	uiEvents := ui.PollEvents()
	origInterval := *updateInterval
	ticker := time.NewTicker(time.Duration(origInterval) * time.Second).C
	for {
		select {
		case e := <-uiEvents:
			switch e.ID { // event string/identifier
			case "q", "<C-c>": // press 'q' or 'C-c' to quit
				return
			case "k": // increase
				*updateInterval = *updateInterval * 2
				ticker = time.NewTicker(time.Duration(*updateInterval) * time.Second).C
			case "j": // decrease
				*updateInterval = *updateInterval / 2
				ticker = time.NewTicker(time.Duration(*updateInterval) * time.Second).C
			case "r": // press 'r' to Reset ticker
				ticker = time.NewTicker(time.Duration(origInterval) * time.Second).C
			}
		// use Go's built-in tickers for updating and drawing data
		case <-ticker:
			ui.Render(grid)
		}
	}
}

// decode NATS message and extract value of given key
func decodeMsg(m *nats.Msg, key, sep string) string {
	msg := string(m.Data)
	for _, v := range strings.Split(msg, sep) {
		arr := strings.Split(v, ":")
		if key == arr[0] {
			return arr[1]
		}
	}
	return ""
}

// helper function to read exit codes from HTTP URL location
func getExitCodes(url string) (map[string]interface{}, error) {
	codes := make(map[string]interface{})
	// read exit code file
	resp, err := http.Get(url)
	if err != nil {
		return codes, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		arr := strings.Split(line, ":")
		codes[strings.Trim(arr[0], " ")] = strings.Trim(arr[1], " ")
	}
	return codes, nil
}

func natsLoop(subj string, servers []string, opts []nats.Option, sep string, verbose int) {
	// Connect to NATS
	nc, err := nats.Connect(strings.Join(servers, ","), opts...)
	if err != nil {
		log.Fatal(err)
	}

	// start the counter for stats
	i := 0
	start := time.Now().Unix()

	nc.Subscribe(subj, func(msg *nats.Msg) {
		i += 1
		if time.Now().Unix()-start > int64(*statsInterval) {
			// printout existing data
			if verbose > 0 {
				log.Println("natsData", natsData)
			}
			// sleep to let termui to draw the data
			sleep := time.Duration(1) * time.Second
			time.Sleep(sleep)
			// reset start counter
			start = time.Now().Unix()
			// reset data types, but keep all widgets
			natsData.Reset()
		} else {
			natsData.Update(msg, sep, verbose)
		}
	})
	nc.Flush()

	if err := nc.LastError(); err != nil {
		log.Fatal(err)
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

func showUsageAndExit(exitcode int) {
	log.Printf("Usage: nats-exitcodes [-s server] [-creds file] [-t] <subject>\n")
	flag.PrintDefaults()
	os.Exit(exitcode)
}

func setupLogfile(logDir, logPath string) (*os.File, error) {
	// create the log directory
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to make the log directory: %v", err)
	}
	// open the log file
	logfile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	// log time, filename, and line number
	log.SetFlags(log.Ltime | log.Lshortfile)
	// log to file
	log.SetOutput(logfile)

	return logfile, nil
}

func main() {
	//     var urls = flag.String("s", "cms-nats.cern.ch", "The nats server URLs (separated by comma)")
	var creds = flag.String("creds", "", "User NSC credentials")
	var cmsAuth = flag.String("cmsAuth", "", "User cms auth file")
	var showTime = flag.Bool("t", false, "Display timestamps")
	var showHelp = flag.Bool("h", false, "Show help message")
	var verbose = flag.Int("verbose", 0, "Show verbose output")
	var logDir = flag.String("logDir", "/tmp/nats", "Use log directory")
	var subj = flag.String("subj", "cms.wmarchive.exitCode.>", "nats topic")
	//     var statsInterval = flag.Int("statsInterval", 10, "stats collect interval in seconds")
	//     var updateInterval = flag.Int("update", 5, "update interval in seconds")
	var sep = flag.String("sep", "   ", "message attribute separator, default 3 spaces")
	var rootCAs = flag.String("rootCAs", "", "Comma separated list of CERN Root CAs files")
	var userKey = flag.String("userKey", "", "x509 user key file")
	var userCert = flag.String("userCert", "", "x509 user certificate")
	var sitePatterns = flag.String("sitePatterns", "", "site patterns to watch")

	// globals
	urls = flag.String("s", "cms-nats.cern.ch", "The nats server URLs (separated by comma)")
	statsInterval = flag.Int("statsInterval", 10, "stats collect interval in seconds")
	updateInterval = flag.Int("update", 5, "update interval in seconds")

	log.SetFlags(0)
	flag.Parse()

	if *showHelp {
		showUsageAndExit(0)
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

	// exitCodes
	url := "https://raw.githubusercontent.com/vkuznet/CMSExitCodes/master/codes/JobExit.txt"
	if ec, err := getExitCodes(url); err == nil {
		exitCodes = ec
	} else {
		log.Printf("Unable to fetch CMS exitCodes codes from %s", url)
	}

	// final settings
	log.Printf("Listening on [%s/%s]", *urls, *subj)
	if *showTime {
		log.SetFlags(log.LstdFlags)
	}
	logPath := filepath.Join(*logDir, "nats.log")
	logfile, err := setupLogfile(*logDir, logPath)
	if *verbose > 0 {
		fmt.Println("Use", logfile, err)
	}

	// init our data types
	barsData = make([][]float64, topS) // top S sites
	dataPoints = make([]float64, topN) // top N non-exit codes
	rows := []string{"no data yet"}
	patterns := strings.Split(*sitePatterns, ",")
	natsData = &DataType{
		Sites:              make(Map),
		Codes:              make(Map),
		InitStatsWidget:    WidgetList("Init status", rows),
		MetaDataWidget:     WidgetList("Meta data", rows),
		SiteListWidget:     WidgetList("Processing Sites", rows),
		CodeListWidget:     WidgetList("ExitCodes", rows),
		CodeSiteListWidget: WidgetList("Sites with non-zero codes", rows),
		Lines:              WidgetSparklines(5, "ExitCodes"),
		BarsWidget:         WidgetBars("Patterns"),
		ExitCodes:          exitCodes,
		SitePatterns:       patterns}

	// perform NATS magic
	go natsLoop(*subj, servers, opts, *sep, *verbose)

	// perform termui magic
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()
	setupGrid()
	eventLoop()
}
