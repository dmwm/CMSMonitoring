package server

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>
//
// Inspired by https://github.com/dmwm/dbs2go/blob/master/web/config.go

import (
	"encoding/json"
	"log"
	"os"
)

// Config represents global configuration object
var Config Configuration

// Configuration stores configuration parameters
type Configuration struct {
	Verbose                int                  `json:"verbose"`                  // verbosity level {0: warn, 1: info, 2: debug, 3: detailed debug}
	Port                   int                  `json:"port"`                     // web service port number
	EnvFile                string               `json:"env_file"`                 // secret environment file path for MongoDD connection credentials
	ReadTimeout            int                  `json:"read_timeout"`             // web service read timeout in sec
	WriteTimeout           int                  `json:"write_timeout"`            // web service write timeout in sec
	MongoConnectionTimeout int                  `json:"mongo_connection_timeout"` // mongo connection timeout in sec
	ProdLockAccounts       []string             `json:"prod_lock_accounts"`       // rucio production accounts that lock files
	CollectionNames        MongoCollectionNames `json:"collection_names"`         // mongodb collection names
	BaseEndpoint           string               `json:"base_endpoint"`            // base_endpoint of web service
}

// MongoCollectionNames mongo collection names struct
type MongoCollectionNames struct {
	MainDatasets          string `json:"main_datasets"`             // main datasets collection name
	DetailedDatasets      string `json:"detailed_datasets"`         // detailed datasets collection name
	DatasetsInTapeAndDisk string `json:"datasets_in_tape_and_disk"` // datasets in both tape and disk collection name
	ShortUrl              string `json:"short_url"`                 // short_url collection name
	DatasourceTimestamp   string `json:"datasource_timestamp"`      // datasource_timestamp collection name
}

// String returns string representation of dbs Config
func (c *Configuration) String() string {
	data, err := json.Marshal(c)
	if err != nil {
		log.Println("[ERROR] fail to marshal configuration", err)
		return ""
	}
	return string(data)
}

// ParseConfig parses given configuration file and initialize Config object
func ParseConfig(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Println("unable to read config file", configFile, err)
		return err
	}
	err = json.Unmarshal(data, &Config)
	if err != nil {
		log.Println("unable to parse config file", configFile, err)
		return err
	}
	return nil
}
