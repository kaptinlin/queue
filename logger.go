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
}

type asynqLogger struct {
	logger Logger
}

func newAsynqLogger(logger Logger) asynqLogger {
	if logger == nil {
		logger = NewDefaultLogger()
	}
	return asynqLogger{logger: logger}
}

func (l asynqLogger) Debug(args ...any) { l.logger.Debug(args...) }
func (l asynqLogger) Info(args ...any)  { l.logger.Info(args...) }
func (l asynqLogger) Warn(args ...any)  { l.logger.Warn(args...) }
func (l asynqLogger) Error(args ...any) { l.logger.Error(args...) }

func (l asynqLogger) Fatal(args ...any) {
	l.logger.Error(args...)
}
