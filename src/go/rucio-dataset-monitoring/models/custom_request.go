package models

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
