package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// DataSourceTS struct which includes used data production time
type DataSourceTS struct {
	Id        primitive.ObjectID `bson:"_id"` // do not send in the json
	CreatedAt string             `bson:"created_at,omitempty" validate:"required"`
}
