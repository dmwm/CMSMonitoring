package main

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

// File       : parser.go
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Thu, 07 May 2020 13:34:15 GMT
// Description: Parser for CERN MONIT infrastructure

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

//Ticket Data struct
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

func nullValueHelper(value **string, data string) {

	if data == "none" || data == "" {
		*value = nil
	} else {
		*value = &data
	}
}

//function for unpacking the CSV data into Ticket Data struct
func parseCSV(in string) []Ticket {
	var ticket Ticket
	var tickets []Ticket

	csvFile, err := os.Open(in)
	if err != nil {
		fmt.Printf("Unable to open CSV file, error: %v\n", err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)

	csvData, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Unable to read CSV file, error: %v\n", err)
		return tickets
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
func (tXml *TicketsXML) parseXML(byteValue []byte) {
	err := xml.Unmarshal(byteValue, &tXml)
	if err != nil {
		log.Printf("Unable to parse XML Data, error: %v\n", err)
		return
	}
}

//function for output the processed data into JSON format
func saveJSON(data []Ticket, out string) {

	jsonData, err := json.Marshal(data)

	if err != nil {
		log.Printf("Unable to convert into JSON format, error: %v\n", err)
		return
	}

	jsonFile, err := os.Create(out)
	if err != nil {
		fmt.Printf("Unable to create JSON file, error: %v\n", err)
	}
	defer jsonFile.Close()

	jsonFile.Write(jsonData)
	jsonFile.Close()

}

//function for fetching XML data from GGUS Ticketing System Endpoint
func getXMLdata() []byte {

	defaultURL := "<GGUS_Ticketing_System_URL>"
	var XMLdata []byte

	collector := colly.NewCollector()

	collector.OnResponse(func(resp *colly.Response) {
		XMLdata = resp.Body
	})

	collector.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	collector.Visit(defaultURL)

	return XMLdata

}

func main() {
	var in string
	var out string
	flag.StringVar(&in, "in", "", "input filename")
	flag.StringVar(&out, "out", "", "out filename")
	flag.Parse()

	if in == "" {
		log.Fatalf("Input filename missing. Exiting....")
	}

	if out == "" {
		log.Fatalf("Output filename missing. Exiting....")
	}

	csvData := parseCSV(in)
	saveJSON(csvData, out)
}
