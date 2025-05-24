package tests

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
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
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}

	// Create a group and apply the group-specific middleware
	emailGroup := worker.Group("email")
	emailGroup.Use(groupMiddleware)

	// Register a dummy job handler within the group
	jobType := "sendEmail"
	if err := emailGroup.Register(jobType, func(ctx context.Context, job *queue.Job) error {
		return nil // Simulate successful job processing
	}); err != nil {
		t.Fatalf("Failed to register job handler: %v", err)
	}

	// Start the worker in a goroutine to process jobs.
	go func() {
		if err := worker.Start(); err != nil {
			log.Fatalf("Failed to start worker: %v", err)
		}
	}()
	defer func() {
		if err := worker.Stop(); err != nil {
			t.Errorf("Failed to stop worker: %v", err)
		}
	}() // Ensure worker is stopped after the test.

	// Allow some time for the worker to initialize.
	time.Sleep(2 * time.Second)

	// Enqueue jobs to be processed by the handler within the email group.
	client, _ := queue.NewClient(redisConfig)
	for i := 0; i < 5; i++ {
		if _, err := client.Enqueue(jobType, map[string]interface{}{"key": "value"}); err != nil {
			t.Fatalf("Failed to enqueue job: %v", err)
		}
	}

	// Wait a bit for jobs to be processed.
	time.Sleep(5 * time.Second) // Adjust this duration based on job processing time.

	// Verify that the group-specific middleware processed the expected number of jobs.
	mu.Lock()
	defer mu.Unlock()
	if processedJobs < 5 {
		t.Errorf("Expected at least 5 processed jobs by group middleware, got %d", processedJobs)
	}
}
