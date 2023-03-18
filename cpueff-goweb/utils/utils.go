package utils

// Copyright (c) 2023 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strings"
)

// ErrorResponseStruct custom response struct, used in case of error
type ErrorResponseStruct struct {
	Status  int               `json:"status"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
	Request any               `json:"request"`
}

// ErrorResponse returns error response with given msg and error
func ErrorResponse(c *gin.Context, msg string, err error, req string) {
	log.Printf("[ERROR] %s %s %#v", msg, err, req)
	c.JSON(http.StatusBadRequest,
		ErrorResponseStruct{
			Status:  http.StatusBadRequest,
			Message: msg,
			Data:    map[string]string{"data": err.Error()},
			Request: req,
		})
	return
}

// ConvertOrderEnumToMongoInt converts DataTable enums ("asc" and "desc") to Mongo sorting integer definitions (1,-1)
func ConvertOrderEnumToMongoInt(dir string) int {
	switch strings.ToLower(dir) {
	case "asc":
		return 1
	case "desc":
		return -1
	default:
		return 0
	}
}
