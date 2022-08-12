package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"time"
)

// Epoch type used for converting integer unix time to string date format with some customization in custom MarshalJSON
type Epoch int64

// dateLayout LastAccess millisecond to date string conversion format
const dateLayout = "2006-01-02"

// Dataset struct which includes Rucio and DBS calculated values
type Dataset struct {
	RseType      string  `bson:"RseType,omitempty" validate:"required"`
	Dataset      string  `bson:"Dataset,omitempty" validate:"required"`
	LastAccess   Epoch   `bson:"LastAccess"`
	Max          float64 `bson:"Max"`
	Min          float64 `bson:"Min"`
	Avg          float64 `bson:"Avg"`
	Sum          float64 `bson:"Sum"`
	RealSize     float64 `bson:"RealSize"`
	TotalFileCnt int64   `bson:"TotalFileCnt"`
	RSEs         string  `bson:"RSEs"`
}

// MarshalJSON marshal integer unix time to date string in YYYY-MM-DD format
func (t Epoch) MarshalJSON() ([]byte, error) {
	var strDate string
	if int64(t) > 0 {
		strDate = time.Unix(int64(t), 0).Format(dateLayout)
	} else {
		// If LastAccess is nil
		strDate = "NOT-ACCESSED"
	}
	out := []byte(`"` + strDate + `"`)
	return out, nil
}
