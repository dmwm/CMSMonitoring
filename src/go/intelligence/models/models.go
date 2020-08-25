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
	ID        string     `json:"id"`        // ID for each silence
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
type AllDashboardsFetched struct {
	ID          float64  `json:"id"`          //ID for the dashboard
	UID         string   `json:"uid"`         //UID for the dashboard
	Title       string   `json:"title"`       //Title for the dashboard
	URI         string   `json:"uri"`         //URI of the dashboard
	URL         string   `json:"url"`         //URL of the dashboard
	Slug        string   `json:"slug"`        //Slug of the dashboard
	Type        string   `json:"type"`        //Type of the dashboard
	Tags        []string `json:"tags"`        //All tags for the dashboard	(eg. prod, jobs, cmsweb etc.)
	IsStarred   bool     `json:"isStarred"`   //if dashboard is starred
	FolderID    float64  `json:"folderId"`    //ID for the folder
	FolderUID   string   `json:"folderUid"`   //UID for the folder
	FolderTitle string   `json:"folderTitle"` //Title of the folder
	FolderURL   string   `json:"folderUrl"`   //URL of the folder
}

//GrafanaDashboard data struct for storing Annotation's information to each dashboard
type GrafanaDashboard struct {
	DashboardID float64  `json:"dashboardId"` // ID of a dashboard
	Time        int64    `json:"time"`        //Start Time of the annotation
	TimeEnd     int64    `json:"timeEnd"`     //End Time of the annotation
	Tags        []string `json:"tags"`        //Dashboard tags to be annotated (eg. prod, jobs, cmsweb etc.)
	Text        string   `json:"text"`        //Annotation Text Field
}

//TestingData data struct
type TestingData struct {
	TestFile             string        `json:"testfile"`             //Test cases file name for testing
	LifetimeOfTestAlerts time.Duration `json:"lifetimeOfTestAlerts"` //Lifetime of test alerts (in minutes)
	AnnotateTestStatus   bool          `json:"annotateTestStatus"`   //Check for dashboards annotation during testing
}

//server data struct
type server struct {
	CMSMONURL              string        `json:"cmsmonURL"`              //CMSMON URL for AlertManager API
	GetAlertsAPI           string        `json:"getAlertsAPI"`           //API endpoint from fetching alerts
	GetSuppressedAlertsAPI string        `json:"getSuppressedAlertsAPI"` //API endpoint from fetching suppressed alerts
	PostAlertsAPI          string        `json:"postAlertsAPI"`          //API endpoint from creating new alerts
	GetSilencesAPI         string        `json:"getSilencesAPI"`         //API endpoint from fetching silences
	PostSilenceAPI         string        `json:"postSilenceAPI"`         //API endpoint from silencing alerts
	DeleteSilenceAPI       string        `json:"deleteSilenceAPI"`       //API endpoint from deleting silences
	HTTPTimeout            int           `json:"httpTimeout"`            //Timeout for HTTP Requests
	Interval               time.Duration `json:"interval"`               //Time Interval at which the intelligence service will repeat
	Verbose                int           `json:"verbose"`                //Verbosity Level
	DryRun                 bool          `json:"dryRun"`                 //DryRun boolean flag for dry run
	Testing                TestingData   `json:"testing"`                // Testing struct for storing details about test scenario
}

type annotationDashboard struct {
	URL                       string        `json:"URL"`                       //Dashboards' Base URL for sending annotation
	DashboardSearchAPI        string        `json:"dashboardSearchAPI"`        //API endpoint for searching dashboards with tags
	AnnotationAPI             string        `json:"annotationAPI"`             //API endpoint for pushing annotations
	Tags                      []string      `json:"tags"`                      //Tags for the dashboards
	Token                     string        `json:"token"`                     //Admin's Token required for sending requests
	DashboardsCacheExpiration time.Duration `json:"dashboardsCacheExpiration"` //Dashboard Cache Expiration in terms of hour(s)
	IntelligenceModuleTag     string        `json:"intelligenceModuleTag"`     //Tag attached to annotations which reflects it is made by the intelligence module (eg. "cmsmon-int")
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
	CreatedBy     string   `json:"createdBy"`     //Name of the creater of the silence (Made configurable)
	Comment       string   `json:"comment"`       //Comment for the silence (Made configurable)
	SilenceStatus []string `json:"silenceStatus"` //Labels for status of the silence
}

//annotations struct for storing specific set of keywords & set of dashboards to annotate when these keywords are matched in alerts.
type annotationsData struct {
	Actions []string `json:"actions"` //A set of keywords of actions taken (eg. outage, intervention, update, upgrade, down etc.)
	Systems []string `json:"systems"` //A set of keywords of systems affected (eg. network, database, rucio etc.)
	Tags    []string `json:"tags"`    //A list of tags of dashboards in Grafana
}

type annotationMap struct {
	Label           string            `json:"label"`       //Unique field of the alert Data where descriptive information about it is given so that keywords are matched in here (eg. for SSB --> "shortDescription", for GGUS --> "Subject" etc.)
	AnnotationsData []annotationsData `json:"annotations"` //annotationsData struct
	URLLabel        string            `json:"urlLabel"`    //Field which identifies URL in the alert data (eg. "URL" has been set for both GGUS & SSB).
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

//ChangeCounters data struct
type ChangeCounters struct {
	NoOfAlerts          int //No of alerts in AM
	NoOfPushedAlerts    int //No of alerts being pushed to AM
	NoOfSilencesCreated int //No of new silences created in AM
	NoOfSilencesDeleted int //No of new silences deleted from AM
	NoOfActiveSilences  int //No of active Silences in AM
	NoOfExpiredSilences int //No of expired Silences in AM
	NoOfPendingSilences int //No of pending Silences in AM
}
