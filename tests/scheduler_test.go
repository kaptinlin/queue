package tests

import (
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
