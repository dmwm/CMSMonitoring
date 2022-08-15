package utils

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"log"
)

// Verbose defines verbosity, user given parameter, default 0: INFO logs
var Verbose int

// InfoLogV0 prints logs with verbosity >= 0
func InfoLogV0(format string, v ...interface{}) {
	if Verbose >= 0 {
		log.Printf("[INFO] "+format, v...)
	}
}

// InfoLogV1 prints logs with verbosity >= 1
func InfoLogV1(format string, v ...interface{}) {
	if Verbose >= 1 {
		log.Printf("[INFO] "+format, v...)
	}
}

// InfoLogV2 prints logs with verbosity >= 2
func InfoLogV2(format string, v ...interface{}) {
	if Verbose >= 2 {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// ErrorLog prints error logs, but not exit the process like log.Fatal
func ErrorLog(format string, v ...interface{}) {
	log.Printf("[ERROR]  "+format, v...)
}
