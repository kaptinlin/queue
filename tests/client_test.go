package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
)

func TestClientEnqueue(t *testing.T) {
	redisConfig := getRedisConfig()

	client, err := queue.NewClient(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Stop()

	jobType := "testEnqueueJob"
	payload := map[string]interface{}{"key": "value"}

	_, err = client.Enqueue(jobType, payload)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}
}

func TestClientEnqueueJob(t *testing.T) {
	redisConfig := getRedisConfig()

	client, err := queue.NewClient(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Stop()

	jobType := "testEnqueueJobJob"
	payload := map[string]interface{}{"key": "value"}
	job := queue.NewJob(jobType, payload)

	_, err = client.EnqueueJob(job)
	if err != nil {
		t.Fatalf("EnqueueJob failed: %v", err)
	}
}

func TestClientWithClientRetention(t *testing.T) {
	redisConfig := getRedisConfig()

	retentionPeriod := 24 * time.Hour
	client, err := queue.NewClient(redisConfig,
		queue.WithClientRetention(retentionPeriod),
	)
	if err != nil {
		t.Fatalf("Failed to create client with retention: %v", err)
	}
	defer client.Stop()
}

func TestClientWithClientErrorHandler(t *testing.T) {
	redisConfig := getRedisConfig()

	handler := &CustomClientErrorHandler{}

	// Initialize the client with the custom error handler
	client, err := queue.NewClient(redisConfig, queue.WithClientErrorHandler(handler))
	if err != nil {
		t.Fatalf("Failed to create client with custom error handler: %v", err)
	}

	// Create a job with an invalid configuration that is known to cause ConvertToAsynqTask to return an error
	jobType := "" // Intentionally left blank to trigger an error in ConvertToAsynqTask
	payload := map[string]interface{}{"key": "value"}

	// Attempt to enqueue the job, which should fail at the job conversion step
	_, err = client.Enqueue(jobType, payload)
	if err == nil {
		t.Errorf("Expected an error from enqueueing a job with invalid configuration, but got nil")
	}

	// Check if the custom error handler was invoked with the conversion error
	if len(handler.errors) == 0 {
		t.Errorf("Expected the custom error handler to be invoked with a conversion error, but it was not")
	}

	// Clean up resources
	if err := client.Stop(); err != nil {
		t.Errorf("Failed to stop client: %v", err)
	}
}

// CustomClientErrorHandler implements the queue.ClientErrorHandler interface.
type CustomClientErrorHandler struct {
	mu     sync.Mutex
	errors []error
}

// HandleError captures enqueue errors.
func (h *CustomClientErrorHandler) HandleError(err error, job *queue.Job) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errors = append(h.errors, err)
}
