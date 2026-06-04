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

func TestWorkerRunCanceledContext(t *testing.T) {
	redisConfig := getRedisConfig()

	// Initialize worker with minimal configuration
	worker, err := queue.NewWorker(redisConfig)
	require.NoError(t, err, "Failed to create worker")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- worker.Run(ctx)
	}()

	// Allow some time for worker to start
	time.Sleep(1 * time.Second)
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err, "Failed to stop worker")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for worker shutdown")
	}
}

func TestWorkerRegister(t *testing.T) {
	redisConfig := getRedisConfig()

	worker, err := queue.NewWorker(redisConfig)
	require.NoError(t, err, "Failed to create worker")

	handlerFunc := func(context.Context, *queue.Delivery) error {
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

	handler := newHandler(t, "test_job", func(context.Context, *queue.Delivery) error {
		// Handler logic
		return nil
	})

	err = worker.RegisterHandler(handler)
	require.NoError(t, err, "Failed to register handler using RegisterHandler")
}

func TestWorkerProcessesStringPayload(t *testing.T) {
	redisConfig := getRedisConfig()
	const queueName = "worker_string_payload_test"
	const jobType = "worker_string_payload"

	manager := setupTestManager()
	defer func() {
		_ = manager.DeleteQueue(queueName, true)
	}()

	worker, err := queue.NewWorker(redisConfig,
		queue.WithWorkerQueue(queueName, 1),
		queue.WithWorkerConcurrency(1),
	)
	require.NoError(t, err, "Failed to create worker")

	processed := make(chan string, 1)
	err = worker.Register(jobType, func(_ context.Context, delivery *queue.Delivery) error {
		var payload string
		if err := delivery.DecodePayload(&payload); err != nil {
			return err
		}
		processed <- payload
		return nil
	}, queue.WithJobQueue(queueName))
	require.NoError(t, err, "Failed to register string payload handler")

	runWorker(t, worker)

	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err, "Failed to create client")
	defer func() {
		assert.NoError(t, client.Close(), "Failed to close client")
	}()

	_, err = client.Enqueue(jobType, "hello", queue.WithQueue(queueName))
	require.NoError(t, err, "Failed to enqueue string payload job")

	select {
	case got := <-processed:
		assert.Equal(t, "hello", got)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for string payload job")
	}
}

func TestWorkerWithWorkerErrorHandler(t *testing.T) {
	redisConfig := getRedisConfig() // Ensure this returns a valid configuration

	errorHandler := NewCustomWorkerErrorHandler()

	worker, err := queue.NewWorker(redisConfig, queue.WithWorkerErrorHandler(errorHandler))
	require.NoError(t, err, "Failed to create worker with error handler")

	jobType := "failJob"
	err = worker.Register(jobType, func(context.Context, *queue.Delivery) error {
		return ErrIntentionalJobFailure
	})
	require.NoError(t, err, "Failed to register failing job handler")

	runWorker(t, worker)

	time.Sleep(2 * time.Second) // Adjusted wait time for startup

	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err, "Failed to create client")
	_, err = client.Enqueue(jobType, map[string]any{"key": "value"})
	require.NoError(t, err, "Failed to enqueue job")

	time.Sleep(2 * time.Second) // Adjusted wait time for job processing

	assert.NotEmpty(t, errorHandler.Errors(), "Expected the custom error handler to capture a processing error")
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
func (h *CustomWorkerErrorHandler) HandleError(err error, delivery *queue.Delivery) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errors = append(h.errors, err)
}

func (h *CustomWorkerErrorHandler) Errors() []error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]error{}, h.errors...)
}
