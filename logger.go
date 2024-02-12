package queue

import (
	"log"
	"os"

	"github.com/golang-queue/queue/core"
)

// NewLogger for simple logger.
func NewLogger() core.Logger {
	return defaultLogger{
		infoLogger:  log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile),
		debugLogger: log.New(os.Stdout, "DEBGU: ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLogger: log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
		fatalLogger: log.New(os.Stderr, "FATAL: ", log.Ldate|log.Ltime|log.Lshortfile),
		warnLogger:  log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

type defaultLogger struct {
	infoLogger  *log.Logger
	debugLogger *log.Logger
	errorLogger *log.Logger
	fatalLogger *log.Logger
	warnLogger  *log.Logger
}

func (l defaultLogger) Debug(args ...interface{}) {
	if len(args) == 1 {
		l.debugLogger.Printf(args[0].(string))
	} else if len(args) > 1 {
		l.debugLogger.Printf(args[0].(string), args[1:]...)
	}
}

func (l defaultLogger) Info(args ...interface{}) {
	if len(args) == 1 {
		l.infoLogger.Printf(args[0].(string))
	} else if len(args) > 1 {
		l.infoLogger.Printf(args[0].(string), args[1:]...)
	}
}

func (l defaultLogger) Error(args ...interface{}) {
	if len(args) == 1 {
		l.errorLogger.Printf(args[0].(string))
	} else if len(args) > 1 {
		l.errorLogger.Printf(args[0].(string), args[1:]...)
	}
}

func (l defaultLogger) Fatal(args ...interface{}) {
	if len(args) == 1 {
		l.fatalLogger.Printf(args[0].(string))
	} else if len(args) > 1 {
		l.fatalLogger.Printf(args[0].(string), args[1:]...)
	}
}

func (l defaultLogger) Warn(args ...interface{}) {
	if len(args) == 1 {
		l.warnLogger.Printf(args[0].(string))
	} else if len(args) > 1 {
		l.warnLogger.Printf(args[0].(string), args[1:]...)
	}
}

// NewEmptyLogger for simple logger.
func NewEmptyLogger() core.Logger {
	return emptyLogger{}
}

// EmptyLogger no meesgae logger
type emptyLogger struct{}

func (l emptyLogger) Info(args ...interface{})  {}
func (l emptyLogger) Error(args ...interface{}) {}
func (l emptyLogger) Fatal(args ...interface{}) {}
func (l emptyLogger) Warn(args ...interface{})  {}
func (l emptyLogger) Debug(args ...interface{}) {}
