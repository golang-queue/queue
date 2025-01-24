package queue

import (
	"fmt"
	"log"
	"os"
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

func (l defaultLogger) logWithCallerf(logger *log.Logger, format string, args ...interface{}) {
	stack := stack(3)
	logger.Printf("%s%s", stack, fmt.Sprintf(format, args...))
}

func (l defaultLogger) logWithCaller(logger *log.Logger, args ...interface{}) {
	stack := stack(3)
	logger.Println(stack, fmt.Sprint(args...))
}

func (l defaultLogger) Infof(format string, args ...interface{}) {
	l.infoLogger.Printf(format, args...)
}

func (l defaultLogger) Errorf(format string, args ...interface{}) {
	l.errorLogger.Printf(format, args...)
}

func (l defaultLogger) Fatalf(format string, args ...interface{}) {
	l.logWithCallerf(l.fatalLogger, format, args...)
}

func (l defaultLogger) Info(args ...interface{}) {
	l.infoLogger.Println(fmt.Sprint(args...))
}

func (l defaultLogger) Error(args ...interface{}) {
	l.errorLogger.Println(fmt.Sprint(args...))
}

func (l defaultLogger) Fatal(args ...interface{}) {
	l.logWithCaller(l.fatalLogger, args...)
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
