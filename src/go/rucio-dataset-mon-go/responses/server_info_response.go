package responses

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

// ServerInfo custom response struct for service information
type ServerInfo struct {
	ServiceVersion string `json:"version"`
	Server         string `json:"server"`
}
