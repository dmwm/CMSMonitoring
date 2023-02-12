package utils

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"io"
	"log"
	"os"
	"time"
)

// Verbose defines verbosity, user given parameter, default 0: INFO logs
var Verbose int

// var logger = log.New(&writer{os.Stdout, "[2006-01-02T15:04:05+01:00] "}, "", 0)
var tformat = "[" + time.RFC3339 + "] "
var logger = log.New(&writer{os.Stdout, tformat}, "", 0)

type writer struct {
	io.Writer
	timeFormat string
}

func (w writer) Write(b []byte) (n int, err error) {
	return w.Writer.Write(append([]byte(time.Now().Format(w.timeFormat)), b...))
}

// InfoLogV0 prints logs with verbosity >= 0
func InfoLogV0(format string, v ...interface{}) {
	if Verbose >= 0 {
		logger.Printf("[INFO] "+format, v...)
	}
}

// InfoLogV1 prints logs with verbosity >= 1
func InfoLogV1(format string, v ...interface{}) {
	if Verbose >= 1 {
		logger.Printf("[INFO] "+format, v...)
	}
}

// InfoLogV2 prints logs with verbosity >= 2
func InfoLogV2(format string, v ...interface{}) {
	if Verbose >= 2 {
		logger.Printf("[DEBUG] "+format, v...)
	}
}

// ErrorLog prints error logs, but not exit the process like log.Fatal
func ErrorLog(format string, v ...interface{}) {
	logger.Printf("[ERROR]  "+format, v...)
}
