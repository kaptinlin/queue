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
	handler := newHandler(t, "test", func(context.Context, *queue.Delivery) error { return nil })
	assert.Equal(t, "test", handler.Type(), "NewHandler() Type should match")
}

func TestHandlerWithRateLimiter(t *testing.T) {
	limiter := rate.NewLimiter(1, 1)
	handler := newHandler(t, "test", func(context.Context, *queue.Delivery) error { return nil }, queue.WithRateLimiter(limiter))
	require.NoError(t, handler.Process(context.Background(), nil))

	err := handler.Process(context.Background(), nil)
	assert.True(t, queue.IsErrRateLimit(err))
}

func TestHandlerWithJobTimeout(t *testing.T) {
	timeout := 5 * time.Second
	handler := newHandler(t, "test", func(context.Context, *queue.Delivery) error { return nil }, queue.WithJobTimeout(timeout))
	assert.Equal(t, timeout, handler.Timeout(), "WithJobTimeout() should set the expected timeout")
}

func TestHandlerWithJobQueue(t *testing.T) {
	queueName := "customQueue"
	handler := newHandler(t, "test", func(context.Context, *queue.Delivery) error { return nil }, queue.WithJobQueue(queueName))
	assert.Equal(t, queueName, handler.Queue(), "WithJobQueue() should set the expected queue name")
}

func TestHandlerWithRetryDelayFunc(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig())
	require.NoError(t, err)

	customFunc := func(attempt int, _ error) time.Duration {
		return time.Duration(attempt) * time.Second
	}
	handler := newHandler(t, "test",
		func(context.Context, *queue.Delivery) error { return nil },
		queue.WithRetryDelayFunc(customFunc),
	)
	require.NoError(t, worker.RegisterHandler(handler))
}

func TestHandlerProcessHandleExecuted(t *testing.T) {
	handleExecuted := false
	handler := newHandler(t, "test", func(context.Context, *queue.Delivery) error {
		handleExecuted = true
		return nil
	})

	err := handler.Process(context.Background(), nil)
	require.NoError(t, err, "Process() should not return error")
	assert.True(t, handleExecuted, "Process() should execute handle function")
}

func TestHandlerProcessWithTimeout(t *testing.T) {
	t.Parallel()

	handler := newHandler(t, "test", func(ctx context.Context, _ *queue.Delivery) error {
		<-ctx.Done()
		time.Sleep(25 * time.Millisecond)
		return nil
	}, queue.WithJobTimeout(10*time.Millisecond))

	err := handler.Process(context.Background(), nil)
	assert.ErrorIs(t, err, queue.ErrJobProcessingTimeout)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestHandlerProcessWithRateLimiter(t *testing.T) {
	limiter := rate.NewLimiter(rate.Every(10*time.Second), 1) // Allow only 1 operation per 10 seconds
	handler := newHandler(t, "test", func(context.Context, *queue.Delivery) error {
		return nil
	}, queue.WithRateLimiter(limiter))

	// The first call should pass due to the rate limiter allowance
	err := handler.Process(context.Background(), nil)
	require.NoError(t, err, "Process() should not return error for first call")

	// The second call should be limited
	err = handler.Process(context.Background(), nil)
	_, ok := errors.AsType[*queue.ErrRateLimit](err)
	assert.True(t, ok, "Process() should return ErrRateLimit error")
}
