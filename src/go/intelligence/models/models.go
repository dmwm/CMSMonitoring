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

//Config data struct
type Config struct {
	CMSMONURL      string        `json:"cmsmonURL"`
	GetAlertsAPI   string        `json:"getAlertsAPI"`
	PostAlertsAPI  string        `json:"postAlertsAPI"`
	PostSilenceAPI string        `json:"postSilenceAPI"`
	HTTPTimeout    int           `json:"httpTimeout"`
	Interval       time.Duration `json:"interval"`
	Verbose        int           `json:"verbose"`

	CreatedBy   string `json:"createdBy"`
	UniqueLabel string `json:"uniqueLabel"`
	Comment     string `json:"comment"`

	SeverityLabel string `json:"severityLabel"`

	SsbKeywordLabel         string            `json:"ssbKeywordLabel"`
	DefaultSSBSeverityLevel string            `json:"defaultSSBSeverityLevel"`
	SsbSeverityMap          map[string]string `json:"ssbSeverityMap"`

	GGUSKeywordLabel         string            `json:"ggusKeywordLabel"`
	DefaultGGUSSeverityLevel string            `json:"defaultGGUSSeverityLevel"`
	GGUSSeverityMap          map[string]string `json:"ggusSeverityMap"`

	SeverityLevels       map[string]int `json:"severityLevels"`
	DefaultSeverityLevel string         `json:"defaultSeverityLevel"`
}
