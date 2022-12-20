package models

import (
	"encoding/json"
	"log"
)

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// DataTable sends specific request json to ajax=>url endpoint.
//    - It is basically a nested json and can be organised with below structs.
//    - Reference: comment in https://datatables.net/forums/discussion/68295/go-golang-unmarshal-json-sent-by-post-method
//    - See example request in `rucio-dataset-monitoring/README.md`

// DataTableRequest main ajax request that come from DataTable, which includes user inputs
//
//	For full field descriptions, please see https://datatables.net/manual/server-side#Sent-parameters
type DataTableRequest struct {
	Draw                 int                  `json:"draw" validate:"required" binding:"required"`    // Just a counter that should be return exactly in the response
	Columns              []DTReqColumn        `json:"columns" validate:"required" binding:"required"` // Includes user input for columns (like search text for the column)
	Length               int64                `json:"length"`                                         // Number of records that the table can display in the current draw.
	Orders               []DTReqOrder         `json:"order"`                                          //
	Start                int64                `json:"start"`                                          //
	SearchBuilderRequest SearchBuilderRequest `json:"searchBuilderRequest"`                           // SearchBuilderRequest (custom addition to DataTables request object)
}

// DTReqSearch represents main search text which client entered and can be regex or not.
//
//	In default in this service, all search text will be behaved as "REGEX" instead of fuzzy search.
//	TODO separate regex and fuzzy search by providing option in the frontend
type DTReqSearch struct {
	Regex bool   `json:"regex"`
	Value string `json:"value"`
}

// DTReqColumn represents client column selections like; sort order, search text for that column
type DTReqColumn struct {
	Data       string      `json:"data"`       // Column name
	Name       string      `json:"name"`       // Column name to be used in page if it is different from source data
	Searchable bool        `json:"searchable"` //
	Orderable  bool        `json:"orderable"`  //
	Search     DTReqSearch `json:"search"`     // If client requested search against individual column
}

// DTReqOrder represents the selected column direction.
type DTReqOrder struct {
	Column int    `json:"column"`                       // Column index number in the columns list (order not changes)
	Dir    string `json:"dir" binding:"oneof=asc desc"` // asc or desc
}

// String returns string representation of DataTableRequest
func (r *DataTableRequest) String() string {
	data, err := json.Marshal(r)
	if err != nil {
		log.Println("[ERROR] fail to marshal DataTableRequest", err)
		return ""
	}
	return string(data)
}
