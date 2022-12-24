package models

import (
	"encoding/json"
	"log"
	"strings"
)

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// SearchBuilderRequest datatables search builder request format
type SearchBuilderRequest struct {
	Criteria     []SingleCriteria `json:"criteria,omitempty"`
	Logic        string           `json:"logic,omitempty"`        // There are  "OR" and "AND" options.
	InputDataset string           `json:"inputDataset,omitempty"` // Main search bar entry for dataset
}

// SingleCriteria condition object of SearchBuilderRequest
//
//	Ref: https://datatables.net/extensions/searchbuilder/examples/
//	Datatables search builder does not support GoLang backend, but here we're :)
//	We're catching the JSON of SB from .getDetails() function and use it for our needs.
type SingleCriteria struct {
	// "contains" renamed as "regex" in the page and regex search will be applied
	Condition string `json:"condition,omitempty"`
	// Column display name
	Data string `json:"data,omitempty"`
	// Actual column name
	OrigData string `json:"origData,omitempty"`
	// Data type that is comprehended by DT
	Type string `json:"type,omitempty"`
	// List of user values
	Value []string `json:"value,omitempty"`
}

func (r *SearchBuilderRequest) GetPrettyURL() string {
	var prettyUrl string
	prettyUrl += strings.ToLower(r.Logic) + "+"
	// If dataset search bar is filled
	if r.InputDataset != "" {
		prettyUrl += "dataset:" + r.InputDataset + "+"
	}
	for _, c := range r.Criteria {
		prettyUrl = prettyUrl +
			c.OrigData + "." +
			strings.ToLower(c.Condition) + ":" +
			strings.Join(c.Value[:], ",") +
			"++"
	}
	prettyUrl = strings.Replace(prettyUrl, "/", "_", -1)
	prettyUrl = strings.Replace(prettyUrl, " ", "", -1)
	return prettyUrl
}

// String returns string representation of dbs SearchBuilderRequest
func (r *SearchBuilderRequest) String() string {
	data, err := json.Marshal(r)
	if err != nil {
		log.Println("[ERROR] fail to marshal SearchBuilderRequest", err)
		return ""
	}
	return string(data)
}
