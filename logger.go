package queue

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// Logger interface is used throughout gorush
type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Info(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
}

// NewLogger for simple logger.
func NewLogger() Logger {
	return defaultLogger{
		infoLogger:  log.New(os.Stderr, "INFO: ", log.Ldate|log.Ltime),
		errorLogger: log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime),
		fatalLogger: log.New(os.Stderr, "FATAL: ", log.Ldate|log.Ltime),
	}
}

type defaultLogger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	fatalLogger *log.Logger
}

func (l defaultLogger) Infof(format string, args ...interface{}) {
	l.infoLogger.Printf(format, args...)
}

func (l defaultLogger) logWithCallerf(logger *log.Logger, format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		shortFile := filepath.Base(file)
		logger.Printf("%s:%d: %s", shortFile, line, fmt.Sprintf(format, args...))
		return
	}
	logger.Printf(format, args...)
}

func (l defaultLogger) logWithCaller(logger *log.Logger, args ...interface{}) {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		shortFile := filepath.Base(file)
		logger.Printf("%s:%d: %s", shortFile, line, fmt.Sprint(args...))
		return
	}
	logger.Println(fmt.Sprint(args...))
}

func (l defaultLogger) Errorf(format string, args ...interface{}) {
	l.logWithCallerf(l.errorLogger, format, args...)
}

func (l defaultLogger) Fatalf(format string, args ...interface{}) {
	l.logWithCallerf(l.fatalLogger, format, args...)
	os.Exit(1)
}

func (l defaultLogger) Info(args ...interface{}) {
	l.infoLogger.Println(fmt.Sprint(args...))
}

func (l defaultLogger) Error(args ...interface{}) {
	l.logWithCaller(l.errorLogger, args...)
}

func (l defaultLogger) Fatal(args ...interface{}) {
	l.logWithCaller(l.fatalLogger, args...)
	os.Exit(1)
}

// NewEmptyLogger for simple logger.
func NewEmptyLogger() Logger {
	return emptyLogger{}
}

// EmptyLogger no meesgae logger
type emptyLogger struct{}

func (l emptyLogger) Infof(format string, args ...interface{})  {}
func (l emptyLogger) Errorf(format string, args ...interface{}) {}
func (l emptyLogger) Fatalf(format string, args ...interface{}) {}
func (l emptyLogger) Info(args ...interface{})                  {}
func (l emptyLogger) Error(args ...interface{})                 {}
func (l emptyLogger) Fatal(args ...interface{})                 {}
