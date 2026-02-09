package queue

import (
	"fmt"
	"log/slog"
	"os"
)

// DefaultLogger provides a default implementation of Logger using slog.
// Users can provide their own Logger implementation via WithClientLogger or WithWorkerLogger.
type DefaultLogger struct{}

// NewDefaultLogger creates a new DefaultLogger.
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{}
}

// Debug logs a message at Debug level using slog.
func (l *DefaultLogger) Debug(args ...any) { slog.Debug(fmt.Sprint(args...)) }

// Info logs a message at Info level using slog.
func (l *DefaultLogger) Info(args ...any) { slog.Info(fmt.Sprint(args...)) }

// Warn logs a message at Warning level using slog.
func (l *DefaultLogger) Warn(args ...any) { slog.Warn(fmt.Sprint(args...)) }

// Error logs a message at Error level using slog.
func (l *DefaultLogger) Error(args ...any) { slog.Error(fmt.Sprint(args...)) }

// Fatal logs a message at Error level using slog and exits the process
// with status code 1.
func (l *DefaultLogger) Fatal(args ...any) {
	slog.Error(fmt.Sprint(args...))
	os.Exit(1)
}
