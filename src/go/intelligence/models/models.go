package models

import "time"

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CMS MONIT infrastructure Intelligence Module

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

// Status struct for the SilenceData
type status struct {
	State string `json:"state"` // Status of the Silence
}

//SilenceData data struct
type SilenceData struct {
	Matchers  []Matchers `json:"matchers"`  //Array of matchers which helps in finding the alert for silencing
	StartsAt  time.Time  `json:"startsAt"`  //Starting time of a silence
	EndsAt    time.Time  `json:"endsAt"`    //Ending time of a silence
	CreatedBy string     `json:"createdBy"` //Name of the creater of the silence
	Comment   string     `json:"comment"`   //Comment for the silence
	Status    status     `json:"status"`    //Status of the silence
}

//AllSilences data struct, array of SilenceData
type AllSilences struct {
	Data []SilenceData // Array of struct SilenceData required for GET API call data storage
}

//AllDashboardsFetched is an array of all Dashboards' information having given tags in common
type AllDashboardsFetched []struct {
	ID          float64  `json:"id"`
	UID         string   `json:"uid"`
	Title       string   `json:"title"`
	URI         string   `json:"uri"`
	URL         string   `json:"url"`
	Slug        string   `json:"slug"`
	Type        string   `json:"type"`
	Tags        []string `json:"tags"`
	IsStarred   bool     `json:"isStarred"`
	FolderID    float64  `json:"folderId"`
	FolderUID   string   `json:"folderUid"`
	FolderTitle string   `json:"folderTitle"`
	FolderURL   string   `json:"folderUrl"`
}

//GrafanaDashboard data struct for storing Annotation's information to each dashboard
type GrafanaDashboard struct {
	DashboardID float64  `json:"dashboardId"`
	Time        int64    `json:"time"`
	TimeEnd     int64    `json:"timeEnd"`
	Tags        []string `json:"tags"`
	Text        string   `json:"text"`
}

//server data struct
type server struct {
	CMSMONURL      string        `json:"cmsmonURL"`      //CMSMON URL for AlertManager API
	GetAlertsAPI   string        `json:"getAlertsAPI"`   //API endpoint from fetching alerts
	PostAlertsAPI  string        `json:"postAlertsAPI"`  //API endpoint from creating new alerts
	GetSilencesAPI string        `json:"getSilencesAPI"` //API endpoint from fetching silences
	PostSilenceAPI string        `json:"postSilenceAPI"` //API endpoint from silencing alerts
	HTTPTimeout    int           `json:"httpTimeout"`    //Timeout for HTTP Requests
	Interval       time.Duration `json:"interval"`       //Time Interval at which the intelligence service will repeat
	Verbose        int           `json:"verbose"`        //Verbosity Level
}

type annotationDashboard struct {
	URL                       string        `json:"URL"`                       //Dashboards' Base URL for sending annotation
	DashboardSearchAPI        string        `json:"dashboardSearchAPI"`        //API endpoint for searching dashboards with tags
	AnnotationAPI             string        `json:"annotationAPI"`             //API endpoint for pushing annotations
	Tags                      []string      `json:"tags"`                      //Tags for the dashboards
	Token                     string        `json:"token"`                     //Admin's Token required for sending requests
	DashboardsCacheExpiration time.Duration `json:"dashboardsCacheExpiration"` //Dashboard Cache Expiration in terms of hour(s)
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
	CreatedBy    string `json:"createdBy"`    //Name of the creater of the silence (Made configurable)
	Comment      string `json:"comment"`      //Comment for the silence (Made configurable)
	ActiveStatus string `json:"activeStatus"` //Label for active status of the silence
}

type annotationMap struct {
	Label   string   `json:"label"`
	Actions []string `json:"actions"`
	Systems []string `json:"systems"`
}

//Service data struct
type Service struct {
	Name          string            `json:"name"`          //Name of a service (eg. SSB, GGUS)
	KeywordLabel  string            `json:"keywordLabel"`  //Field in which the service tries to match keyword
	DefaultLevel  string            `json:"defaultLevel"`  //Default Severity Level assigned to the alert at the time of it's creation
	SeverityMap   map[string]string `json:"severityMap"`   //Map for severity levels for a service
	AnnotationMap annotationMap     `json:"annotationMap"` //Map for Dashboard annotations' keywords
}

//Config data struct
type Config struct {
	Server              server              `json:"server"`              //server struct
	AnnotationDashboard annotationDashboard `json:"annotationDashboard"` //annotation Dashboard struct
	Alerts              alert               `json:"alerts"`              //Alert struct
	Silence             silence             `json:"silence"`             //Silence struct
	Services            []Service           `json:"services"`            //Array of Service
}
