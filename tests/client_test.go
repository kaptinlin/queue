package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientEnqueue(t *testing.T) {
	redisConfig := getRedisConfig()

	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err, "Failed to create client")
	defer func() {
		assert.NoError(t, client.Stop(), "Failed to stop client")
	}()

	jobType := "testEnqueueJob"
	payload := map[string]interface{}{"key": "value"}

	_, err = client.Enqueue(jobType, payload)
	require.NoError(t, err, "Enqueue failed")
}

func TestClientEnqueueJob(t *testing.T) {
	redisConfig := getRedisConfig()

	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err, "Failed to create client")
	defer func() {
		assert.NoError(t, client.Stop(), "Failed to stop client")
	}()

	jobType := "testEnqueueJobJob"
	payload := map[string]interface{}{"key": "value"}
	job := queue.NewJob(jobType, payload)

	_, err = client.EnqueueJob(job)
	require.NoError(t, err, "EnqueueJob failed")
}

func TestClientWithClientRetention(t *testing.T) {
	redisConfig := getRedisConfig()

	retentionPeriod := 24 * time.Hour
	client, err := queue.NewClient(redisConfig,
		queue.WithClientRetention(retentionPeriod),
	)
	require.NoError(t, err, "Failed to create client with retention")
	defer func() {
		assert.NoError(t, client.Stop(), "Failed to stop client")
	}()
}

func TestClientWithClientErrorHandler(t *testing.T) {
	redisConfig := getRedisConfig()

	handler := &CustomClientErrorHandler{}

	// Initialize the client with the custom error handler
	client, err := queue.NewClient(redisConfig, queue.WithClientErrorHandler(handler))
	require.NoError(t, err, "Failed to create client with custom error handler")

	// Create a job with an invalid configuration that is known to cause ConvertToAsynqTask to return an error
	jobType := "" // Intentionally left blank to trigger an error in ConvertToAsynqTask
	payload := map[string]interface{}{"key": "value"}

	// Attempt to enqueue the job, which should fail at the job conversion step
	_, err = client.Enqueue(jobType, payload)
	assert.Error(t, err, "Expected an error from enqueueing a job with invalid configuration")

	// Check if the custom error handler was invoked with the conversion error
	assert.NotEmpty(t, handler.errors, "Expected the custom error handler to be invoked with a conversion error")

	// Clean up resources
	assert.NoError(t, client.Stop(), "Failed to stop client")
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
