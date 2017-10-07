package main

import (
	"fmt"
	"log"
)

// Logger is a guild-prefixed logger for info and error messages.
type Logger struct {
	Guild string
}

// Printf formats the message with args and logs with INFO level.
func (l *Logger) Printf(m string, args ...interface{}) {
	l.Infof(m, args...)
}

// Infof formats the message with args and logs with INFO level.
func (l *Logger) Infof(m string, args ...interface{}) {
	log.Printf(fmt.Sprintf("INFO  [%s] %s", l.Guild, m), args...)
}

// Errorf formats the message with args and logs with ERROR level.
func (l *Logger) Errorf(m string, args ...interface{}) {
	log.Printf(fmt.Sprintf("ERROR [%s] %s", l.Guild, m), args...)
}

// Fatalf formats the message and calls log.Fatalf with a FATAL level.
// Program will terminate after this call.
func (l *Logger) Fatalf(m string, args ...interface{}) {
	log.Fatalf(fmt.Sprintf("FATAL [%s] %s", l.Guild, m), args...)
}
