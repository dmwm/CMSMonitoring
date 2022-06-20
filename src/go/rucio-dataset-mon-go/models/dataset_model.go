package models

// Dataset struct which includes Rucio and DBS calculated values
type Dataset struct {
	RseType      string  `bson:"RseType,omitempty" validate:"required"`
	Dataset      string  `bson:"Dataset,omitempty" validate:"required"`
	LastAccess   string  `bson:"LastAccess"`
	LastAccessMs int64   `bson:"LastAccessMs"`
	MaxTB        float64 `bson:"MaxTB"`
	MinTB        float64 `bson:"MinTB"`
	AvgTB        float64 `bson:"AvgTB"`
	SumTB        float64 `bson:"SumTB"`
	RSEs         string  `bson:"RSEs"`
}
