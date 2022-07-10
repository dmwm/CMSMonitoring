package utils

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

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

// MiddlewareReqHandler handles CORS and HTTP request settings for the context router
func MiddlewareReqHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		//c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// ErrorResponse returns error response with given msg and error
func ErrorResponse(c *gin.Context, msg string, err error, req string) {
	log.Printf("[ERROR] %s %s %#v", msg, err, req)
	c.AbortWithStatusJSON(http.StatusBadRequest,
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
