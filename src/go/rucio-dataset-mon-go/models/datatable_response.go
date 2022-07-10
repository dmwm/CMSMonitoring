package models

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// DatatableDatasetsResponse datatable response struct. Full description https://datatables.net/manual/server-side#Returned-data
type DatatableDatasetsResponse struct {
	Draw            int       `json:"draw" validate:"required"`            // The value that came in DataTable request, same should be returned
	RecordsTotal    int64     `json:"recordsTotal" validate:"required"`    // Total records which will be showed in the footer
	RecordsFiltered int64     `json:"recordsFiltered" validate:"required"` // Filtered record count which will be showed in the footer
	Data            []Dataset `json:"data" validate:"required"`            // Data itself that contains datasets
}

// DatatableDetailedDsResponse datatable response struct.
type DatatableDetailedDsResponse struct {
	Draw            int          `json:"draw" validate:"required"`            // The value that came in DataTable request, same should be returned
	RecordsTotal    int64        `json:"recordsTotal" validate:"required"`    // Total records which will be showed in the footer
	RecordsFiltered int64        `json:"recordsFiltered" validate:"required"` // Filtered record count which will be showed in the footer
	Data            []DetailedDs `json:"data" validate:"required"`            // Data itself that contains datasets
}
