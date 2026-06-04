package queue

import (
	"fmt"
	"log/slog"
)

// DefaultLogger provides a default implementation of Logger using slog.
// Users can provide their own Logger implementation via WithClientLogger or WithWorkerLogger.
type DefaultLogger struct{}

// NewDefaultLogger creates a new DefaultLogger.
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{}
}

// Debug logs a message at Debug level using slog.
func (l *DefaultLogger) Debug(args ...any) {
	msg, attrs := loggerMessageAndAttrs(args...)
	slog.Debug(msg, attrs...)
}

// Info logs a message at Info level using slog.
func (l *DefaultLogger) Info(args ...any) {
	msg, attrs := loggerMessageAndAttrs(args...)
	slog.Info(msg, attrs...)
}

// Warn logs a message at Warning level using slog.
func (l *DefaultLogger) Warn(args ...any) {
	msg, attrs := loggerMessageAndAttrs(args...)
	slog.Warn(msg, attrs...)
}

// Error logs a message at Error level using slog.
func (l *DefaultLogger) Error(args ...any) {
	msg, attrs := loggerMessageAndAttrs(args...)
	slog.Error(msg, attrs...)
}

func loggerMessageAndAttrs(args ...any) (string, []any) {
	if len(args) == 0 {
		return "", nil
	}

	msg, ok := args[0].(string)
	if !ok || !validLogAttrs(args[1:]) {
		return fmt.Sprint(args...), nil
	}
	return msg, args[1:]
}

func validLogAttrs(attrs []any) bool {
	if len(attrs)%2 != 0 {
		return false
	}
	for i := 0; i < len(attrs); i += 2 {
		if _, ok := attrs[i].(string); !ok {
			return false
		}
	}
	return true
}
