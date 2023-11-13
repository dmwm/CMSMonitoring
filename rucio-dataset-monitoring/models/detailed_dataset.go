package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"strings"
)

// StringArray type used for converting array with comma separated
type StringArray []string

// DetailedDataset struct which includes Rucio and DBS calculated values for detailed datasets info
type DetailedDataset struct {
	Type                 string      `bson:"Type" validate:"required"`
	Dataset              string      `bson:"Dataset,omitempty" validate:"required"`
	RSE                  string      `bson:"RSE,omitempty" validate:"required"`
	Tier                 string      `bson:"Tier" validate:"required"`
	C                    string      `bson:"C" validate:"required"` // Country
	RseKind              string      `bson:"RseKind" validate:"required"`
	SizeBytes            int64       `bson:"SizeBytes"`
	LastAccess           Epoch       `bson:"LastAccess"` // Last access to dataset in ISO8601 format
	LastCreate           Epoch       `bson:"LastCreate"` // Latest created at of the dataset in ISO8601 format
	IsFullyReplicated    bool        `bson:"IsFullyReplicated"`
	IsLocked             string      `bson:"IsLocked"`
	FilePercentage       float64     `bson:"FilePercentage"`
	FileCount            int64       `bson:"FileCount"` // File count of the dataset in the RSE
	AccessedFileCount    int64       `bson:"AccessedFileCount"`
	BlockCount           int64       `bson:"BlockCount"`
	ProdLockedBlockCount int64       `bson:"ProdLockedBlockCount"`
	ProdAccounts         StringArray `bson:"ProdAccounts"` // Production accounts that locked the files
	BlockRuleIDs         StringArray `bson:"BlockRuleIDs"`
}

// MarshalJSON marshal integer unix time to date string in YYYY-MM-DD format
func (t StringArray) MarshalJSON() ([]byte, error) {
	var out string
	if len(t) > 0 {
		out = strings.Join(t, ", ")
	}
	return []byte(`"` + out + `"`), nil
}
