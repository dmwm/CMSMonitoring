package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// ShortUrlRequest represents short url incoming request that will be fired from client
type ShortUrlRequest struct {
	Request    DataTableRequest       `json:"dtRequest"`
	SavedState map[string]interface{} `json:"savedState"`
}

// ShortUrl struct is used for key:value couples of unique id and datatables request
type ShortUrl struct {
	HashId     string                 `bson:"hashId,omitempty" validate:"required"`
	Request    DataTableRequest       `bson:"dtRequest,omitempty" validate:"required"`
	SavedState map[string]interface{} `bson:"savedState"` // Saved state of datatables
}
