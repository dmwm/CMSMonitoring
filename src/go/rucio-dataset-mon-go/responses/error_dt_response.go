package responses

// DtErrorResponse custom response struct, used in case of error
type DtErrorResponse struct {
	Status  int                    `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}
