package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Dataset struct which includes Rucio and DBS calculated values
type Dataset struct {
	Id                     primitive.ObjectID `bson:"_id"`
	RseType                string             `bson:"RSE_TYPE,omitempty" validate:"required"`
	Dataset                string             `bson:"dataset,omitempty" validate:"required"`
	LastAccess             string             `bson:"last_access"`
	LastAccessMs           int64              `bson:"last_access_ms"`
	MaxDatasetSizeInRsesGB float64            `bson:"max_dataset_size_in_rses(GB)"`
	MinDatasetSizeInRsesGB float64            `bson:"min_dataset_size_in_rses(GB)"`
	SumDatasetSizeInRsesGB float64            `bson:"sum_dataset_size_in_rses(GB)"`
	RSEs                   string             `bson:"RSE(s)"`
}
