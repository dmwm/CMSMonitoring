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
	"os"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"time"

	logs "github.com/sirupsen/logrus"
	"github.com/vkuznet/x509proxy"
)

// File       : parser.go
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Thu, 07 May 2020 13:34:15 GMT
// Description: Parser for CERN MONIT infrastructure

// TIMEOUT defines timeout for net/url request
var TIMEOUT int

//TicketsXML Data struct
type TicketsXML struct {
	Ticket []struct {
		TicketID        string `xml:"Ticket-ID"`
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
	LastUpdate      int64
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

//CSVFile function for CSVFile Read
func CSVFile(in string) [][]string {
	csvFile, err := os.Open(in)
	if err != nil {
		fmt.Printf("Unable to open CSV file, error: %v\n", err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)

	csvData, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Unable to read CSV file, error: %v\n", err)
	}

	return csvData
}

//function for unpacking the CSV data into Ticket Data struct
func parseCSV(data io.ReadCloser) []Ticket {
	var ticket Ticket
	var tickets []Ticket

	reader := csv.NewReader(data)
	csvData, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Unable to read CSV file, error: %v\n", err)
	}

	for ind := range csvData {

		if ind > 0 {
			each := strings.Split(csvData[ind][0], ";")

			ticket.TicketID, _ = strconv.Atoi(each[0])

			nullValueHelper(&ticket.Type, each[1])
			nullValueHelper(&ticket.VO, each[2])
			nullValueHelper(&ticket.Site, each[3])
			nullValueHelper(&ticket.Priority, each[4])
			nullValueHelper(&ticket.ResponsibleUnit, each[5])
			nullValueHelper(&ticket.Status, each[6])

			UnixTS, _ := time.Parse(time.RFC3339, (strings.ReplaceAll(each[7], " ", "T") + "Z"))
			ticket.LastUpdate = UnixTS.Unix()

			nullValueHelper(&ticket.Subject, each[8])
			nullValueHelper(&ticket.Scope, each[9])
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
	if err != nil {
		log.Printf("Unable to parse XML Data, error: %v\n", errp)
		return
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
		logs.WithFields(logs.Fields{
			"Expire": t.Expire,
		}).Debug("Read new certs")
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

	timeout := time.Duration(TIMEOUT) * time.Second
	if len(certs) == 0 {
		if TIMEOUT > 0 {
			return &http.Client{Timeout: time.Duration(timeout)}
		}
		return &http.Client{}
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{Certificates: certs,
			InsecureSkipVerify: true},
	}
	if TIMEOUT > 0 {
		return &http.Client{Transport: tr, Timeout: timeout}
	}
	return &http.Client{Transport: tr}
}

//Response Data struct
type Response struct {
	data io.ReadCloser
	err  error
}

//FetchResponse response
func FetchResponse(url string) Response {
	var response Response
	var req *http.Request
	req, _ = http.NewRequest("GET", url, nil)
	req.Header.Add("Accept-Encoding", "identity")
	req.Header.Add("Accept", "text/csv")

	client := HTTPClient()

	resp, err := client.Do(req)

	if err != nil {
		response.err = err
		return response
	}
	defer resp.Body.Close()

	response.data = resp.Body
	return response
}

func main() {
	var csvURL string
	var xmlURL string
	var out string
	flag.StringVar(&csvURL, "csv", "", "GGUS CSV endpoint")
	flag.StringVar(&xmlURL, "xml", "", "GGUS XML endpoint")
	flag.StringVar(&out, "out", "", "out filename")

	flag.Parse()

	if csvURL == "" || xmlURL == "" {
		log.Fatalf("GGUS endpoint URL missing. Exiting....")
	}

	if out == "" {
		log.Fatalf("Output filename missing. Exiting....")
	}

	if csvURL != "" {
		csvData := FetchResponse(csvURL)
		Data := parseCSV(csvData.data)
		saveJSON(Data, out)
	} else {
		xmlData := FetchResponse(xmlURL)
		Data := &TicketsXML{}
		Data.parseXML(xmlData.data)
		saveJSON(Data, out)
	}

}
