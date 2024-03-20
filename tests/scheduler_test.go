package tests

import (
	"log"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
)

func TestSchedulerInitialization(t *testing.T) {
	redisConfig := getRedisConfig()

	// Attempt to create a Scheduler instance
	scheduler, err := queue.NewScheduler(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	if scheduler == nil {
		t.Fatal("Scheduler instance is nil, expected a valid instance")
	}
}

func TestSchedulerRegisterAndStart(t *testing.T) {
	redisConfig := getRedisConfig()

	scheduler, _ := queue.NewScheduler(redisConfig)

	// Define a dummy job
	jobType := "dummy_job"
	payload := map[string]interface{}{"data": "test"}

	// Register a cron job
	_, err := scheduler.RegisterCron("0 */1 * * *", jobType, payload) // Every hour
	if err != nil {
		t.Fatalf("Failed to register cron job: %v", err)
	}

	// Register a periodic job
	_, err = scheduler.RegisterPeriodic(30*time.Minute, jobType, payload) // Every 30 minutes
	if err != nil {
		t.Fatalf("Failed to register periodic job: %v", err)
	}

	// Start the scheduler in a goroutine to not block the test
	go func() {
		if err := scheduler.Start(); err != nil {
			log.Fatalf("Failed to start scheduler: %v", err)
		}
	}()

	// Let the scheduler run for some time to possibly enqueue jobs
	time.Sleep(2 * time.Second)

	// Stop the scheduler
	if err := scheduler.Stop(); err != nil {
		t.Fatalf("Failed to stop scheduler: %v", err)
	}
}

func TestSchedulerErrorHandling(t *testing.T) {
	redisConfig := getRedisConfig()

	var encounteredError bool
	errorHandler := func(job *queue.Job, err error) {
		encounteredError = true
	}

	scheduler, _ := queue.NewScheduler(redisConfig,
		queue.WithSchedulerEnqueueErrorHandler(errorHandler),
	)

	// Attempt to register a job which should fail and invoke the error handler
	scheduler.RegisterCron("wrong cron", "error_job", nil) // Every hour

	// Start the scheduler, expecting it to attempt and fail to enqueue the job
	go scheduler.Start()

	// Wait briefly for the scheduler to attempt job enqueueing
	time.Sleep(1 * time.Second)

	if !encounteredError {
		t.Fatal("Expected an error to be encountered and handled, but it was not")
	}

	scheduler.Stop()
}
