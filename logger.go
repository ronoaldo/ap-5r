package main

import (
	"fmt"
	"log"
)

// Logger is a guild-prefixed logger for info and error messages.
type Logger struct {
	Guild string
}

func (l *Logger) Printf(m string, args ...interface{}) {
	l.Infof(m, args...)
}

func (l *Logger) Infof(m string, args ...interface{}) {
	log.Printf(fmt.Sprintf("INFO  [%s] %s", l.Guild, m), args...)
}

func (l *Logger) Errorf(m string, args ...interface{}) {
	log.Printf(fmt.Sprintf("ERROR [%s] %s", l.Guild, m), args...)
}

func (l *Logger) Fatalf(m string, args ...interface{}) {
	log.Fatalf(fmt.Sprintf("FATAL [%s] %s", l.Guild, m), args...)
}
