package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/kaptinlin/queue"
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
	t.Parallel()

	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error {
		<-ctx.Done()
		time.Sleep(25 * time.Millisecond)
		return nil
	}, queue.WithJobTimeout(10*time.Millisecond))

	job := &queue.Job{Type: "test", Payload: "data"}
	err := handler.Process(context.Background(), job)
	assert.ErrorIs(t, err, queue.ErrJobProcessingTimeout)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
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
