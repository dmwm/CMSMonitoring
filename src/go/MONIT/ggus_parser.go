package main

import (
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vkuznet/x509proxy"
)

// File       : ggus_parser.go
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Thu, 07 May 2020 13:34:15 GMT
// Description: Parser for CERN MONIT infrastructure

// Timeout defines timeout for net/url request
var Timeout int

// Verbose defines verbosity level
var Verbose int

//TicketsXML Data struct
type TicketsXML struct {
	Ticket []struct {
		TicketID        int    `xml:"Ticket-ID"`
		Type            string `xml:"Type"`
		VO              string `xml:"VO"`
		Site            string `xml:"Site"`
		Priority        string `xml:"Priority"`
		ResponsibleUnit string `xml:"Responsible_Unit"`
		Status          string `xml:"Status"`
		LastUpdate      string `xml:"Last_Update"`
		Subject         string `xml:"Subject"`
		Scope           string `xml:"Scope"`
	} `xml:"ticket"`
}

//Ticket Data struct (CSV)
type Ticket struct {
	TicketID        int
	Type            *string
	VO              *string
	Site            *string
	Priority        *string
	ResponsibleUnit *string
	Status          *string
	LastUpdate      string
	Subject         *string
	Scope           *string
}

//function for eliminating "none" or empty values in JSON
func nullValueHelper(value **string, data string) {

	if data == "none" || data == "" {
		*value = nil
	} else {
		*value = &data
	}
}

func convertTime(timestamp string) int64 {

	UnixTS, errTime := time.Parse(time.RFC3339, (strings.ReplaceAll(timestamp, " ", "T") + "Z"))
	if errTime != nil {
		log.Printf("Unable to parse LastUpdate TimeStamp, error: %v\n", errTime)
	}

	return UnixTS.Unix()
}

//function for unpacking the CSV data into Ticket Data struct
func parseCSV(data io.ReadCloser) []Ticket {
	var ticket Ticket
	var tickets []Ticket

	reader := csv.NewReader(data)
	reader.Comma = ';'
	reader.LazyQuotes = true
	csvData, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Unable to read CSV file, error: %v\n", err)
	}

	for ind := range csvData {

		if ind > 0 {

			ticket.TicketID, _ = strconv.Atoi(csvData[ind][0])

			nullValueHelper(&ticket.Type, csvData[ind][1])
			nullValueHelper(&ticket.VO, csvData[ind][2])
			nullValueHelper(&ticket.Site, csvData[ind][3])
			nullValueHelper(&ticket.Priority, csvData[ind][4])
			nullValueHelper(&ticket.ResponsibleUnit, csvData[ind][5])
			nullValueHelper(&ticket.Status, csvData[ind][6])

			ticket.LastUpdate = strconv.FormatInt(convertTime(csvData[ind][7]), 10)

			nullValueHelper(&ticket.Subject, csvData[ind][8])
			nullValueHelper(&ticket.Scope, csvData[ind][9])
			tickets = append(tickets, ticket)
		}
	}
	return tickets
}

//function for unpacking the XML data into TicketsXML Data struct
func (tXml *TicketsXML) parseXML(data io.ReadCloser) {

	byteValue, err := ioutil.ReadAll(data)
	if err != nil {
		log.Printf("Unable to read XML Data, error: %v\n", err)
		return
	}

	errp := xml.Unmarshal(byteValue, &tXml)
	if errp != nil {
		log.Printf("Unable to parse XML Data, error: %v\n", errp)
		return
	}

	for ind := range tXml.Ticket {
		tXml.Ticket[ind].LastUpdate = strconv.FormatInt(convertTime(tXml.Ticket[ind].LastUpdate), 10)
	}

}

//function for output the processed data into JSON format
func saveJSON(data interface{}, out string) error {

	jsonData, err := json.Marshal(data)

	if err != nil {
		log.Printf("Unable to convert into JSON format, error: %v\n", err)
		return err
	}

	jsonFile, err := os.Create(out)
	if err != nil {
		fmt.Printf("Unable to create JSON file, error: %v\n", err)
		return err
	}
	defer jsonFile.Close()

	jsonFile.Write(jsonData)
	jsonFile.Close()

	return nil

}

//
// The following block of code was taken from
// https://github.com/dmwm/das2go/blob/master/utils/fetch.go#L75

// TLSCertsRenewInterval controls interval to re-read TLS certs (in seconds)
var TLSCertsRenewInterval time.Duration

// TLSCertsManager holds TLS certificates for the server
type TLSCertsManager struct {
	Certs  []tls.Certificate
	Expire time.Time
}

// GetCerts return fresh copy of certificates
func (t *TLSCertsManager) GetCerts() ([]tls.Certificate, error) {
	var lock = sync.Mutex{}
	lock.Lock()
	defer lock.Unlock()
	// we'll use existing certs if our window is not expired
	if t.Certs == nil || time.Since(t.Expire) > TLSCertsRenewInterval {
		if Verbose > 0 {
			log.Println("read new certs with expire", t.Expire)
		}
		t.Expire = time.Now()
		certs, err := tlsCerts()
		if err == nil {
			t.Certs = certs
		} else {
			panic(err.Error())
		}
	}
	return t.Certs, nil
}

// global TLSCerts manager
var tlsManager TLSCertsManager

// client X509 certificates
func tlsCerts() ([]tls.Certificate, error) {
	uproxy := os.Getenv("X509_USER_PROXY")
	uckey := os.Getenv("X509_USER_KEY")
	ucert := os.Getenv("X509_USER_CERT")
	// check if /tmp/x509up_u$UID exists, if so setup X509_USER_PROXY env
	u, err := user.Current()
	if err == nil {
		fname := fmt.Sprintf("/tmp/x509up_u%s", u.Uid)
		if _, err := os.Stat(fname); err == nil {
			uproxy = fname
		}
	}

	if uproxy == "" && uckey == "" { // user doesn't have neither proxy or user certs
		return nil, nil
	}
	if uproxy != "" {
		// use local implementation of LoadX409KeyPair instead of tls one
		x509cert, err := x509proxy.LoadX509Proxy(uproxy)
		if err != nil {
			return nil, fmt.Errorf("failed to parse X509 proxy: %v", err)
		}

		certs := []tls.Certificate{x509cert}
		return certs, nil
	}
	x509cert, err := tls.LoadX509KeyPair(ucert, uckey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user X509 certificate: %v", err)
	}

	certs := []tls.Certificate{x509cert}
	return certs, nil
}

// HTTPClient is HTTP client for urlfetch server
func HTTPClient() *http.Client {
	// get X509 certs
	certs, err := tlsManager.GetCerts()
	if err != nil {
		panic(err.Error())
	}

	timeout := time.Duration(Timeout) * time.Second
	if len(certs) == 0 {
		if Timeout > 0 {
			return &http.Client{Timeout: time.Duration(timeout)}
		}
		return &http.Client{}
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{Certificates: certs,
			InsecureSkipVerify: true},
	}
	if Timeout > 0 {
		return &http.Client{Transport: tr, Timeout: timeout}
	}
	return &http.Client{Transport: tr}
}

//processResponse function for fetching data from GGUS endpoint and dumping it into JSON format
func processResponse(url, format, accept, out string) {
	var req *http.Request
	req, _ = http.NewRequest("GET", url, nil)
	req.Header.Add("Accept-Encoding", "identity")
	req.Header.Add("Accept", accept)

	client := HTTPClient()

	if Verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("request: ", string(dump))
		}
	}

	resp, err := client.Do(req)

	if err != nil {
		log.Println(err)
	}

	if Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("response:", string(dump))
		}
	}

	defer resp.Body.Close()

	if format == "csv" {
		data := parseCSV(resp.Body)
		saveJSON(data, out)
	} else if format == "xml" {
		data := &TicketsXML{}
		data.parseXML(resp.Body)
		saveJSON(data.Ticket, out)
	}
}

func main() {
	var format string
	var out string
	flag.StringVar(&format, "format", "csv", "GGUS data-format to use (csv or xml)")
	flag.StringVar(&out, "out", "", "out filename")
	flag.IntVar(&Verbose, "verbose", 0, "verbosity level")
	flag.IntVar(&Timeout, "timeout", 0, "http client timeout operation, zero means no timeout")
	flag.Parse()

	ggus := "https://ggus.eu/?mode=ticket_search&status=open&date_type=creation+date&tf_radio=1&timeframe=any&orderticketsby=REQUEST_ID&orderhow=desc"
	var accept string
	if format == "csv" {
		ggus = ggus + "&writeFormat=CSV"
		accept = "text/csv"
	} else if format == "xml" {
		ggus = ggus + "&writeFormat=XML"
		accept = "text/xml"
	}

	if out == "" {
		log.Fatalf("Output filename missing. Exiting....")
	}

	processResponse(ggus, format, accept, out)

}
