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

func (l *DefaultLogger) Debug(args ...interface{}) { slog.Debug(fmt.Sprint(args...)) }
func (l *DefaultLogger) Info(args ...interface{})  { slog.Info(fmt.Sprint(args...)) }
func (l *DefaultLogger) Warn(args ...interface{})  { slog.Warn(fmt.Sprint(args...)) }
func (l *DefaultLogger) Error(args ...interface{}) { slog.Error(fmt.Sprint(args...)) }
func (l *DefaultLogger) Fatal(args ...interface{}) {
	slog.Error(fmt.Sprint(args...))
	panic(fmt.Sprint(args...))
}
