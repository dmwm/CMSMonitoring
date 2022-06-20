package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// DetailedDs struct which includes Rucio and DBS calculated values for detailed datasets info
type DetailedDs struct {
	Id         primitive.ObjectID `bson:"-" json:"-"` // do not send in the json
	Type       string             `bson:"Type,omitempty" validate:"required"`
	Dataset    string             `bson:"Dataset,omitempty" validate:"required"`
	RSE        string             `bson:"RSE,omitempty" validate:"required"`
	Tier       string             `bson:"Tier,omitempty" validate:"required"`
	C          string             `bson:"C,omitempty" validate:"required"` // Country
	RseKind    string             `bson:"RseKind,omitempty" validate:"required"`
	SizeBytes  int64              `json:"SizeBytes,omitempty"`
	LastAcc    string             `bson:"LastAcc"`    // Last access to dataset in ISO8601 format
	LastAccMs  int64              `bson:"LastAccMs"`  // Last access to dataset in unix ts
	Fpct       float64            `bson:"Fpct"`       // File percentage over total files of dataset definition
	Fcnt       int64              `bson:"Fcnt"`       // File count of the dataset in the RSE
	TotFcnt    int64              `bson:"TotFcnt"`    // File count of the dataset in definition
	AccFcnt    int64              `bson:"AccFcnt"`    // Accessed file count of dataset in the RSE
	ProdLckCnt int64              `bson:"ProdLckCnt"` // Count of files, locked by production accounts
	OthLckCnt  int64              `bson:"OthLckCnt"`  // Count of files, locked by non-production accounts
	ProdAccts  string             `bson:"ProdAccts"`  // Production accounts that locked the files
}

// DetailedDs struct which includes Rucio and DBS calculated values for detailed datasets info
type RawDetailedDs struct {
	Id         primitive.ObjectID `bson:"_id" json:"id,omitempty"`
	Type       string             `bson:"Type,omitempty" validate:"required"`
	Dataset    string             `bson:"Dataset,omitempty" validate:"required"`
	RSE        string             `bson:"RSE,omitempty" validate:"required"`
	Tier       string             `bson:"Tier,omitempty" validate:"required"`
	C          string             `bson:"C,omitempty" validate:"required"` // Country
	RseKind    string             `bson:"RseKind,omitempty" validate:"required"`
	SizeBytes  int64              `json:"SizeBytes,omitempty"`
	LastAcc    string             `bson:"LastAcc"`    // Last access to dataset in ISO8601 format
	LastAccMs  int64              `bson:"LastAccMs"`  // Last access to dataset in unix ts
	Fpct       float64            `bson:"Fpct"`       // File percentage over total files of dataset definition
	Fcnt       int64              `bson:"Fcnt"`       // File count of the dataset in the RSE
	TotFcnt    int64              `bson:"TotFcnt"`    // File count of the dataset in definition
	AccFcnt    int64              `bson:"AccFcnt"`    // Accessed file count of dataset in the RSE
	ProdLckCnt int64              `bson:"ProdLckCnt"` // Count of files, locked by production accounts
	OthLckCnt  int64              `bson:"OthLckCnt"`  // Count of files, locked by non-production accounts
	ProdAccts  string             `bson:"ProdAccts"`  // Production accounts that locked the files
}
