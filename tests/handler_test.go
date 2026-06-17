package tests

import (
	"context"
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

func TestHandlerWithLocalRateLimiter(t *testing.T) {
	limiter := rate.NewLimiter(rate.Every(15*time.Millisecond), 1)
	handler := newHandler(t, "test", func(context.Context, *queue.Delivery) error { return nil }, queue.WithLocalRateLimiter(limiter))
	require.NoError(t, handler.Process(context.Background(), nil))

	start := time.Now()
	err := handler.Process(context.Background(), nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, time.Since(start), 10*time.Millisecond)
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
		return ctx.Err()
	}, queue.WithJobTimeout(10*time.Millisecond))

	start := time.Now()
	err := handler.Process(context.Background(), nil)
	assert.GreaterOrEqual(t, time.Since(start), 25*time.Millisecond)
	assert.ErrorIs(t, err, queue.ErrJobProcessingTimeout)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestHandlerProcessWithLocalRateLimiter(t *testing.T) {
	limiter := rate.NewLimiter(rate.Every(time.Second), 1)
	handler := newHandler(t, "test", func(context.Context, *queue.Delivery) error {
		return nil
	}, queue.WithLocalRateLimiter(limiter))

	err := handler.Process(context.Background(), nil)
	require.NoError(t, err, "Process() should not return error for first call")

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	err = handler.Process(ctx, nil)
	assert.Error(t, err)
	assert.False(t, queue.IsRateLimitError(err))
}
