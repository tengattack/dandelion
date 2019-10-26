package client

import (
	"log"
	"os"
)

var std Logger

// Logger for dandelion client
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// defaultLogger logs info & error
type defaultLogger struct {
	accessLog *log.Logger
	errorLog  *log.Logger
}

func (l *defaultLogger) Debugf(format string, args ...interface{}) {
}

func (l *defaultLogger) Infof(format string, args ...interface{}) {
	l.accessLog.Printf(format, args...)
}

func (l *defaultLogger) Errorf(format string, args ...interface{}) {
	l.errorLog.Printf(format, args...)
}

func newDefaultLogger() Logger {
	return &defaultLogger{
		accessLog: log.New(os.Stdout, "", log.LstdFlags),
		errorLog:  log.New(os.Stderr, "", log.LstdFlags),
	}
}

func init() {
	std = newDefaultLogger()
	SetLogger(nil)
}

// SetLogger sets client logger
func SetLogger(lg Logger) {
	if lg == nil {
		clientLogger = std
		return
	}
	clientLogger = lg
}
