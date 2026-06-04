package tests

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
)

// mockLogger captures log messages for testing.
type mockLogger struct {
	mu       sync.Mutex
	messages []mockLogEntry
}

type mockLogEntry struct {
	level string
	args  []any
}

func (l *mockLogger) log(level string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, mockLogEntry{level: level, args: args})
}

func (l *mockLogger) Debug(args ...any) { l.log("debug", args...) }
func (l *mockLogger) Info(args ...any)  { l.log("info", args...) }
func (l *mockLogger) Warn(args ...any)  { l.log("warn", args...) }
func (l *mockLogger) Error(args ...any) { l.log("error", args...) }

func (l *mockLogger) hasLevel(level string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, m := range l.messages {
		if m.level == level {
			return true
		}
	}
	return false
}

func TestSchedulerInitialization(t *testing.T) {
	redisConfig := getRedisConfig()

	// Initialize a Scheduler instance with default settings.
	scheduler, err := queue.NewScheduler(redisConfig)
	require.NoError(t, err, "Failed to create scheduler")
	assert.NotNil(t, scheduler, "Expected a valid Scheduler instance")
}

func TestSchedulerRunCanceledContext(t *testing.T) {
	redisConfig := getRedisConfig()
	scheduler, err := queue.NewScheduler(redisConfig)
	require.NoError(t, err, "Failed to create scheduler")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- scheduler.Run(ctx)
	}()

	time.Sleep(2 * time.Second)
	cancel()

	select {
	case err := <-done:
		require.NoError(t, err, "Failed to stop scheduler")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for scheduler shutdown")
	}
}

func TestSchedulerRegisterWithInvalidCronSpec(t *testing.T) {
	redisConfig := getRedisConfig()

	// Initialize a Scheduler
	scheduler, err := queue.NewScheduler(redisConfig)
	require.NoError(t, err, "Failed to create scheduler")

	// Register a job with an invalid cron expression to trigger the error handler.
	_, err = scheduler.RegisterCron("error_job", "wrong cron", "error_job", nil)

	// Verify the error was triggered.
	assert.ErrorIs(t, err, queue.ErrInvalidCronSpec)
}

func TestSchedulerPostEnqueueUsesConfiguredLogger(t *testing.T) {
	redisConfig := getRedisConfig()
	logger := &mockLogger{}

	scheduler, err := queue.NewScheduler(redisConfig,
		queue.WithSchedulerLogger(logger),
	)
	require.NoError(t, err, "Failed to create scheduler with custom logger")

	_, err = scheduler.RegisterCron("logger_test", "@every 1s", "logger_test", map[string]any{"key": "value"})
	require.NoError(t, err, "Failed to register cron job")

	runScheduler(t, scheduler)

	time.Sleep(5 * time.Second) // Wait for the scheduler to enqueue jobs

	assert.True(t, logger.hasLevel("info"), "Expected configured logger to receive scheduler enqueue log")
}

// TestSchedulerCronTriggerWithWorkerProcessing verifies the full lifecycle:
// scheduler enqueues a cron job → worker picks it up and processes it.
func TestSchedulerCronTriggerWithWorkerProcessing(t *testing.T) {
	redisConfig := getRedisConfig()

	const jobType = "scheduler_cron_process_test"
	payload := map[string]string{"action": "cron_trigger"}

	// Track job processing via worker.
	var processed atomic.Bool
	var processedPayload sync.Map

	// Set up worker to process the scheduled job.
	worker, err := queue.NewWorker(redisConfig)
	require.NoError(t, err, "Failed to create worker")

	err = worker.Register(jobType, func(_ context.Context, delivery *queue.Delivery) error {
		var p map[string]string
		if decErr := delivery.DecodePayload(&p); decErr != nil {
			return decErr
		}
		processedPayload.Store("action", p["action"])
		processed.Store(true)
		return nil
	})
	require.NoError(t, err, "Failed to register handler")

	runWorker(t, worker)

	// Set up scheduler with a 1-second cron interval.
	scheduler, err := queue.NewScheduler(redisConfig,
		queue.WithSyncInterval(1*time.Second),
	)
	require.NoError(t, err, "Failed to create scheduler")

	_, err = scheduler.RegisterCron(jobType, "@every 1s", jobType, payload)
	require.NoError(t, err, "Failed to register cron job")

	runScheduler(t, scheduler)

	// Wait for the job to be enqueued and processed.
	require.Eventually(t, processed.Load, 10*time.Second, 200*time.Millisecond,
		"Expected worker to process the cron-triggered job")

	// Verify the payload was correctly passed through.
	val, ok := processedPayload.Load("action")
	require.True(t, ok, "Expected payload key 'action' to exist")
	assert.Equal(t, "cron_trigger", val, "Payload mismatch")
}

// TestSchedulerPeriodicMultipleExecutions verifies that a periodic job
// fires multiple times over a time window.
func TestSchedulerPeriodicMultipleExecutions(t *testing.T) {
	redisConfig := getRedisConfig()

	const jobType = "scheduler_periodic_multi_test"
	var execCount atomic.Int64

	// Set up worker.
	worker, err := queue.NewWorker(redisConfig)
	require.NoError(t, err, "Failed to create worker")

	err = worker.Register(jobType, func(_ context.Context, _ *queue.Delivery) error {
		execCount.Add(1)
		return nil
	})
	require.NoError(t, err, "Failed to register handler")

	runWorker(t, worker)

	// Set up scheduler with RegisterPeriodic.
	scheduler, err := queue.NewScheduler(redisConfig,
		queue.WithSyncInterval(1*time.Second),
	)
	require.NoError(t, err, "Failed to create scheduler")

	_, err = scheduler.RegisterPeriodic(
		jobType, 1*time.Second, jobType, map[string]string{"key": "periodic"},
	)
	require.NoError(t, err, "Failed to register periodic job")

	runScheduler(t, scheduler)

	// Wait until the job has been processed at least 2 times.
	require.Eventually(t, func() bool {
		return execCount.Load() >= 2
	}, 15*time.Second, 200*time.Millisecond,
		"Expected periodic job to execute at least 2 times")
}
