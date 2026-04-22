package tests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
)

var ErrIntentionalJobFailure = errors.New("intentional job failure")

func TestWorkerStartStop(t *testing.T) {
	redisConfig := getRedisConfig()

	// Initialize worker with minimal configuration
	worker, err := queue.NewWorker(redisConfig)
	require.NoError(t, err, "Failed to create worker")

	// Start worker in a goroutine
	go func() {
		assert.NoError(t, worker.Start(), "Failed to start worker")
	}()

	// Allow some time for worker to start
	time.Sleep(1 * time.Second)

	// Stop worker
	err = worker.Stop()
	assert.NoError(t, err, "Failed to stop worker")
}

func TestWorkerRegister(t *testing.T) {
	redisConfig := getRedisConfig()

	worker, err := queue.NewWorker(redisConfig)
	require.NoError(t, err, "Failed to create worker")

	handlerFunc := func(ctx context.Context, job *queue.Job) error {
		// Handler logic here
		return nil
	}

	// Register a job handler
	err = worker.Register("test_job", handlerFunc)
	require.NoError(t, err, "Failed to register handler")
}

func TestWorkerRegisterHandler(t *testing.T) {
	redisConfig := getRedisConfig()

	worker, err := queue.NewWorker(redisConfig)
	require.NoError(t, err, "Failed to create worker")

	handler := queue.NewHandler("test_job", func(ctx context.Context, job *queue.Job) error {
		// Handler logic
		return nil
	})

	err = worker.RegisterHandler(handler)
	require.NoError(t, err, "Failed to register handler using RegisterHandler")
}

func TestWorkerWithWorkerErrorHandler(t *testing.T) {
	redisConfig := getRedisConfig() // Ensure this returns a valid configuration

	errorHandler := NewCustomWorkerErrorHandler()

	worker, err := queue.NewWorker(redisConfig, queue.WithWorkerErrorHandler(errorHandler))
	require.NoError(t, err, "Failed to create worker with error handler")

	jobType := "failJob"
	err = worker.Register(jobType, func(ctx context.Context, job *queue.Job) error {
		return ErrIntentionalJobFailure
	})
	require.NoError(t, err, "Failed to register failing job handler")

	go func() {
		assert.NoError(t, worker.Start(), "Failed to start worker")
	}()

	time.Sleep(2 * time.Second) // Adjusted wait time for startup

	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err, "Failed to create client")
	_, err = client.Enqueue(jobType, map[string]any{"key": "value"})
	require.NoError(t, err, "Failed to enqueue job")

	time.Sleep(2 * time.Second) // Adjusted wait time for job processing

	err = worker.Stop()
	assert.NoError(t, err, "Failed to stop worker")

	assert.NotEmpty(t, errorHandler.errors, "Expected the custom error handler to capture a processing error")
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
func (h *CustomWorkerErrorHandler) HandleError(err error, job *queue.Job) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errors = append(h.errors, err)
}
