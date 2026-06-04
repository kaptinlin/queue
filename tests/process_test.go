package tests

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
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
		assert.NoError(t, client.Close(), "Failed to close client")
	}()

	// Define the job payload and enqueue a new job.
	payload := TestJobPayload{Message: "Hello, Test Queue!"}
	job := newJob(t, testJobType, payload)

	// Initialize the worker responsible for processing jobs.
	worker, err := queue.NewWorker(redisConfig)
	require.NoError(t, err, "Failed to create worker")

	// Use a WaitGroup to wait for the handler execution.
	var wg sync.WaitGroup

	// Define and register the job handler.
	testJobHandler := func(ctx context.Context, delivery *queue.Delivery) error {
		defer wg.Done() // Signal that the job has been processed.

		var payload TestJobPayload
		if err := delivery.DecodePayload(&payload); err != nil {
			return err
		}

		// Logging for demonstration. Replace with actual job processing logic.
		t.Logf("Processing job with message: %s\n", payload.Message)

		return nil
	}

	// Register the job type and its handler with the worker.
	err = worker.Register(testJobType, testJobHandler)
	require.NoError(t, err, "Failed to register job handler")

	runWorker(t, worker)

	// Enqueue the job after worker is ready
	wg.Add(1) // Expecting one job to be processed
	_, err = client.EnqueueJob(job)
	require.NoError(t, err, "Failed to enqueue job")

	// Wait for the job handler to execute.
	wg.Wait()
}
