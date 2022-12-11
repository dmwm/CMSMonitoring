package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// SingleDetailedDatasetsRequest accepts dataset and its RSE type
type SingleDetailedDatasetsRequest struct {
	Dataset string `json:"dataset" validate:"required" binding:"required"` // dataset name
	Type    string `json:"type" validate:"required" binding:"required"`    // RSE type name
}
