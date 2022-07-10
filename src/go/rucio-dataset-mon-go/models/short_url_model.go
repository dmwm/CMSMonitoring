package models

// ShortUrl struct is used for key:value couples of unique id and datatables request
type ShortUrl struct {
	HashId     string                 `bson:"hash_id,omitempty" validate:"required"`
	Request    DataTableCustomRequest `bson:"dt_request,omitempty" validate:"required"`
	SavedState map[string]interface{} `bson:"saved_state"` // Saved state of datatables
}

type ShortUrlRequest struct {
	Request    DataTableCustomRequest `json:"dt_request"`
	SavedState map[string]interface{} `json:"saved_state"`
}
