package responses

// ErrorResponseStruct custom response struct, used in case of error
type ErrorResponseStruct struct {
	Status  int               `json:"status"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
}
