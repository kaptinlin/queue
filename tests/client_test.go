package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
)

func TestClient_Enqueue(t *testing.T) {
	redisConfig := getRedisConfig()

	client, err := queue.NewClient(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	jobType := "testEnqueueJob"
	payload := map[string]interface{}{"key": "value"}

	_, err = client.Enqueue(jobType, payload)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}
}

func TestClient_EnqueueJob(t *testing.T) {
	redisConfig := getRedisConfig()

	client, err := queue.NewClient(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	jobType := "testEnqueueJobJob"
	payload := map[string]interface{}{"key": "value"}
	job := queue.NewJob(jobType, payload)

	_, err = client.EnqueueJob(job)
	if err != nil {
		t.Fatalf("EnqueueJob failed: %v", err)
	}
}

func TestClient_WithClientRetention(t *testing.T) {
	redisConfig := getRedisConfig()

	retentionPeriod := 24 * time.Hour
	client, err := queue.NewClient(redisConfig,
		queue.WithClientRetention(retentionPeriod),
	)
	if err != nil {
		t.Fatalf("Failed to create client with retention: %v", err)
	}
	defer client.Close()

	// Here, you'd enqueue a job and then assert that it has the expected retention period.
	// This likely requires inspecting the job metadata in Redis directly to verify the retention is as expected.
	// Note: This test is more challenging to implement without support from the library to inspect job metadata.
	// You may need to use Redis commands directly to inspect the job's retention setting, which requires knowledge
	// of how the library stores this data.
}

func TestClient_WithClientErrorHandler(t *testing.T) {
	redisConfig := getRedisConfig() // Ensure this returns a valid Redis configuration

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
	if err := client.Close(); err != nil {
		t.Errorf("Failed to close client: %v", err)
	}
}

// CustomClientErrorHandler implements the queue.ClientErrorHandler interface.
type CustomClientErrorHandler struct {
	mu     sync.Mutex
	errors []error
}

// HandleError captures enqueue errors.
func (h *CustomClientErrorHandler) HandleError(err error, context map[string]interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errors = append(h.errors, err)
}
