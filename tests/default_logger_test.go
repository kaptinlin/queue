package tests

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultLoggerMethods(t *testing.T) {
	logger := queue.NewDefaultLogger()

	// These should not panic or cause errors.
	assert.NotPanics(t, func() { logger.Debug("debug message") })
	assert.NotPanics(t, func() { logger.Info("info message") })
	assert.NotPanics(t, func() { logger.Warn("warn message") })
	assert.NotPanics(t, func() { logger.Error("error message") })
}

func TestDefaultLoggerFatalExitsWithCode1(t *testing.T) {
	if os.Getenv("TEST_FATAL_EXIT") == "1" {
		logger := queue.NewDefaultLogger()
		logger.Fatal("fatal message")
		return
	}

	// Re-invoke this test in a subprocess with the env var set.
	cmd := exec.Command(os.Args[0], "-test.run=^TestDefaultLoggerFatalExitsWithCode1$")
	cmd.Env = append(os.Environ(), "TEST_FATAL_EXIT=1")

	err := cmd.Run()

	// The subprocess should have exited with code 1.
	require.Error(t, err, "expected Fatal to cause a non-zero exit")

	exitErr, ok := errors.AsType[*exec.ExitError](err)
	require.True(t, ok, "expected *exec.ExitError")
	assert.Equal(t, 1, exitErr.ExitCode(), "expected exit code 1")
}
