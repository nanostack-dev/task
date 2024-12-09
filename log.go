package task

import (
	"fmt"
	"log"
)

// Logger is the interface for logging in the library.
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// DefaultLogger is the default logger implementation using the standard library log package.
type DefaultLogger struct {
	logger *log.Logger
}

// NewDefaultLogger creates a new DefaultLogger.
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{logger: log.Default()}
}

func (d *DefaultLogger) Debugf(format string, args ...interface{}) {
	d.logger.SetPrefix("DEBUG: ")
	d.logger.Println(fmt.Sprintf(format, args...))
}

func (d *DefaultLogger) Infof(format string, args ...interface{}) {
	d.logger.SetPrefix("INFO: ")
	d.logger.Println(fmt.Sprintf(format, args...))
}

func (d *DefaultLogger) Warnf(format string, args ...interface{}) {
	d.logger.SetPrefix("WARN: ")
	d.logger.Println(fmt.Sprintf(format, args...))
}

func (d *DefaultLogger) Errorf(format string, args ...interface{}) {
	d.logger.SetPrefix("ERROR: ")
	d.logger.Println(fmt.Sprintf(format, args...))
}

// Global logger instance
var logger Logger = NewDefaultLogger()

// SetLogger allows users to provide a custom logger.
func SetLogger(customLogger Logger) {
	logger = customLogger
}
