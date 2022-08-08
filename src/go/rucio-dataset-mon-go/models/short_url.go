package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// ShortUrlRequest represents short url incoming request that will be fired from client
type ShortUrlRequest struct {
	Request    DataTableRequest       `json:"dt_request"`
	SavedState map[string]interface{} `json:"saved_state"`
}

// ShortUrl struct is used for key:value couples of unique id and datatables request
type ShortUrl struct {
	HashId     string                 `bson:"hash_id,omitempty" validate:"required"`
	Request    DataTableRequest       `bson:"dt_request,omitempty" validate:"required"`
	SavedState map[string]interface{} `bson:"saved_state"` // Saved state of datatables
}
