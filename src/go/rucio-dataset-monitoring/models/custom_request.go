package models

import (
	"encoding/json"
	"log"
)

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// CustomRequest represents custom fields that added for details page
type CustomRequest struct {
	Dataset    string   `json:"dataset"`
	Rse        string   `json:"rse"`
	Tier       string   `json:"tier"`
	RseCountry string   `json:"rseCountry"`
	RseKind    string   `json:"rseKind"`
	Accounts   []string `json:"accounts"`
	RseType    []string `json:"rseType"`
}

// String returns string representation of dbs SearchBuilderRequest
func (c *CustomRequest) String() string {
	data, err := json.Marshal(c)
	if err != nil {
		log.Println("[ERROR] fail to marshal CustomRequest", err)
		return ""
	}
	return string(data)
}
