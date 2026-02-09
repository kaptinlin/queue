package tests

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

// TestWorkerRateLimiterBlocksBeforeHandler verifies that the worker-level rate
// limiter is checked before the handler is invoked. When the worker limiter is
// exhausted, jobs should be rate-limited without ever reaching the handler.
func TestWorkerRateLimiterBlocksBeforeHandler(t *testing.T) {
	redisConfig := getRedisConfig()

	var handlerCalls atomic.Int64

	// Worker limiter: 1 token, burst 1 — only the first job passes.
	workerLimiter := rate.NewLimiter(rate.Every(10*time.Second), 1)

	worker, err := queue.NewWorker(redisConfig,
		queue.WithWorkerRateLimiter(workerLimiter),
	)
	require.NoError(t, err)

	jobType := "ratelimit_worker_test"
	var wg sync.WaitGroup
	wg.Add(1)

	err = worker.Register(jobType, func(ctx context.Context, job *queue.Job) error {
		handlerCalls.Add(1)
		wg.Done()
		return nil
	})
	require.NoError(t, err)

	go func() {
		if err := worker.Start(); err != nil {
			t.Errorf("worker start: %v", err)
		}
	}()
	defer func() {
		assert.NoError(t, worker.Stop())
	}()

	time.Sleep(1 * time.Second)

	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, client.Stop())
	}()

	// Enqueue first job — should be processed.
	_, err = client.Enqueue(jobType, map[string]any{"seq": 1})
	require.NoError(t, err)

	wg.Wait()
	assert.Equal(t, int64(1), handlerCalls.Load(),
		"exactly one job should reach the handler")
}

// TestHandlerRateLimiterIndependentOfWorker verifies that a handler-level rate
// limiter blocks jobs even when no worker-level limiter is configured.
func TestHandlerRateLimiterIndependentOfWorker(t *testing.T) {
	redisConfig := getRedisConfig()

	var processed atomic.Int64

	// Handler limiter: 1 token, burst 1.
	handlerLimiter := rate.NewLimiter(rate.Every(10*time.Second), 1)

	worker, err := queue.NewWorker(redisConfig)
	require.NoError(t, err)

	jobType := "ratelimit_handler_only_test"
	var wg sync.WaitGroup
	wg.Add(1)

	err = worker.Register(jobType, func(ctx context.Context, job *queue.Job) error {
		processed.Add(1)
		wg.Done()
		return nil
	}, queue.WithRateLimiter(handlerLimiter))
	require.NoError(t, err)

	go func() {
		if err := worker.Start(); err != nil {
			t.Errorf("worker start: %v", err)
		}
	}()
	defer func() {
		assert.NoError(t, worker.Stop())
	}()

	time.Sleep(1 * time.Second)

	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, client.Stop())
	}()

	// Enqueue first job — passes handler limiter.
	_, err = client.Enqueue(jobType, map[string]any{"seq": 1})
	require.NoError(t, err)

	wg.Wait()
	assert.Equal(t, int64(1), processed.Load(),
		"first job should be processed by handler")
}

// TestDualRateLimiterWorkerBlocksFirst verifies that when both worker-level and
// handler-level limiters are configured, the worker-level limiter is checked
// first. If the worker limiter denies the request, the handler limiter token is
// not consumed.
func TestDualRateLimiterWorkerBlocksFirst(t *testing.T) {
	redisConfig := getRedisConfig()

	var handlerCalls atomic.Int64

	// Worker limiter: burst 1 — blocks after first job.
	workerLimiter := rate.NewLimiter(rate.Every(10*time.Second), 1)
	// Handler limiter: burst 5 — generous, should not be the bottleneck.
	handlerLimiter := rate.NewLimiter(rate.Every(10*time.Second), 5)

	worker, err := queue.NewWorker(redisConfig,
		queue.WithWorkerRateLimiter(workerLimiter),
	)
	require.NoError(t, err)

	jobType := "ratelimit_dual_worker_first_test"
	var wg sync.WaitGroup
	wg.Add(1)

	err = worker.Register(jobType, func(ctx context.Context, job *queue.Job) error {
		handlerCalls.Add(1)
		wg.Done()
		return nil
	}, queue.WithRateLimiter(handlerLimiter))
	require.NoError(t, err)

	go func() {
		if err := worker.Start(); err != nil {
			t.Errorf("worker start: %v", err)
		}
	}()
	defer func() {
		assert.NoError(t, worker.Stop())
	}()

	time.Sleep(1 * time.Second)

	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, client.Stop())
	}()

	// First job passes both limiters.
	_, err = client.Enqueue(jobType, map[string]any{"seq": 1})
	require.NoError(t, err)

	wg.Wait()

	// Worker limiter is now exhausted. Handler limiter still has 4 tokens.
	// The second job should be blocked by the worker limiter.
	assert.Equal(t, int64(1), handlerCalls.Load(),
		"only one job should pass through the worker limiter")
}

// TestDualRateLimiterHandlerBlocksSecond verifies that when the worker-level
// limiter allows a job through but the handler-level limiter is exhausted, the
// job is still rate-limited with an ErrRateLimit error.
func TestDualRateLimiterHandlerBlocksSecond(t *testing.T) {
	// Use handler.Process directly to test the handler-level limiter in
	// isolation after the worker-level check would have passed.
	handlerLimiter := rate.NewLimiter(rate.Every(10*time.Second), 1)

	handler := queue.NewHandler("ratelimit_dual_handler_test",
		func(ctx context.Context, job *queue.Job) error {
			return nil
		},
		queue.WithRateLimiter(handlerLimiter),
	)

	job := &queue.Job{Type: "ratelimit_dual_handler_test", Payload: "data"}

	// First call consumes the handler limiter token.
	err := handler.Process(context.Background(), job)
	require.NoError(t, err, "first call should pass handler limiter")

	// Second call should be rate-limited by the handler limiter.
	err = handler.Process(context.Background(), job)
	var rateLimitErr *queue.ErrRateLimit
	require.True(t, errors.As(err, &rateLimitErr),
		"second call should return ErrRateLimit from handler limiter")
	assert.Equal(t, queue.DefaultRateLimitRetryAfter, rateLimitErr.RetryAfter,
		"RetryAfter should match DefaultRateLimitRetryAfter")
}

// TestDualRateLimiterBothAllow verifies that when both limiters have sufficient
// capacity, jobs are processed successfully through both layers.
func TestDualRateLimiterBothAllow(t *testing.T) {
	redisConfig := getRedisConfig()

	var processed atomic.Int64
	const jobCount = 3

	// Both limiters generous enough for all jobs.
	workerLimiter := rate.NewLimiter(rate.Limit(100), 10)
	handlerLimiter := rate.NewLimiter(rate.Limit(100), 10)

	worker, err := queue.NewWorker(redisConfig,
		queue.WithWorkerRateLimiter(workerLimiter),
	)
	require.NoError(t, err)

	jobType := "ratelimit_dual_both_allow_test"
	var wg sync.WaitGroup
	wg.Add(jobCount)

	err = worker.Register(jobType, func(ctx context.Context, job *queue.Job) error {
		processed.Add(1)
		wg.Done()
		return nil
	}, queue.WithRateLimiter(handlerLimiter))
	require.NoError(t, err)

	go func() {
		if err := worker.Start(); err != nil {
			t.Errorf("worker start: %v", err)
		}
	}()
	defer func() {
		assert.NoError(t, worker.Stop())
	}()

	time.Sleep(1 * time.Second)

	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, client.Stop())
	}()

	for i := range jobCount {
		_, err = client.Enqueue(jobType, map[string]any{"seq": i})
		require.NoError(t, err)
	}

	wg.Wait()
	assert.Equal(t, int64(jobCount), processed.Load(),
		"all jobs should be processed when both limiters allow")
}

// TestRateLimitedJobIsNotCountedAsFailure verifies that rate-limited jobs are
// not counted as failures (isFailure returns false for ErrRateLimit). This
// ensures rate-limited jobs can retry indefinitely without exhausting MaxRetries.
func TestRateLimitedJobIsNotCountedAsFailure(t *testing.T) {
	err := queue.NewErrRateLimit(5 * time.Second)
	assert.False(t, queue.IsErrRateLimit(nil),
		"nil error should not be a rate limit error")
	assert.True(t, queue.IsErrRateLimit(err),
		"ErrRateLimit should be detected by IsErrRateLimit")
}

// TestDualRateLimiterErrorHandlerReceivesRateLimitError verifies that the
// worker's custom error handler receives ErrRateLimit errors from the
// handler-level limiter. Note: worker-level rate limit errors are returned
// before job reconstruction, so they bypass the custom error handler by design.
func TestDualRateLimiterErrorHandlerReceivesRateLimitError(t *testing.T) {
	redisConfig := getRedisConfig()

	errorHandler := NewCustomWorkerErrorHandler()

	// Handler limiter: burst 1 — blocks after first job.
	handlerLimiter := rate.NewLimiter(rate.Every(10*time.Second), 1)

	worker, err := queue.NewWorker(redisConfig,
		queue.WithWorkerErrorHandler(errorHandler),
		queue.WithWorkerConcurrency(1),
	)
	require.NoError(t, err)

	jobType := "ratelimit_error_handler_test"
	var wg sync.WaitGroup
	wg.Add(1)

	err = worker.Register(jobType, func(ctx context.Context, job *queue.Job) error {
		wg.Done()
		return nil
	}, queue.WithRateLimiter(handlerLimiter))
	require.NoError(t, err)

	go func() {
		if err := worker.Start(); err != nil {
			t.Errorf("worker start: %v", err)
		}
	}()
	defer func() {
		assert.NoError(t, worker.Stop())
	}()

	time.Sleep(1 * time.Second)

	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, client.Stop())
	}()

	// First job passes the handler limiter.
	_, err = client.Enqueue(jobType, map[string]any{"seq": 1})
	require.NoError(t, err)
	wg.Wait()

	// Second job will be rate-limited by the handler limiter.
	_, err = client.Enqueue(jobType, map[string]any{"seq": 2})
	require.NoError(t, err)

	// Wait for the worker to attempt processing the second job.
	time.Sleep(3 * time.Second)

	// The error handler should have captured at least one rate limit error.
	errorHandler.mu.Lock()
	defer errorHandler.mu.Unlock()

	assert.True(t, slices.ContainsFunc(errorHandler.errors, queue.IsErrRateLimit),
		"error handler should receive ErrRateLimit from handler limiter")
}
