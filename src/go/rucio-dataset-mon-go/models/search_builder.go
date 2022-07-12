package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// Ref: https://datatables.net/extensions/searchbuilder/examples/
//   Datatables search builder does not support GoLang backend, but here we're :)
//   We're catching the JSON of SB from .getDetails() function and use it for our needs.

type SingleCriteria struct {
	// "contains" renamed as "regex" in the page and regex search will be applied
	Condition string `json:"condition"`
	// Column display name
	Data string `json:"data"`
	// Actual column name
	OrigData string `json:"origData"`
	// Data type that is comprehended by DT
	Type string `json:"type"`
	// List of user values
	Value []string `json:"value"`
}

// SearchBuilderRequest datatables search builder request format
type SearchBuilderRequest struct {
	Criteria []SingleCriteria `json:"criteria"`
	// There are  "OR" and "AND" options.
	Logic string `json:"logic"`
}
