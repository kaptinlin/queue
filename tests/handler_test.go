package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestNewHandler(t *testing.T) {
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error { return nil })
	assert.Equal(t, "test", handler.JobType, "NewHandler() JobType should match")
}

func TestHandlerWithRateLimiter(t *testing.T) {
	limiter := rate.NewLimiter(1, 1)
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error { return nil }, queue.WithRateLimiter(limiter))
	assert.Equal(t, limiter, handler.Limiter, "WithRateLimiter() should set the expected limiter")
}

func TestHandlerWithJobTimeout(t *testing.T) {
	timeout := 5 * time.Second
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error { return nil }, queue.WithJobTimeout(timeout))
	assert.Equal(t, timeout, handler.JobTimeout, "WithJobTimeout() should set the expected timeout")
}

func TestHandlerWithJobQueue(t *testing.T) {
	queueName := "customQueue"
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error { return nil }, queue.WithJobQueue(queueName))
	assert.Equal(t, queueName, handler.JobQueue, "WithJobQueue() should set the expected queue name")
}

func TestHandlerWithRetryDelayFunc(t *testing.T) {
	customFuncCalled := false
	customFunc := func(attempt int, err error) time.Duration {
		customFuncCalled = true
		return 0
	}
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error { return nil }, queue.WithRetryDelayFunc(customFunc))
	handler.RetryDelayFunc(0, nil)
	assert.True(t, customFuncCalled, "WithRetryDelayFunc() should set the expected function")
}

func TestHandlerProcessHandleExecuted(t *testing.T) {
	handleExecuted := false
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error {
		handleExecuted = true
		return nil
	})

	job := &queue.Job{Type: "test", Payload: "data"}
	err := handler.Process(context.Background(), job)
	require.NoError(t, err, "Process() should not return error")
	assert.True(t, handleExecuted, "Process() should execute handle function")
}

func TestHandlerProcessWithTimeout(t *testing.T) {
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error {
		time.Sleep(2 * time.Second) // Simulate work that exceeds the timeout
		return nil
	}, queue.WithJobTimeout(1*time.Second))

	job := &queue.Job{Type: "test", Payload: "data"}
	err := handler.Process(context.Background(), job)
	assert.Error(t, err, "Process() should return timeout error")
	assert.Equal(t, "job processing exceeded timeout: context deadline exceeded", err.Error(), "Error message should match expected timeout message")
}

func TestHandlerProcessWithRateLimiter(t *testing.T) {
	limiter := rate.NewLimiter(rate.Every(10*time.Second), 1) // Allow only 1 operation per 10 seconds
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error {
		return nil
	}, queue.WithRateLimiter(limiter))

	// The first call should pass due to the rate limiter allowance
	job := &queue.Job{Type: "test", Payload: "data"}
	err := handler.Process(context.Background(), job)
	require.NoError(t, err, "Process() should not return error for first call")

	// The second call should be limited
	err = handler.Process(context.Background(), job)
	_, ok := errors.AsType[*queue.ErrRateLimit](err)
	assert.True(t, ok, "Process() should return ErrRateLimit error")
}
