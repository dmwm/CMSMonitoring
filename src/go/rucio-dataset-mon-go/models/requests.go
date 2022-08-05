package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// SingleDetailedDatasetsRequest accepts dataset and its RSE type
type SingleDetailedDatasetsRequest struct {
	Dataset string `json:"dataset" validate:"required" binding:"required"` // dataset name
	Type    string `json:"type" validate:"required" binding:"required"`    // RSE type name
}

// SearchBuilderRequest datatables search builder request format
type SearchBuilderRequest struct {
	Criteria []SingleCriteria `json:"criteria"`
	// There are  "OR" and "AND" options.
	Logic string `json:"logic"`
}

type ShortUrlRequest struct {
	Request    DataTableCustomRequest `json:"dt_request"`
	SavedState map[string]interface{} `json:"saved_state"`
}
