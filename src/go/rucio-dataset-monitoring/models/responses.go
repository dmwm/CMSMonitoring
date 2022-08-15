package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// DatatableBaseResponse represents JQuery DataTables response format
type DatatableBaseResponse struct {
	Draw            int         `json:"draw" validate:"required"`            // The value that came in DataTable request, same should be returned
	RecordsTotal    int64       `json:"recordsTotal" validate:"required"`    // Total records which will be showed in the footer
	RecordsFiltered int64       `json:"recordsFiltered" validate:"required"` // Filtered record count which will be showed in the footer
	Data            interface{} `json:"data" validate:"required"`            // Data
}

// ServerInfoResp custom response struct for service information
type ServerInfoResp struct {
	ServiceVersion string `json:"version"`
	Server         string `json:"server"`
}
