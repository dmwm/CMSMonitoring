package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// DataTable sends specific request json to ajax=>url endpoint.
//    - It is basically a nested json and can be organised with below structs.
//    - Reference: comment in https://datatables.net/forums/discussion/68295/go-golang-unmarshal-json-sent-by-post-method
//    - See example request in `rucio-dataset-mon-go/README.md`

// DataTableRequest main ajax request that come from DataTable, which includes user inputs
//     For full field descriptions, please see https://datatables.net/manual/server-side#Sent-parameters
type DataTableRequest struct {
	Draw    int           `json:"draw" validate:"required" binding:"required"`    // Just a counter that should be return exactly in the response
	Columns []DTReqColumn `json:"columns" validate:"required" binding:"required"` // Includes user input for columns (like search text for the column)
	Length  int64         `json:"length"`                                         // Number of records that the table can display in the current draw.
	Orders  []DTReqOrder  `json:"order"`                                          //
	Search  DTReqSearch   `json:"search"`                                         //
	Start   int64         `json:"start"`                                          //
}

// DataTableSearchBuilderRequest customized ajax request that come from DataTable
type DataTableSearchBuilderRequest struct {
	Draw          int                  `json:"draw" validate:"required" binding:"required"`                     // Just a counter that should be return exactly in the response
	Columns       []DTReqColumn        `json:"columns" validate:"required" binding:"required"`                  // Includes user input for columns (like search text for the column)
	Length        int64                `json:"length"`                                                          // Number of records that the table can display in the current draw.
	Orders        []DTReqOrder         `json:"order"`                                                           //
	Search        DTReqSearch          `json:"search"`                                                          //
	Start         int64                `json:"start"`                                                           //
	SearchBuilder SearchBuilderRequest `json:"search_builder,omitempty" validate:"required" binding:"required"` // SearchBuilder
}

// DataTableCustomRequest customized ajax request that come from DataTable
type DataTableCustomRequest struct {
	Draw    int           `json:"draw" validate:"required" binding:"required"`    // Just a counter that should be return exactly in the response
	Columns []DTReqColumn `json:"columns" validate:"required" binding:"required"` // Includes user input for columns (like search text for the column)
	Length  int64         `json:"length"`                                         // Number of records that the table can display in the current draw.
	Orders  []DTReqOrder  `json:"order"`                                          //
	Search  DTReqSearch   `json:"search"`                                         //
	Start   int64         `json:"start"`                                          //
	Custom  Custom        `json:"custom"`                                         // Custom
}

// DTReqSearch represents main search text which client entered and can be regex or not.
//     In default in this service, all search text will be behaved as "REGEX" instead of fuzzy search.
//     TODO separate regex and fuzzy search by providing option in the frontend
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

// Custom represents custom fields that added for details page
type Custom struct {
	Dataset    string   `json:"dataset"`
	Rse        string   `json:"rse"`
	Tier       string   `json:"tier"`
	RseCountry string   `json:"rseCountry"`
	RseKind    string   `json:"rseKind"`
	Accounts   []string `json:"accounts"`
	RseType    []string `json:"rseType"`
}