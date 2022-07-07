package models

// Dataset struct which includes Rucio and DBS calculated values
type Dataset struct {
	RseType    string  `bson:"RseType,omitempty" validate:"required"`
	Dataset    string  `bson:"Dataset,omitempty" validate:"required"`
	LastAccess string  `bson:"LastAccess"`
	Max        float64 `bson:"Max"`
	Min        float64 `bson:"Min"`
	Avg        float64 `bson:"Avg"`
	Sum        float64 `bson:"Sum"`
	RSEs       string  `bson:"RSEs"`
}
