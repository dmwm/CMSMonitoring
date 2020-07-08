package models

import "time"

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CERN MONIT infrastructure Intelligence Module

//AmJSON AlertManager API acceptable JSON Data
type AmJSON struct {
	Labels      map[string]interface{} `json:"labels"`      // Map of Labels for each alert
	Annotations map[string]interface{} `json:"annotations"` //Map of Annotations for each alert
	StartsAt    time.Time              `json:"startsAt"`    //Starting time of an alert
	EndsAt      time.Time              `json:"endsAt"`      //Ending time of an alert
}

//AmData data struct, array of AmJSON
type AmData struct {
	Data []AmJSON // Array of struct AmJSON required for GET API call data storage
}

//Matchers for Silence Data
type Matchers struct {
	Name  string `json:"name"`  //Name of a matcher
	Value string `json:"value"` //Value of a matcher
}

//SilenceData data struct
type SilenceData struct {
	Matchers  []Matchers `json:"matchers"`  //Array of matchers which helps in finding the alert for silencing
	StartsAt  time.Time  `json:"startsAt"`  //Starting time of a silence
	EndsAt    time.Time  `json:"endsAt"`    //Ending time of a silence
	CreatedBy string     `json:"createdBy"` //Name of the creater of the silence
	Comment   string     `json:"comment"`   //Comment for the silence
}

//server data struct
type server struct {
	CMSMONURL      string        `json:"cmsmonURL"`      //CMSMON URL for AlertManager API
	GetAlertsAPI   string        `json:"getAlertsAPI"`   //API endpoint from fetching alerts
	PostAlertsAPI  string        `json:"postAlertsAPI"`  //API endpoint from creating new alerts
	PostSilenceAPI string        `json:"postSilenceAPI"` //API endpoint from silencing alerts
	HTTPTimeout    int           `json:"httpTimeout"`    //Timeout for HTTP Requests
	Interval       time.Duration `json:"interval"`       //Time Interval at which the intelligence service will repeat
	Verbose        int           `json:"verbose"`        //Verbosity Level
}

//alert data struct
type alert struct {
	UniqueLabel          string         `json:"uniqueLabel"`          //Label which defines an alert uniquely
	SeverityLabel        string         `json:"severityLabel"`        //Label for severity level of an alert
	ServiceLabel         string         `json:"serviceLabel"`         //Label for service of an alert
	DefaultSeverityLevel string         `json:"defaultSeverityLevel"` //Default severity level value in case service is not able to assign one
	SeverityLevels       map[string]int `json:"severityLevels"`       //Map for defined severity levels and their priority
}

//silence data struct
type silence struct {
	CreatedBy string `json:"createdBy"` //Name of the creater of the silence (Made configurable)
	Comment   string `json:"comment"`   //Comment for the silence (Made configurable)
}

//Service data struct
type Service struct {
	Name                 string            `json:"name"`                 //Name of a service (eg. SSB, GGUS)
	KeywordLabel         string            `json:"keywordLabel"`         //Field in which the service tries to match keyword
	DefaultLevel         string            `json:"defaultLevel"`         //Default Severity Level assigned to the alert at the time of it's creation
	KeywordMatchFunction string            `json:"keywordMatchFunction"` //Function name which handles the keyword matching for a service
	SeverityMap          map[string]string `json:"severityMap"`          //Map for severity levels for a service
}

//Config data struct
type Config struct {
	Server   server    `json:"server"`   //server struct
	Alerts   alert     `json:"alerts"`   //Alert struct
	Silence  silence   `json:"silence"`  //Silence struct
	Services []Service `json:"services"` //Array of Service
}
