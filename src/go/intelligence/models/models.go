package models

import "time"

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CERN MONIT infrastructure Intelligence Module

//AmJSON AlertManager API acceptable JSON Data
type AmJSON struct {
	Labels      map[string]interface{} `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`
	StartsAt    time.Time              `json:"startsAt"`
	EndsAt      time.Time              `json:"endsAt"`
}

//AmData data struct, array of AmJSON
type AmData struct {
	Data []AmJSON
}

//Matchers for Silence Data
type Matchers struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

//SilenceData data struct
type SilenceData struct {
	Matchers  []Matchers `json:"matchers"`
	StartsAt  time.Time  `json:"startsAt"`
	EndsAt    time.Time  `json:"endsAt"`
	CreatedBy string     `json:"createdBy"`
	Comment   string     `json:"comment"`
}

//server data struct
type server struct {
	CMSMONURL      string        `json:"cmsmonURL"`
	GetAlertsAPI   string        `json:"getAlertsAPI"`
	PostAlertsAPI  string        `json:"postAlertsAPI"`
	PostSilenceAPI string        `json:"postSilenceAPI"`
	HTTPTimeout    int           `json:"httpTimeout"`
	Interval       time.Duration `json:"interval"`
	Verbose        int           `json:"verbose"`
}

//alert data struct
type alert struct {
	UniqueLabel          string         `json:"uniqueLabel"`
	SeverityLabel        string         `json:"severityLabel"`
	ServiceLabel         string         `json:"serviceLabel"`
	DefaultSeverityLevel string         `json:"defaultSeverityLevel"`
	SeverityLevels       map[string]int `json:"severityLevels"`
}

//silence data struct
type silence struct {
	CreatedBy string `json:"createdBy"`
	Comment   string `json:"comment"`
}

//Service data struct
type Service struct {
	Name                 string            `json:"name"`
	KeywordLabel         string            `json:"keywordLabel"`
	DefaultLevel         string            `json:"defaultLevel"`
	KeywordMatchFunction string            `json:"keywordMatchFunction"`
	SeverityMap          map[string]string `json:"severityMap"`
}

//Config data struct
type Config struct {
	Server   server    `json:"server"`
	Alerts   alert     `json:"alerts"`
	Silence  silence   `json:"silence"`
	Services []Service `json:"services"`
}
