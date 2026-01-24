// Package utils provides utility functions for the application.
package utils

import (
	"log"
	"os"
)

// Logger provides structured logging capabilities.
type Logger struct {
	info  *log.Logger
	warn  *log.Logger
	error *log.Logger
	debug *log.Logger
}

// NewLogger creates a new logger instance.
func NewLogger() *Logger {
	flags := log.LstdFlags | log.Lshortfile
	return &Logger{
		info:  log.New(os.Stdout, "[INFO] ", flags),
		warn:  log.New(os.Stdout, "[WARN] ", flags),
		error: log.New(os.Stderr, "[ERROR] ", flags),
		debug: log.New(os.Stdout, "[DEBUG] ", flags),
	}
}

// Info logs an info message.
func (l *Logger) Info(format string, v ...interface{}) {
	l.info.Printf(format, v...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, v ...interface{}) {
	l.warn.Printf(format, v...)
}

// Error logs an error message.
func (l *Logger) Error(format string, v ...interface{}) {
	l.error.Printf(format, v...)
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, v ...interface{}) {
	l.debug.Printf(format, v...)
}
