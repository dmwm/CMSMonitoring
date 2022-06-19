package models

// Dataset struct which includes Rucio and DBS calculated values
type Dataset struct {
	RseType                string  `bson:"RseType,omitempty" validate:"required"`
	Dataset                string  `bson:"Dataset,omitempty" validate:"required"`
	LastAccess             string  `bson:"LastAccess"`
	LastAccessMs           int64   `bson:"LastAccessMs"`
	MaxDatasetSizeInRsesGB float64 `bson:"MaxDatasetSizeInRsesGB"`
	MinDatasetSizeInRsesGB float64 `bson:"MinDatasetSizeInRsesGB"`
	AvgDatasetSizeInRsesGB float64 `bson:"AvgDatasetSizeInRsesGB"`
	SumDatasetSizeInRsesGB float64 `bson:"SumDatasetSizeInRsesGB"`
	RSEs                   string  `bson:"RSEs"`
}
