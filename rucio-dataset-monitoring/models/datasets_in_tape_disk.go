package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// DatasetInTapeDisk struct of datasets in both tape and disk, collection: dataset_in_tape_disk
type DatasetInTapeDisk struct {
	Dataset                     string      `bson:"Dataset,omitempty" validate:"required"`
	MaxSize                     int64       `bson:"MaxSize"` // Max size of dataset in listed RSEs
	TapeFullyReplicatedRseCount int64       `bson:"TapeFullyReplicatedRseCount"`
	DiskFullyReplicatedRseCount int64       `bson:"DiskFullyReplicatedRseCount"`
	TapeFullyLockedRseCount     int64       `bson:"TapeFullyLockedRseCount"`
	DiskFullyLockedRseCount     int64       `bson:"DiskFullyLockedRseCount"`
	TapeRseCount                int64       `bson:"TapeRseCount"`
	DiskRseCount                int64       `bson:"DiskRseCount"`
	TapeRseSet                  StringArray `bson:"TapeRseSet"` // Tape RSEs array
	DiskRseSet                  StringArray `bson:"DiskRseSet"` // Disk RSEs array
}
