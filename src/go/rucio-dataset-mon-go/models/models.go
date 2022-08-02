package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Dataset struct which includes Rucio and DBS calculated values
type Dataset struct {
	RseType      string  `bson:"RseType,omitempty" validate:"required"`
	Dataset      string  `bson:"Dataset,omitempty" validate:"required"`
	LastAccess   string  `bson:"LastAccess"`
	LastAccessMs int64   `bson:"LastAccessMs"`
	Max          float64 `bson:"Max"`
	Min          float64 `bson:"Min"`
	Avg          float64 `bson:"Avg"`
	Sum          float64 `bson:"Sum"`
	RSEs         string  `bson:"RSEs"`
}

// DetailedDataset struct which includes Rucio and DBS calculated values for detailed datasets info
type DetailedDataset struct {
	Type       string  `bson:"Type" validate:"required"`
	Dataset    string  `bson:"Dataset,omitempty" validate:"required"`
	RSE        string  `bson:"RSE,omitempty" validate:"required"`
	Tier       string  `bson:"Tier" validate:"required"`
	C          string  `bson:"C" validate:"required"` // Country
	RseKind    string  `bson:"RseKind" validate:"required"`
	SizeBytes  int64   `json:"SizeBytes"`
	LastAcc    string  `bson:"LastAcc"`    // Last access to dataset in ISO8601 format
	LastAccMs  int64   `bson:"LastAccMs"`  // Last access to dataset in unix ts
	Fpct       float64 `bson:"Fpct"`       // File percentage over total files of dataset definition
	Fcnt       int64   `bson:"Fcnt"`       // File count of the dataset in the RSE
	TotFcnt    int64   `bson:"TotFcnt"`    // File count of the dataset in definition
	AccFcnt    int64   `bson:"AccFcnt"`    // Accessed file count of dataset in the RSE
	ProdLckCnt int64   `bson:"ProdLckCnt"` // Count of files, locked by production accounts
	OthLckCnt  int64   `bson:"OthLckCnt"`  // Count of files, locked by non-production accounts
	ProdAccts  string  `bson:"ProdAccts"`  // Production accounts that locked the files
}

// DetailedDs struct which includes Rucio and DBS calculated values for detailed datasets info
type DetailedDs struct {
	Id         primitive.ObjectID `bson:"-" json:"-"` // do not send in the json
	Type       string             `bson:"Type,omitempty" validate:"required"`
	Dataset    string             `bson:"Dataset,omitempty" validate:"required"`
	RSE        string             `bson:"RSE,omitempty" validate:"required"`
	Tier       string             `bson:"Tier,omitempty" validate:"required"`
	C          string             `bson:"C,omitempty" validate:"required"` // Country
	RseKind    string             `bson:"RseKind,omitempty" validate:"required"`
	SizeBytes  int64              `json:"SizeBytes,omitempty"`
	LastAcc    string             `bson:"LastAcc"`    // Last access to dataset in ISO8601 format
	LastAccMs  int64              `bson:"LastAccMs"`  // Last access to dataset in unix ts
	Fpct       float64            `bson:"Fpct"`       // File percentage over total files of dataset definition
	Fcnt       int64              `bson:"Fcnt"`       // File count of the dataset in the RSE
	TotFcnt    int64              `bson:"TotFcnt"`    // File count of the dataset in definition
	AccFcnt    int64              `bson:"AccFcnt"`    // Accessed file count of dataset in the RSE
	ProdLckCnt int64              `bson:"ProdLckCnt"` // Count of files, locked by production accounts
	OthLckCnt  int64              `bson:"OthLckCnt"`  // Count of files, locked by non-production accounts
	ProdAccts  string             `bson:"ProdAccts"`  // Production accounts that locked the files
}

// DataSourceTS struct contains data production time means; alas hadoop dumps time-stamp
type DataSourceTS struct {
	Id        primitive.ObjectID `bson:"_id"` // do not send in the json
	CreatedAt string             `bson:"created_at,omitempty" validate:"required"`
}

// SingleCriteria condition object of SearchBuilder
//     Ref: https://datatables.net/extensions/searchbuilder/examples/
//     Datatables search builder does not support GoLang backend, but here we're :)
//     We're catching the JSON of SB from .getDetails() function and use it for our needs.
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

// ShortUrl struct is used for key:value couples of unique id and datatables request
type ShortUrl struct {
	HashId     string                 `bson:"hash_id,omitempty" validate:"required"`
	Request    DataTableCustomRequest `bson:"dt_request,omitempty" validate:"required"`
	SavedState map[string]interface{} `bson:"saved_state"` // Saved state of datatables
}
