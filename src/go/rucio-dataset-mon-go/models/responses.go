package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// DatasetsResp datatable response struct. Full description https://datatables.net/manual/server-side#Returned-data
type DatasetsResp struct {
	Draw            int       `json:"draw" validate:"required"`            // The value that came in DataTable request, same should be returned
	RecordsTotal    int64     `json:"recordsTotal" validate:"required"`    // Total records which will be showed in the footer
	RecordsFiltered int64     `json:"recordsFiltered" validate:"required"` // Filtered record count which will be showed in the footer
	Data            []Dataset `json:"data" validate:"required"`            // Data itself that contains datasets
}

// DetailedDatasetsResp datatable response struct.
type DetailedDatasetsResp struct {
	Draw            int          `json:"draw" validate:"required"`            // The value that came in DataTable request, same should be returned
	RecordsTotal    int64        `json:"recordsTotal" validate:"required"`    // Total records which will be showed in the footer
	RecordsFiltered int64        `json:"recordsFiltered" validate:"required"` // Filtered record count which will be showed in the footer
	Data            []DetailedDs `json:"data" validate:"required"`            // Data itself that contains datasets
}

// ServerInfoResp custom response struct for service information
type ServerInfoResp struct {
	ServiceVersion string `json:"version"`
	Server         string `json:"server"`
}
