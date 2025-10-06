package tests

import (
	"context"
	"sync"
	"testing"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define a test-specific job type and payload.
const testJobType = "test_job"

type TestJobPayload struct {
	Message string `json:"message"`
}

// TestEnqueueAndProcessJobWithVerification tests the full lifecycle of a job, ensuring that the job handler executes.
func TestEnqueueAndProcessJobWithVerification(t *testing.T) {
	redisConfig := getRedisConfig()

	// Initialize the client to interact with the queue.
	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err, "Failed to create client")
	defer func() {
		assert.NoError(t, client.Stop(), "Failed to stop client")
	}()

	// Define the job payload and enqueue a new job.
	payload := TestJobPayload{Message: "Hello, Test Queue!"}
	job := queue.NewJob(testJobType, payload)
	_, err = client.EnqueueJob(job)
	require.NoError(t, err, "Failed to enqueue job")

	// Initialize the worker responsible for processing jobs.
	worker, err := queue.NewWorker(redisConfig)
	require.NoError(t, err, "Failed to create worker")

	// Use a WaitGroup to wait for the handler execution.
	var wg sync.WaitGroup
	wg.Add(1) // Expecting one job to be processed

	// Define and register the job handler.
	testJobHandler := func(ctx context.Context, job *queue.Job) error {
		defer wg.Done() // Signal that the job has been processed.

		var payload TestJobPayload
		if err := job.DecodePayload(&payload); err != nil {
			return err
		}

		// Logging for demonstration. Replace with actual job processing logic.
		t.Logf("Processing job with message: %s\n", payload.Message)

		return nil
	}

	// Register the job type and its handler with the worker.
	err = worker.Register(testJobType, testJobHandler)
	require.NoError(t, err, "Failed to register job handler")

	// Start the worker in a separate goroutine to process jobs.
	go func() {
		if err := worker.Start(); err != nil {
			t.Errorf("Worker failed to start: %v", err)
		}
	}()
	defer func() {
		assert.NoError(t, worker.Stop(), "Failed to stop worker")
	}() // Ensure the worker is stopped after the test.

	// Wait for the job handler to execute.
	wg.Wait()
}
