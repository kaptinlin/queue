package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
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

// --- Handler validation ---

func TestNewHandler_EmptyJobType(t *testing.T) {
	handler, err := queue.NewHandler("", func(_ context.Context,
		_ *queue.Delivery) error {
		return nil
	})
	assert.Nil(t, handler)
	assert.ErrorIs(t, err, queue.ErrNoJobTypeSpecified)
}

func TestNewHandler_EmptyQueue(t *testing.T) {
	handler, err := queue.NewHandler("test", func(_ context.Context,
		_ *queue.Delivery) error {
		return nil
	}, queue.WithJobQueue(""))
	assert.Nil(t, handler)
	assert.ErrorIs(t, err, queue.ErrNoJobQueueSpecified)
}

func TestNewHandler_NilFunc(t *testing.T) {
	handler, err := queue.NewHandler("test", nil)
	assert.Nil(t, handler)
	assert.ErrorIs(t, err, queue.ErrNoHandlerFuncSpecified)
}

func TestWorkerRegisterHandler_NilHandler(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig())
	require.NoError(t, err)

	err = worker.RegisterHandler(nil)
	assert.ErrorIs(t, err, queue.ErrInvalidHandler)
}

func TestWorkerRegisterHandler_Duplicate(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig())
	require.NoError(t, err)

	h := func(_ context.Context, _ *queue.Delivery) error {
		return nil
	}
	err = worker.Register("dup_job", h)
	require.NoError(t, err)

	err = worker.Register("dup_job", h)
	assert.ErrorIs(t, err, queue.ErrHandlerAlreadyRegistered)
}

// --- Worker.Run already started ---

func TestWorkerRunAlreadyStarted(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- worker.Run(ctx)
	}()

	// Allow time for the worker to start.
	time.Sleep(1 * time.Second)

	err = worker.Run(context.Background())
	assert.ErrorIs(t, err, queue.ErrWorkerAlreadyStarted)

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for worker shutdown")
	}
}
