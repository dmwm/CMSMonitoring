package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// EachRseDetailsRequest accepts dataset and its RSE type
type EachRseDetailsRequest struct {
	Dataset string `json:"dataset" validate:"required" binding:"required"` // dataset name
	Type    string `json:"type"`                                           // RSE type name
}
