package tests

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
func (l *mockLogger) Fatal(args ...any) { l.log("fatal", args...) }

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

func TestSchedulerStartAndStop(t *testing.T) {
	redisConfig := getRedisConfig()
	scheduler, err := queue.NewScheduler(redisConfig)
	require.NoError(t, err, "Failed to create scheduler")

	go func() {
		if err := scheduler.Start(); err != nil {
			t.Errorf("Failed to start scheduler: %v", err)
		}
	}()

	time.Sleep(2 * time.Second)

	err = scheduler.Stop()
	require.NoError(t, err, "Failed to stop scheduler")
}

func TestSchedulerRegisterWithInvalidCronSpec(t *testing.T) {
	redisConfig := getRedisConfig()

	// Initialize a Scheduler
	scheduler, err := queue.NewScheduler(redisConfig)
	require.NoError(t, err, "Failed to create scheduler")

	// Register a job with an invalid cron expression to trigger the error handler.
	_, err = scheduler.RegisterCron("wrong cron", "error_job", nil)

	// Verify the error was triggered.
	assert.Error(t, err, "Expected an error when registering a job with an invalid cron expression")
}

func TestSchedulerPreEnqueueHook(t *testing.T) {
	redisConfig := getRedisConfig()

	var preEnqueueCalled atomic.Bool
	preEnqueueHook := func(job *queue.Job) {
		preEnqueueCalled.Store(true)
		t.Log("PreEnqueueFunc called")
	}

	scheduler, err := queue.NewScheduler(redisConfig, queue.WithPreEnqueueFunc(preEnqueueHook))
	require.NoError(t, err, "Failed to create scheduler with pre enqueue hook")

	jobType := "pre_enqueue_test"
	payload := map[string]any{"data": "pre"}

	_, err = scheduler.RegisterCron("@every 1s", jobType, payload)
	require.NoError(t, err, "Failed to register cron job")

	go func() {
		if err := scheduler.Start(); err != nil {
			t.Errorf("Failed to start scheduler: %v", err)
		}
	}()
	defer func() {
		assert.NoError(t, scheduler.Stop(), "Failed to stop scheduler")
	}()

	time.Sleep(5 * time.Second) // Wait for the scheduler to potentially enqueue jobs

	assert.True(t, preEnqueueCalled.Load(), "Expected PreEnqueueFunc to be called")
}

func TestSchedulerPostEnqueueHook(t *testing.T) {
	redisConfig := getRedisConfig()

	var postEnqueueCalled atomic.Bool
	postEnqueueHook := func(job *queue.JobInfo, err error) {
		postEnqueueCalled.Store(true)
		t.Logf("PostEnqueueFunc called with job: %+v, error: %v", job, err)
	}

	scheduler, err := queue.NewScheduler(redisConfig, queue.WithPostEnqueueFunc(postEnqueueHook))
	require.NoError(t, err, "Failed to create scheduler with post enqueue hook")

	jobType := "post_enqueue_test"
	payload := map[string]any{"data": "post"}

	_, err = scheduler.RegisterCron("@every 1s", jobType, payload)
	require.NoError(t, err, "Failed to register cron job")

	go func() {
		if err := scheduler.Start(); err != nil {
			t.Errorf("Failed to start scheduler: %v", err)
		}
	}()
	defer func() {
		assert.NoError(t, scheduler.Stop(), "Failed to stop scheduler")
	}()

	time.Sleep(5 * time.Second) // Wait for the scheduler to potentially enqueue jobs

	assert.True(t, postEnqueueCalled.Load(), "Expected PostEnqueueFunc to be called")
}

func TestSchedulerPostEnqueueUsesConfiguredLogger(t *testing.T) {
	redisConfig := getRedisConfig()
	logger := &mockLogger{}

	scheduler, err := queue.NewScheduler(redisConfig,
		queue.WithSchedulerLogger(logger),
	)
	require.NoError(t, err, "Failed to create scheduler with custom logger")

	_, err = scheduler.RegisterCron("@every 1s", "logger_test", map[string]any{"key": "value"})
	require.NoError(t, err, "Failed to register cron job")

	go func() {
		if err := scheduler.Start(); err != nil {
			t.Errorf("Failed to start scheduler: %v", err)
		}
	}()
	defer func() {
		assert.NoError(t, scheduler.Stop(), "Failed to stop scheduler")
	}()

	time.Sleep(5 * time.Second) // Wait for the scheduler to enqueue jobs

	// Verify the configured logger received info-level messages from PostEnqueueFunc
	assert.True(t, logger.hasLevel("info"), "Expected configured logger to receive info log from PostEnqueueFunc")
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

	err = worker.Register(jobType, func(ctx context.Context, job *queue.Job) error {
		var p map[string]string
		if decErr := job.DecodePayload(&p); decErr != nil {
			return decErr
		}
		processedPayload.Store("action", p["action"])
		processed.Store(true)
		return nil
	})
	require.NoError(t, err, "Failed to register handler")

	go func() {
		if startErr := worker.Start(); startErr != nil {
			t.Errorf("Worker failed to start: %v", startErr)
		}
	}()
	defer func() {
		assert.NoError(t, worker.Stop(), "Failed to stop worker")
	}()

	// Set up scheduler with a 1-second cron interval.
	scheduler, err := queue.NewScheduler(redisConfig,
		queue.WithSyncInterval(1*time.Second),
	)
	require.NoError(t, err, "Failed to create scheduler")

	_, err = scheduler.RegisterCron("@every 1s", jobType, payload)
	require.NoError(t, err, "Failed to register cron job")

	go func() {
		if startErr := scheduler.Start(); startErr != nil {
			t.Errorf("Scheduler failed to start: %v", startErr)
		}
	}()
	defer func() {
		assert.NoError(t, scheduler.Stop(), "Failed to stop scheduler")
	}()

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

	err = worker.Register(jobType, func(_ context.Context, _ *queue.Job) error {
		execCount.Add(1)
		return nil
	})
	require.NoError(t, err, "Failed to register handler")

	go func() {
		if startErr := worker.Start(); startErr != nil {
			t.Errorf("Worker failed to start: %v", startErr)
		}
	}()
	defer func() {
		assert.NoError(t, worker.Stop(), "Failed to stop worker")
	}()

	// Set up scheduler with RegisterPeriodic.
	scheduler, err := queue.NewScheduler(redisConfig,
		queue.WithSyncInterval(1*time.Second),
	)
	require.NoError(t, err, "Failed to create scheduler")

	_, err = scheduler.RegisterPeriodic(
		1*time.Second, jobType, map[string]string{"key": "periodic"},
	)
	require.NoError(t, err, "Failed to register periodic job")

	go func() {
		if startErr := scheduler.Start(); startErr != nil {
			t.Errorf("Scheduler failed to start: %v", startErr)
		}
	}()
	defer func() {
		assert.NoError(t, scheduler.Stop(), "Failed to stop scheduler")
	}()

	// Wait until the job has been processed at least 2 times.
	require.Eventually(t, func() bool {
		return execCount.Load() >= 2
	}, 15*time.Second, 200*time.Millisecond,
		"Expected periodic job to execute at least 2 times")
}

// TestSchedulerPreEnqueueFuncReceivesJobType verifies that the
// PreEnqueueFunc callback receives a Job with the correct type.
func TestSchedulerPreEnqueueFuncReceivesJobType(t *testing.T) {
	redisConfig := getRedisConfig()

	const jobType = "pre_enqueue_type_verify"

	var mu sync.Mutex
	var capturedTypes []string

	scheduler, err := queue.NewScheduler(redisConfig,
		queue.WithSyncInterval(1*time.Second),
		queue.WithPreEnqueueFunc(func(job *queue.Job) {
			mu.Lock()
			defer mu.Unlock()
			capturedTypes = append(capturedTypes, job.Type)
		}),
	)
	require.NoError(t, err, "Failed to create scheduler")

	_, err = scheduler.RegisterCron(
		"@every 1s", jobType, map[string]string{"k": "v"},
	)
	require.NoError(t, err, "Failed to register cron job")

	go func() {
		if startErr := scheduler.Start(); startErr != nil {
			t.Errorf("Scheduler failed to start: %v", startErr)
		}
	}()

	// Allow scheduler to fully initialize before deferred Stop.
	time.Sleep(1 * time.Second)

	defer func() {
		assert.NoError(t, scheduler.Stop(), "Failed to stop scheduler")
	}()

	// Wait for at least one callback invocation.
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(capturedTypes) >= 1
	}, 10*time.Second, 200*time.Millisecond,
		"Expected PreEnqueueFunc to be called at least once")

	mu.Lock()
	defer mu.Unlock()
	for _, ct := range capturedTypes {
		assert.Equal(t, jobType, ct,
			"PreEnqueueFunc received unexpected job type")
	}
}

// TestSchedulerPostEnqueueFuncReceivesJobInfo verifies that the
// PostEnqueueFunc callback receives a valid JobInfo with correct
// type and no error on successful enqueue.
func TestSchedulerPostEnqueueFuncReceivesJobInfo(t *testing.T) {
	redisConfig := getRedisConfig()

	const jobType = "post_enqueue_info_verify"

	type postEnqueueRecord struct {
		jobInfo *queue.JobInfo
		err     error
	}

	var mu sync.Mutex
	var records []postEnqueueRecord

	scheduler, err := queue.NewScheduler(redisConfig,
		queue.WithSyncInterval(1*time.Second),
		queue.WithPostEnqueueFunc(func(info *queue.JobInfo, err error) {
			mu.Lock()
			defer mu.Unlock()
			records = append(records, postEnqueueRecord{
				jobInfo: info,
				err:     err,
			})
		}),
	)
	require.NoError(t, err, "Failed to create scheduler")

	_, err = scheduler.RegisterCron(
		"@every 1s", jobType, map[string]string{"k": "v"},
	)
	require.NoError(t, err, "Failed to register cron job")

	go func() {
		if startErr := scheduler.Start(); startErr != nil {
			t.Errorf("Scheduler failed to start: %v", startErr)
		}
	}()
	defer func() {
		assert.NoError(t, scheduler.Stop(), "Failed to stop scheduler")
	}()

	// Wait for at least one callback invocation.
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(records) >= 1
	}, 10*time.Second, 200*time.Millisecond,
		"Expected PostEnqueueFunc to be called at least once")

	mu.Lock()
	defer mu.Unlock()
	for i, rec := range records {
		assert.NoError(t, rec.err,
			"PostEnqueueFunc record %d should have no error", i)
		require.NotNil(t, rec.jobInfo,
			"PostEnqueueFunc record %d should have non-nil JobInfo", i)
		assert.Equal(t, jobType, rec.jobInfo.Type,
			"PostEnqueueFunc record %d has wrong job type", i)
		assert.NotEmpty(t, rec.jobInfo.ID,
			"PostEnqueueFunc record %d should have a job ID", i)
		assert.NotEmpty(t, rec.jobInfo.Queue,
			"PostEnqueueFunc record %d should have a queue name", i)
	}
}

// TestSchedulerPreAndPostEnqueueFuncsBothFire verifies that when
// both Pre and Post enqueue hooks are configured, both are invoked
// for each scheduled job enqueue.
func TestSchedulerPreAndPostEnqueueFuncsBothFire(t *testing.T) {
	redisConfig := getRedisConfig()

	const jobType = "pre_post_both_test"

	var preCount atomic.Int64
	var postCount atomic.Int64

	scheduler, err := queue.NewScheduler(redisConfig,
		queue.WithSyncInterval(1*time.Second),
		queue.WithPreEnqueueFunc(func(_ *queue.Job) {
			preCount.Add(1)
		}),
		queue.WithPostEnqueueFunc(func(_ *queue.JobInfo, _ error) {
			postCount.Add(1)
		}),
	)
	require.NoError(t, err, "Failed to create scheduler")

	_, err = scheduler.RegisterCron(
		"@every 1s", jobType, map[string]string{"k": "v"},
	)
	require.NoError(t, err, "Failed to register cron job")

	go func() {
		if startErr := scheduler.Start(); startErr != nil {
			t.Errorf("Scheduler failed to start: %v", startErr)
		}
	}()
	defer func() {
		assert.NoError(t, scheduler.Stop(), "Failed to stop scheduler")
	}()

	// Wait for at least one invocation of each hook.
	require.Eventually(t, func() bool {
		return preCount.Load() >= 1 && postCount.Load() >= 1
	}, 10*time.Second, 200*time.Millisecond,
		"Expected both PreEnqueueFunc and PostEnqueueFunc to fire")

	// Both hooks should have been called the same number of times.
	assert.Equal(t, preCount.Load(), postCount.Load(),
		"Pre and Post enqueue call counts should match")
}

// TestSchedulerUnregisterCronJobStopsEnqueue verifies that after
// unregistering a cron job, the scheduler stops enqueuing it.
func TestSchedulerUnregisterCronJobStopsEnqueue(t *testing.T) {
	redisConfig := getRedisConfig()

	const jobType = "unregister_stop_test"
	var enqueueCount atomic.Int64

	scheduler, err := queue.NewScheduler(redisConfig,
		queue.WithSyncInterval(1*time.Second),
		queue.WithPostEnqueueFunc(func(_ *queue.JobInfo, _ error) {
			enqueueCount.Add(1)
		}),
	)
	require.NoError(t, err, "Failed to create scheduler")

	id, err := scheduler.RegisterCron(
		"@every 1s", jobType, map[string]string{"k": "v"},
	)
	require.NoError(t, err, "Failed to register cron job")

	go func() {
		if startErr := scheduler.Start(); startErr != nil {
			t.Errorf("Scheduler failed to start: %v", startErr)
		}
	}()
	defer func() {
		assert.NoError(t, scheduler.Stop(), "Failed to stop scheduler")
	}()

	// Wait for at least one enqueue.
	require.Eventually(t, func() bool {
		return enqueueCount.Load() >= 1
	}, 10*time.Second, 200*time.Millisecond,
		"Expected at least one enqueue before unregister")

	// Unregister the job.
	err = scheduler.UnregisterCronJob(id)
	require.NoError(t, err, "Failed to unregister cron job")

	// Wait for sync interval so the scheduler picks up the config change,
	// plus buffer for any in-flight enqueue to complete.
	time.Sleep(3 * time.Second)

	// Record count after the scheduler has synced the unregistration.
	countAfterSync := enqueueCount.Load()

	// Wait another interval to confirm no more enqueues happen.
	time.Sleep(3 * time.Second)

	assert.Equal(t, countAfterSync, enqueueCount.Load(),
		"No more enqueues should happen after unregistering the job")
}
