package queue

import (
	"fmt"
	"log"
	"os"
)

// Logger defines the interface for logging queue events, errors, and fatal conditions.
// The queue uses this interface to report:
//   - Info: Normal operations (shutdown, retry attempts)
//   - Error: Recoverable errors (task failures, runtime errors)
//   - Fatal: Panics and critical failures (includes stack traces)
//
// Implement this interface to integrate with custom logging systems (logrus, zap, etc.).
type Logger interface {
	// Infof logs formatted informational messages.
	Infof(format string, args ...any)

	// Errorf logs formatted error messages.
	Errorf(format string, args ...any)

	// Fatalf logs formatted fatal errors with stack trace information.
	// Used for panics and critical failures.
	Fatalf(format string, args ...any)

	// Info logs informational messages.
	Info(args ...any)

	// Error logs error messages.
	Error(args ...any)

	// Fatal logs fatal errors with stack trace information.
	// Used for panics and critical failures.
	Fatal(args ...any)
}

// NewLogger creates a standard logger that writes to stderr with timestamps.
// This is the default logger used by queues unless overridden with WithLogger.
//
// Log format:
//   - INFO messages: Simple timestamped output
//   - ERROR messages: Simple timestamped output
//   - FATAL messages: Includes stack trace with file:line information
//
// Use cases:
//   - Development and debugging
//   - Simple production deployments without structured logging
//   - When detailed error context is needed
func NewLogger() Logger {
	return defaultLogger{
		infoLogger:  log.New(os.Stderr, "INFO: ", log.Ldate|log.Ltime),
		errorLogger: log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime),
		fatalLogger: log.New(os.Stderr, "FATAL: ", log.Ldate|log.Ltime),
	}
}

// defaultLogger is the standard logger implementation using Go's log package.
// It provides timestamped output to stderr and includes stack traces for fatal errors.
type defaultLogger struct {
	infoLogger  *log.Logger // Logger for informational messages
	errorLogger *log.Logger // Logger for error messages
	fatalLogger *log.Logger // Logger for fatal errors with stack traces
}

// logWithCallerf logs a formatted message with stack trace information.
// The stack trace helps identify where the fatal error occurred.
// Skip depth of 3: logWithCallerf -> Fatalf -> user code
func (l defaultLogger) logWithCallerf(logger *log.Logger, format string, args ...any) {
	stackInfo := stack(3) // Get caller info (file:line)
	// Prepend stack info to the log message
	logger.Printf("%s "+format, append([]any{stackInfo}, args...)...)
}

// logWithCaller logs unformatted arguments with stack trace information.
// The stack trace helps identify where the fatal error occurred.
// Skip depth of 3: logWithCaller -> Fatal -> user code
func (l defaultLogger) logWithCaller(logger *log.Logger, args ...any) {
	stack := stack(3) // Get caller info (file:line)
	logger.Println(append([]any{stack}, args...)...)
}

func (l defaultLogger) Infof(format string, args ...any) {
	l.infoLogger.Printf(format, args...)
}

func (l defaultLogger) Errorf(format string, args ...any) {
	l.errorLogger.Printf(format, args...)
}

func (l defaultLogger) Fatalf(format string, args ...any) {
	l.logWithCallerf(l.fatalLogger, format, args...)
}

func (l defaultLogger) Info(args ...any) {
	l.infoLogger.Println(fmt.Sprint(args...))
}

func (l defaultLogger) Error(args ...any) {
	l.errorLogger.Println(fmt.Sprint(args...))
}

func (l defaultLogger) Fatal(args ...any) {
	l.logWithCaller(l.fatalLogger, args...)
}

// NewEmptyLogger creates a no-op logger that discards all log messages.
// This is useful for:
//   - Performance-sensitive production environments where logging overhead matters
//   - Testing scenarios where log output would clutter test results
//   - Silent background workers that don't need observability
//
// Example:
//
//	q := queue.NewPool(5, queue.WithLogger(queue.NewEmptyLogger()))
func NewEmptyLogger() Logger {
	return emptyLogger{}
}

// emptyLogger is a no-op implementation that discards all log messages.
// All methods are implemented as empty functions with no side effects.
type emptyLogger struct{}

func (l emptyLogger) Infof(format string, args ...any)  {}
func (l emptyLogger) Errorf(format string, args ...any) {}
func (l emptyLogger) Fatalf(format string, args ...any) {}
func (l emptyLogger) Info(args ...any)                  {}
func (l emptyLogger) Error(args ...any)                 {}
func (l emptyLogger) Fatal(args ...any)                 {}
