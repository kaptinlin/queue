package queue

// Logger supports logging at various log levels.
type Logger interface {
	// Debug logs a message at Debug level.
	Debug(args ...any)

	// Info logs a message at Info level.
	Info(args ...any)

	// Warn logs a message at Warning level.
	Warn(args ...any)

	// Error logs a message at Error level.
	Error(args ...any)

	// Fatal logs a message at Fatal level
	// and process will exit with status set to 1.
	Fatal(args ...any)
}
