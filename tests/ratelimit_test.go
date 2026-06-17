package tests

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/kaptinlin/queue"
)

func TestWorkerLocalRateLimiterWaitsBeforeHandler(t *testing.T) {
	redisConfig := getRedisConfig()

	var processed atomic.Int64
	limiter := rate.NewLimiter(rate.Every(40*time.Millisecond), 1)
	worker, err := queue.NewWorker(redisConfig,
		queue.WithWorkerLocalRateLimiter(limiter),
		queue.WithWorkerConcurrency(1),
	)
	require.NoError(t, err)

	jobType := "ratelimit_worker_wait_test"
	var wg sync.WaitGroup
	err = worker.Register(jobType, func(context.Context, *queue.Delivery) error {
		defer wg.Done()
		processed.Add(1)
		return nil
	})
	require.NoError(t, err)

	runWorker(t, worker)

	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, client.Close())
	}()

	wg.Add(2)
	start := time.Now()
	_, err = client.Enqueue(jobType, map[string]any{"seq": 1})
	require.NoError(t, err)
	_, err = client.Enqueue(jobType, map[string]any{"seq": 2})
	require.NoError(t, err)

	wg.Wait()
	assert.Equal(t, int64(2), processed.Load())
	assert.GreaterOrEqual(t, time.Since(start), 30*time.Millisecond)
}

func TestHandlerLocalRateLimiterWaitsIndependently(t *testing.T) {
	limiter := rate.NewLimiter(rate.Every(20*time.Millisecond), 1)
	handler := newHandler(t, "ratelimit_handler_wait_test",
		func(context.Context, *queue.Delivery) error {
			return nil
		},
		queue.WithLocalRateLimiter(limiter),
	)

	require.NoError(t, handler.Process(context.Background(), nil))

	start := time.Now()
	err := handler.Process(context.Background(), nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, time.Since(start), 15*time.Millisecond)
}

func TestLocalRateLimiterReturnsContextErrorWhenWaitIsCanceled(t *testing.T) {
	limiter := rate.NewLimiter(rate.Every(time.Second), 1)
	handler := newHandler(t, "ratelimit_handler_cancel_test",
		func(context.Context, *queue.Delivery) error {
			return nil
		},
		queue.WithLocalRateLimiter(limiter),
	)

	require.NoError(t, handler.Process(context.Background(), nil))

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	err := handler.Process(ctx, nil)
	assert.Error(t, err)
	assert.False(t, queue.IsRateLimitError(err))
}

func TestRateLimitErrorIsNotCountedAsFailure(t *testing.T) {
	err := queue.NewRateLimitError(5 * time.Second)

	assert.False(t, queue.IsRateLimitError(nil))
	assert.True(t, queue.IsRateLimitError(err))
}

func TestBusinessRateLimitErrorReachesWorkerErrorHandler(t *testing.T) {
	redisConfig := getRedisConfig()
	errorHandler := NewCustomWorkerErrorHandler()

	worker, err := queue.NewWorker(redisConfig,
		queue.WithWorkerErrorHandler(errorHandler),
		queue.WithWorkerConcurrency(1),
	)
	require.NoError(t, err)

	jobType := "ratelimit_business_error_test"
	err = worker.Register(jobType, func(context.Context, *queue.Delivery) error {
		return queue.NewRateLimitError(50 * time.Millisecond)
	})
	require.NoError(t, err)

	runWorker(t, worker)

	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, client.Close())
	}()

	_, err = client.Enqueue(jobType, map[string]any{"seq": 1})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return slices.ContainsFunc(errorHandler.Errors(), queue.IsRateLimitError)
	}, 3*time.Second, 20*time.Millisecond)
}
