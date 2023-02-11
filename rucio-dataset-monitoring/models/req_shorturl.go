package models

import (
	"encoding/json"
	"log"
)

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// ShortUrlRequest represents short url incoming request that will be fired from client
type ShortUrlRequest struct {
	Page       string                 `json:"page"`
	Request    DataTableRequest       `json:"dtRequest"`
	SavedState map[string]interface{} `json:"savedState"`
}

// ShortUrl struct is used for key:value couples of unique id and datatables request
type ShortUrl struct {
	HashId     string                 `bson:"hashId,omitempty" validate:"required"`
	Page       string                 `bson:"page" validate:"required"` // Page name: main, detailed, etc.
	Request    DataTableRequest       `bson:"dtRequest,omitempty" validate:"required"`
	SavedState map[string]interface{} `bson:"savedState"` // Saved state of datatables
}

// String returns string representation of ShortUrlRequest
func (r *ShortUrlRequest) String() string {
	data, err := json.Marshal(r)
	if err != nil {
		log.Println("[ERROR] fail to marshal ShortUrlRequest", err)
		return ""
	}
	return string(data)
}
