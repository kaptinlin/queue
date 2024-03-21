package tests

import (
	"testing"
	"time"

	"github.com/kaptinlin/queue"
)

func TestSchedulerInitialization(t *testing.T) {
	redisConfig := getRedisConfig()

	// Initialize a Scheduler instance with default settings.
	scheduler, err := queue.NewScheduler(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	if scheduler == nil {
		t.Fatal("Expected a valid Scheduler instance, got nil")
	}
}

func TestSchedulerStartAndStop(t *testing.T) {
	redisConfig := getRedisConfig()
	scheduler, err := queue.NewScheduler(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	go scheduler.Start()

	time.Sleep(2 * time.Second)

	if err := scheduler.Stop(); err != nil {
		t.Fatalf("Failed to stop scheduler: %v", err)
	}
}

func TestSchedulerRegisterWithInvalidCronSpec(t *testing.T) {
	redisConfig := getRedisConfig()

	// Initialize a Scheduler
	scheduler, _ := queue.NewScheduler(redisConfig)

	// Register a job with an invalid cron expression to trigger the error handler.
	_, err := scheduler.RegisterCron("wrong cron", "error_job", nil)

	// Verify the error was triggered.
	if err == nil {
		t.Fatal("Expected an error when registering a job with an invalid cron expression, but got nil")
	}
}

func TestSchedulerPreEnqueueHook(t *testing.T) {
	redisConfig := getRedisConfig()

	preEnqueueCalled := false
	preEnqueueHook := func(job *queue.Job) {
		preEnqueueCalled = true
		t.Log("PreEnqueueFunc called")
	}

	scheduler, err := queue.NewScheduler(redisConfig, queue.WithPreEnqueueFunc(preEnqueueHook))
	if err != nil {
		t.Fatalf("Failed to create scheduler with pre enqueue hook: %v", err)
	}

	jobType := "pre_enqueue_test"
	payload := map[string]interface{}{"data": "pre"}

	_, err = scheduler.RegisterCron("@every 1s", jobType, payload)
	if err != nil {
		t.Fatalf("Failed to register cron job: %v", err)
	}

	go scheduler.Start()
	defer scheduler.Stop()

	time.Sleep(5 * time.Second) // Wait for the scheduler to potentially enqueue jobs

	if !preEnqueueCalled {
		t.Error("Expected PreEnqueueFunc to be called, but it was not")
	}
}

func TestSchedulerPostEnqueueHook(t *testing.T) {
	redisConfig := getRedisConfig()

	postEnqueueCalled := false
	postEnqueueHook := func(job *queue.JobInfo, err error) {
		postEnqueueCalled = true
		t.Logf("PostEnqueueFunc called with job: %+v, error: %v", job, err)
	}

	scheduler, err := queue.NewScheduler(redisConfig, queue.WithPostEnqueueFunc(postEnqueueHook))
	if err != nil {
		t.Fatalf("Failed to create scheduler with post enqueue hook: %v", err)
	}

	jobType := "post_enqueue_test"
	payload := map[string]interface{}{"data": "post"}

	_, err = scheduler.RegisterCron("@every 1s", jobType, payload)
	if err != nil {
		t.Fatalf("Failed to register cron job: %v", err)
	}

	go scheduler.Start()
	defer scheduler.Stop()

	time.Sleep(5 * time.Second) // Wait for the scheduler to potentially enqueue jobs

	if !postEnqueueCalled {
		t.Error("Expected PostEnqueueFunc to be called, but it was not")
	}
}
