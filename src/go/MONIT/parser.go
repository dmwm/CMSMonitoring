package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"

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

//function for unpacking the XML data into TicketsXML Data struct
func (tXml *TicketsXML) parseXML(byteValue []byte) {
	err := xml.Unmarshal(byteValue, &tXml)
	if err != nil {
		log.Printf("Unable to parse XML Data, error: %v\n", err)
		return
	}
}

//function for output the processed data into JSON format
func (tXml *TicketsXML) saveJSON() {
	jsonFile, errF := os.OpenFile("output.json", os.O_WRONLY, 0644)
	if errF != nil {
		log.Printf("Unable to open output.json file, error: %v\n", errF)
		return
	}

	jsonData, err := json.Marshal(tXml)

	if errF != nil {
		log.Printf("Unable to convert into JSON format, error: %v\n", err)
		return
	}

	jsonFile.Write(jsonData)

	defer jsonFile.Close()

}

func getXMLdata() []byte {

	defaultURL := "https://medium.com/feed/@Medium"
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
	// xmlFile, _ := os.Open("output.xml")
	// byteValue, _ := ioutil.ReadAll(xmlFile)

	xmlData := getXMLdata()

	data := &TicketsXML{}
	data.parseXML(xmlData)
	data.saveJSON()

	// defer xmlFile.Close()
}
