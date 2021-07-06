package log

import (
	stdlog "log"
)

var debugMode = false

func EnableDebug(v bool) {
	debugMode = v
}

func Println(v ...interface{}) {
	stdlog.Println(v...)
}

func Printf(format string, v ...interface{}) {
	stdlog.Printf(format, v...)
}

func Fatalf(format string, v ...interface{}) {
	stdlog.Fatalf(format, v...)
}

func Debugf(format string, v ...interface{}) {
	if debugMode {
		Printf(format, v...)
	}
}
