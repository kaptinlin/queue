package tests

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
)

// TestGlobalMiddleware ensures global middleware is applied to all jobs processed by the worker.
func TestGlobalMiddleware(t *testing.T) {
	redisConfig := getRedisConfig()

	// Define a simple global middleware to count jobs
	var (
		mu            sync.Mutex
		processedJobs int
	)
	globalMiddleware := func(next queue.HandlerFunc) queue.HandlerFunc {
		return func(ctx context.Context, job *queue.Job) error {
			mu.Lock()
			defer mu.Unlock()

			processedJobs++
			log.Printf("Processing job: %v", job.Type)
			return next(ctx, job)
		}
	}

	worker, err := queue.NewWorker(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}
	worker.Use(globalMiddleware) // Apply the global middleware

	// Register a dummy job handler
	jobType := "dummyJob"
	if err := worker.Register(jobType, func(ctx context.Context, job *queue.Job) error {
		return nil // Simulate successful job processing
	}); err != nil {
		t.Fatalf("Failed to register dummy job handler: %v", err)
	}

	// Start the worker in a goroutine
	go func() {
		if err := worker.Start(); err != nil {
			log.Fatalf("Failed to start worker: %v", err)
		}
	}()
	defer worker.Stop()

	// Give the worker some time to start
	time.Sleep(2 * time.Second)

	// Enqueue a couple of jobs
	client, _ := queue.NewClient(redisConfig)
	for i := 0; i < 5; i++ {
		if _, err := client.Enqueue(jobType, map[string]interface{}{"key": "value"}); err != nil {
			t.Fatalf("Failed to enqueue job: %v", err)
		}
	}

	// Wait a bit for jobs to be processed
	time.Sleep(5 * time.Second) // Adjusted wait time for job processing

	// Stop the worker and check the middleware count
	worker.Stop()

	mu.Lock()
	defer mu.Unlock()
	if processedJobs < 5 {
		t.Errorf("Expected at least 5 processed jobs, got %d", processedJobs)
	}
}

// TestScopedMiddleware ensures that scoped middleware is correctly applied to specific job handlers.
func TestScopedMiddleware(t *testing.T) {
	redisConfig := getRedisConfig()

	// Define a simple scoped middleware to count processed jobs.
	var (
		mu            sync.Mutex
		processedJobs int
	)
	scopedMiddleware := func(next queue.HandlerFunc) queue.HandlerFunc {
		return func(ctx context.Context, job *queue.Job) error {
			mu.Lock()
			processedJobs++
			mu.Unlock()

			log.Printf("Processing job: %v", job.Type)
			return next(ctx, job)
		}
	}

	worker, err := queue.NewWorker(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}

	// Register a dummy job handler with scoped middleware
	jobType := "scopedJob"
	handlerFunc := func(ctx context.Context, job *queue.Job) error {
		return nil // Simulate successful job processing
	}
	if err := worker.Register(jobType, handlerFunc, queue.WithMiddleware(scopedMiddleware)); err != nil {
		t.Fatalf("Failed to register dummy job handler with scoped middleware: %v", err)
	}

	// Start the worker in a goroutine to process jobs.
	go func() {
		if err := worker.Start(); err != nil {
			log.Fatalf("Failed to start worker: %v", err)
		}
	}()
	defer worker.Stop() // Ensure worker is stopped after the test.

	// Allow some time for the worker to initialize.
	time.Sleep(2 * time.Second)

	// Enqueue jobs to be processed by the scoped middleware-enhanced handler.
	client, _ := queue.NewClient(redisConfig)
	for i := 0; i < 3; i++ {
		if _, err := client.Enqueue(jobType, map[string]interface{}{"key": "value"}); err != nil {
			t.Fatalf("Failed to enqueue job: %v", err)
		}
	}

	// Wait a bit for jobs to be processed.
	time.Sleep(5 * time.Second) // Adjust this duration based on job processing time.

	// Verify that the scoped middleware processed the expected number of jobs.
	mu.Lock()
	defer mu.Unlock()
	if processedJobs < 3 {
		t.Errorf("Expected at least 3 scoped processed jobs, got %d", processedJobs)
	}
}
