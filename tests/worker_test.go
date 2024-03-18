package tests

import (
	"context"
	"errors"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
)

func TestWorker_StartStop(t *testing.T) {
	redisConfig := getRedisConfig()

	// Initialize worker with minimal configuration
	worker, err := queue.NewWorker(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}

	// Start worker in a goroutine
	go func() {
		if err := worker.Start(); err != nil {
			t.Errorf("Failed to start worker: %v", err)
		}
	}()

	// Allow some time for worker to start
	time.Sleep(1 * time.Second)

	// Stop worker
	worker.Stop()
}

func TestWorker_Register(t *testing.T) {
	redisConfig := getRedisConfig()

	worker, err := queue.NewWorker(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}

	handlerFunc := func(ctx context.Context, job *queue.Job) error {
		// Handler logic here
		return nil
	}

	// Register a job handler
	err = worker.Register("test_job", handlerFunc)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}
}

func TestWorker_RegisterHandler(t *testing.T) {
	redisConfig := getRedisConfig()

	worker, err := queue.NewWorker(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}

	handler := queue.NewHandler("test_job", func(ctx context.Context, job *queue.Job) error {
		// Handler logic
		return nil
	})

	err = worker.RegisterHandler(handler)
	if err != nil {
		t.Fatalf("Failed to register handler using RegisterHandler: %v", err)
	}
}

func TestWorker_WithWorkerErrorHandler(t *testing.T) {
	redisConfig := getRedisConfig() // Ensure this returns a valid configuration

	errorHandler := NewCustomWorkerErrorHandler()

	worker, err := queue.NewWorker(redisConfig, queue.WithWorkerErrorHandler(errorHandler))
	if err != nil {
		t.Fatalf("Failed to create worker with error handler: %v", err)
	}

	jobType := "failJob"
	if err := worker.Register(jobType, func(ctx context.Context, job *queue.Job) error {
		return errors.New("intentional job failure")
	}); err != nil {
		t.Fatalf("Failed to register failing job handler: %v", err)
	}

	go func() {
		if err := worker.Start(); err != nil {
			log.Fatalf("Failed to start worker: %v", err)
		}
	}()

	time.Sleep(2 * time.Second) // Adjusted wait time for startup

	client, _ := queue.NewClient(redisConfig)
	if _, err := client.Enqueue(jobType, map[string]interface{}{"key": "value"}); err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	time.Sleep(2 * time.Second) // Adjusted wait time for job processing

	worker.Stop()

	if len(errorHandler.errors) == 0 {
		t.Errorf("Expected the custom error handler to capture a processing error, but it did not")
	}
}

// CustomWorkerErrorHandler implements the queue.WorkerErrorHandler interface.
type CustomWorkerErrorHandler struct {
	mu     sync.Mutex
	errors []error
}

func NewCustomWorkerErrorHandler() *CustomWorkerErrorHandler {
	return &CustomWorkerErrorHandler{
		errors: make([]error, 0),
	}
}

// HandleError captures job processing errors.
func (h *CustomWorkerErrorHandler) HandleError(err error, context map[string]interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errors = append(h.errors, err)
}
