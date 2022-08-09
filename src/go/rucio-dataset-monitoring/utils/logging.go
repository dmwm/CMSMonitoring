package utils

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"log"
)

// Verbose defines verbosity, user given parameter, default 0: INFO logs
var Verbose int

// InfoLogV0 prints logs with verbosity >= 0
func InfoLogV0(s string, v ...any) {
	if Verbose >= 0 {
		log.Printf("[INFO] "+s, v)
	}
}

// InfoLogV1 prints logs with verbosity >= 1
func InfoLogV1(s string, v ...any) {
	if Verbose >= 1 {
		log.Printf("[INFO] "+s, v)
	}
}

// InfoLogV2 prints logs with verbosity >= 2
func InfoLogV2(s string, v ...any) {
	if Verbose >= 2 {
		log.Printf("[DEBUG] "+s, v)
	}
}

// WarnLog prints warning logs
func WarnLog(s string, v ...any) {
	log.Printf("[WARN] "+s, v)
}

// ErrorLog prints error logs, but not exit the process like log.Fatal
func ErrorLog(s string, v ...any) {
	log.Printf("[ERROR] "+s, v)
}
