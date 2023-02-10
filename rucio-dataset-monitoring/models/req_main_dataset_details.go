package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// MainDatasetDetailsRequest accepts dataset and its RSE type
type MainDatasetDetailsRequest struct {
	Dataset string `json:"dataset" validate:"required" binding:"required"` // dataset name
	Type    string `json:"type" validate:"required" binding:"required"`    // RSE type name
}
