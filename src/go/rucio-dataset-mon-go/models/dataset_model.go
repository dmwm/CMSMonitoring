package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Dataset struct which includes Rucio and DBS calculated values
type Dataset struct {
	Id      primitive.ObjectID `json:"id,omitempty"`
	Dataset string             `json:"dataset,omitempty" validate:"required"`
	Rse     string             `json:"rse,omitempty" validate:"required"`
	Size    int64              `json:"size,omitempty" validate:"required"`
}
