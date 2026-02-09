package tests

import (
	"context"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewWorker validation ---

func TestNewWorker_NilRedisConfig(t *testing.T) {
	_, err := queue.NewWorker(nil)
	assert.ErrorIs(t, err, queue.ErrInvalidRedisConfig)
}

func TestNewWorker_InvalidRedisConfig(t *testing.T) {
	cfg := &queue.RedisConfig{Network: "bad", Addr: ""}
	_, err := queue.NewWorker(cfg)
	assert.Error(t, err)
}

// --- RegisterHandler edge cases ---

func TestWorkerRegisterHandler_EmptyJobType(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig())
	require.NoError(t, err)

	handler := queue.NewHandler("", func(_ context.Context,
		_ *queue.Job) error {
		return nil
	})
	err = worker.RegisterHandler(handler)
	assert.ErrorIs(t, err, queue.ErrNoJobTypeSpecified)
}

func TestWorkerRegisterHandler_EmptyQueue(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig())
	require.NoError(t, err)

	handler := queue.NewHandler("test", func(_ context.Context,
		_ *queue.Job) error {
		return nil
	}, queue.WithJobQueue(""))
	err = worker.RegisterHandler(handler)
	assert.ErrorIs(t, err, queue.ErrNoJobQueueSpecified)
}

func TestWorkerRegisterHandler_Duplicate(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig())
	require.NoError(t, err)

	h := func(_ context.Context, _ *queue.Job) error {
		return nil
	}
	err = worker.Register("dup_job", h)
	require.NoError(t, err)

	err = worker.Register("dup_job", h)
	assert.ErrorContains(t, err, "handler already registered")
}

// --- Worker.Start already started ---

func TestWorkerStartAlreadyStarted(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig())
	require.NoError(t, err)

	go func() { _ = worker.Start() }()

	// Allow time for the worker to start.
	time.Sleep(1 * time.Second)

	err = worker.Start()
	assert.ErrorIs(t, err, queue.ErrWorkerAlreadyStarted)

	assert.NoError(t, worker.Stop())
}
