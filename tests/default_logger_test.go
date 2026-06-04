package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kaptinlin/queue"
)

func TestDefaultLoggerMethods(t *testing.T) {
	logger := queue.NewDefaultLogger()

	// These should not panic or cause errors.
	assert.NotPanics(t, func() { logger.Debug("debug message") })
	assert.NotPanics(t, func() { logger.Info("info message") })
	assert.NotPanics(t, func() { logger.Warn("warn message") })
	assert.NotPanics(t, func() { logger.Error("error message") })
}
