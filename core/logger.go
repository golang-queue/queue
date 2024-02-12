package core

// Logger interface is used throughout queue package
type Logger interface {
	// Debug logs debug-level messages.
	Debug(args ...interface{})

	// Info logs info-level messages.
	Info(args ...interface{})

	// Warn logs warning-level messages.
	Warn(args ...interface{})

	// Error logs error-level messages.
	Error(args ...interface{})

	// Fatal logs fatal-level messages.
	Fatal(args ...interface{})
}
