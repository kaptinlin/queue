package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
)

// Define a test-specific job type and payload.
const testJobType = "test_job"

type TestJobPayload struct {
	Message string `json:"message"`
}

// TestEnqueueAndProcessJobWithVerification tests the full lifecycle of a job and verifies the handler execution.
func TestEnqueueAndProcessJobWithVerification(t *testing.T) {
	// Assuming getRedisConfig() provides a valid *queue.RedisConfig.
	redisConfig := getRedisConfig()

	client, err := queue.NewClient(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create and enqueue a job.
	payload := TestJobPayload{Message: "Hello, Test Queue!"}
	job := queue.NewJob(testJobType, payload)
	if _, err = client.EnqueueJob(job); err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	worker, err := queue.NewWorker(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}

	// Shared state to verify handler execution.
	var handlerExecuted bool
	var handlerExecutionLock sync.Mutex

	// Define the test job handler function with verification logic.
	testJobHandler := func(ctx context.Context, job *queue.Job) error {
		var payload TestJobPayload
		if err := job.DecodePayload(&payload); err != nil {
			return err
		}

		// Handler logic goes here. For demonstration, we're just logging.
		t.Logf("Processing job with message: %s\n", payload.Message)

		// Mark the handler as executed.
		handlerExecutionLock.Lock()
		handlerExecuted = true
		handlerExecutionLock.Unlock()

		return nil
	}

	// Register the test job handler with the worker.
	handler := queue.NewHandler(testJobType, testJobHandler)
	if err = worker.RegisterHandler(handler); err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Start the worker in a goroutine.
	go func() {
		if err := worker.Start(); err != nil {
			t.Errorf("Worker failed to start: %v", err)
		}
	}()
	defer worker.Stop()

	// Wait a reasonable amount of time for the job to be processed.
	time.Sleep(5 * time.Second) // Adjust this sleep time based on expected job processing time.

	// Check if the handler was executed.
	handlerExecutionLock.Lock()
	if !handlerExecuted {
		t.Errorf("Handler was not executed")
	}
	handlerExecutionLock.Unlock()
}
