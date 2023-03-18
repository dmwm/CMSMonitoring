package models

// Copyright (c) 2023 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>
//
// Inspired by https://github.com/dmwm/dbs2go/blob/master/web/config.go

import (
	"encoding/json"
	"log"
)

// Configuration stores configuration parameters
type Configuration struct {
	Verbose                int                  `json:"verbose"`                  // verbosity level {0: warn, 1: info, 2: debug, 3: detailed debug}
	IsTest                 bool                 `json:"is_test"`                  // if the app is test run, sets "localhost:PORT" instead of ":PORT"
	Port                   int                  `json:"port"`                     // web service port number
	EnvFile                string               `json:"env_file"`                 // secret environment file path for MongoDD connection credentials
	ReadTimeout            int                  `json:"read_timeout"`             // web service read timeout in sec
	WriteTimeout           int                  `json:"write_timeout"`            // web service write timeout in sec
	MongoConnectionTimeout int                  `json:"mongo_connection_timeout"` // mongo connection timeout in sec
	BaseEndpoint           string               `json:"base_endpoint"`            // base_endpoint of web service
	CollectionNames        MongoCollectionNames `json:"collection_names"`         // mongodb collection names
	ExternalLinks          ExternalLinks        `json:"external_links"`           // external links
}

// ExternalLinks mongo collection names struct
type ExternalLinks struct {
	StrStartDate             string `json:"str_start_date"`              // replace string of str_start_date
	StrEndDate               string `json:"str_end_date"`                // replace string of str_end_date
	StrWorkflow              string `json:"str_workflow"`                // replace string of str_workflow
	StrWmagentRequestName    string `json:"str_wmagent_request_name"`    // replace string of str_wmagent_request_name
	StrSite                  string `json:"str_site"`                    // replace string of str_wmagent_request_name
	StrJobType               string `json:"str_job_type"`                // replace string of str_job_type stepchain
	StrJobState              string `json:"str_job_state"`               // replace string of str_job_state stepchain
	StrTaskName              string `json:"str_task_name"`               // replace string of str_task_name stepchain
	StrTaskNamePrefix        string `json:"str_task_name_prefix"`        // replace string of str_task_name_prefix stepchain
	LinkMonteCarloManagement string `json:"link_monte_carlo_management"` // link of string of link_monte_carlo_management
	LinkEsCms                string `json:"link_es_cms"`                 // link of string of link_es_cms
	LinkEsCmsWithSite        string `json:"link_es_cms_with_site"`       // link of string of link_es_cms_with_site
	LinkDmytroProdMon        string `json:"link_dmytro_prod_mon"`        // link of string of link_dmytro_prod_mon
	LinkUnifiedReport        string `json:"link_unified_report"`         // link of string of link_unified_logs
	LinkEsWmarchive          string `json:"link_es_wmarchive"`           // link of string of link_es_wmarchive stepchain
	LinkEsWmarchiveJobType   string `json:"link_es_wmarchive_jobtype"`   // link of string of link_es_wmarchive_jobtype stepchain
	LinkReqMgr               string `json:"link_reqmgr"`                 // link of string of link_reqmgr stepchain
}

// MongoCollectionNames mongo collection names struct
type MongoCollectionNames struct {
	CondorMainWorkflows     string `json:"condor_main"`                 // condor main wf cpu efficiencies collection name
	CondorDetailed          string `json:"condor_detailed"`             // condor detailed wf site cpu efficiencies collection name
	CondorTiers             string `json:"condor_tiers"`                // condor tiers wf site cpu efficiencies collection name
	ScTask                  string `json:"sc_task"`                     // Stepchain Task cpu efficiencies collection name
	ScTaskCmsrun            string `json:"sc_task_cmsrun"`              // Stepchain Task, CmsRun details cpu efficiencies collection name
	ScTaskCmsrunJobtype     string `json:"sc_task_cmsrun_jobtype"`      // Stepchain Task, CmsRun, JobType details cpu efficiencies collection name
	ScTaskCmsrunJobtypeSite string `json:"sc_task_cmsrun_jobtype_site"` // Stepchain Task, CmsRun, JobType, Site details cpu efficiencies collection name
	ShortUrl                string `json:"short_url"`                   // short_url collection name
	DatasourceTimestamp     string `json:"datasource_timestamp"`        // datasource_timestamp collection name
}

// String returns string representation of cpueff Config
func (c *Configuration) String() string {
	data, err := json.Marshal(c)
	if err != nil {
		log.Println("[ERROR] fail to marshal configuration", err)
		return ""
	}
	return string(data)
}
