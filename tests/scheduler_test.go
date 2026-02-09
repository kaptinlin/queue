package tests

import (
	"sync"
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
	args  []interface{}
}

func (l *mockLogger) log(level string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, mockLogEntry{level: level, args: args})
}

func (l *mockLogger) Debug(args ...interface{}) { l.log("debug", args...) }
func (l *mockLogger) Info(args ...interface{})  { l.log("info", args...) }
func (l *mockLogger) Warn(args ...interface{})  { l.log("warn", args...) }
func (l *mockLogger) Error(args ...interface{}) { l.log("error", args...) }
func (l *mockLogger) Fatal(args ...interface{}) { l.log("fatal", args...) }

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

	preEnqueueCalled := false
	preEnqueueHook := func(job *queue.Job) {
		preEnqueueCalled = true
		t.Log("PreEnqueueFunc called")
	}

	scheduler, err := queue.NewScheduler(redisConfig, queue.WithPreEnqueueFunc(preEnqueueHook))
	require.NoError(t, err, "Failed to create scheduler with pre enqueue hook")

	jobType := "pre_enqueue_test"
	payload := map[string]interface{}{"data": "pre"}

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

	assert.True(t, preEnqueueCalled, "Expected PreEnqueueFunc to be called")
}

func TestSchedulerPostEnqueueHook(t *testing.T) {
	redisConfig := getRedisConfig()

	postEnqueueCalled := false
	postEnqueueHook := func(job *queue.JobInfo, err error) {
		postEnqueueCalled = true
		t.Logf("PostEnqueueFunc called with job: %+v, error: %v", job, err)
	}

	scheduler, err := queue.NewScheduler(redisConfig, queue.WithPostEnqueueFunc(postEnqueueHook))
	require.NoError(t, err, "Failed to create scheduler with post enqueue hook")

	jobType := "post_enqueue_test"
	payload := map[string]interface{}{"data": "post"}

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

	assert.True(t, postEnqueueCalled, "Expected PostEnqueueFunc to be called")
}

func TestSchedulerPostEnqueueUsesConfiguredLogger(t *testing.T) {
	redisConfig := getRedisConfig()
	logger := &mockLogger{}

	scheduler, err := queue.NewScheduler(redisConfig,
		queue.WithSchedulerLogger(logger),
	)
	require.NoError(t, err, "Failed to create scheduler with custom logger")

	_, err = scheduler.RegisterCron("@every 1s", "logger_test", map[string]interface{}{"key": "value"})
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
