package responses

// ServerInfo custom response struct for service information
type ServerInfo struct {
	ServiceVersion string `json:"version"`
	Server         string `json:"server"`
}
