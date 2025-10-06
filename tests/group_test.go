package tests

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGroupMiddleware ensures that group-specific middleware is correctly applied to jobs processed by handlers within the group.
func TestGroupMiddleware(t *testing.T) {
	redisConfig := getRedisConfig()

	// Define group-specific middleware to count processed jobs within the group.
	var (
		mu            sync.Mutex
		processedJobs int
	)
	groupMiddleware := func(next queue.HandlerFunc) queue.HandlerFunc {
		return func(ctx context.Context, job *queue.Job) error {
			mu.Lock()
			processedJobs++
			mu.Unlock()

			log.Printf("Group processing job: %v", job.Type)
			return next(ctx, job)
		}
	}

	worker, err := queue.NewWorker(redisConfig)
	require.NoError(t, err, "Failed to create worker")

	// Create a group and apply the group-specific middleware
	emailGroup := worker.Group("email")
	emailGroup.Use(groupMiddleware)

	// Register a dummy job handler within the group
	jobType := "sendEmail"
	err = emailGroup.Register(jobType, func(ctx context.Context, job *queue.Job) error {
		return nil // Simulate successful job processing
	})
	require.NoError(t, err, "Failed to register job handler")

	// Start the worker in a goroutine to process jobs.
	go func() {
		if err := worker.Start(); err != nil {
			log.Fatalf("Failed to start worker: %v", err)
		}
	}()
	defer func() {
		assert.NoError(t, worker.Stop(), "Failed to stop worker")
	}() // Ensure worker is stopped after the test.

	// Allow some time for the worker to initialize.
	time.Sleep(2 * time.Second)

	// Enqueue jobs to be processed by the handler within the email group.
	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err, "Failed to create client")
	for i := 0; i < 5; i++ {
		_, err := client.Enqueue(jobType, map[string]interface{}{"key": "value"})
		require.NoError(t, err, "Failed to enqueue job")
	}

	// Wait a bit for jobs to be processed.
	time.Sleep(5 * time.Second) // Adjust this duration based on job processing time.

	// Verify that the group-specific middleware processed the expected number of jobs.
	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, processedJobs, 5, "Expected at least 5 processed jobs by group middleware")
}
