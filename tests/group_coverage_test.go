package tests

import (
	"context"
	"testing"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Group.RegisterHandler ---

func TestGroupRegisterHandler(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig())
	require.NoError(t, err)

	mwCalled := false
	mw := func(next queue.HandlerFunc) queue.HandlerFunc {
		return func(ctx context.Context, job *queue.Job) error {
			mwCalled = true
			return next(ctx, job)
		}
	}

	group := worker.Group("email")
	group.Use(mw)

	handler := queue.NewHandler("email:send",
		func(_ context.Context, _ *queue.Job) error {
			return nil
		},
	)
	err = group.RegisterHandler(handler)
	require.NoError(t, err)

	// Verify middleware was applied by processing.
	job := &queue.Job{Type: "email:send", Payload: "data"}
	err = handler.Process(context.Background(), job)
	require.NoError(t, err)
	assert.True(t, mwCalled)
}

func TestGroupRegisterHandler_NoMiddleware(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig())
	require.NoError(t, err)

	group := worker.Group("plain")

	handler := queue.NewHandler("plain:job",
		func(_ context.Context, _ *queue.Job) error {
			return nil
		},
	)
	err = group.RegisterHandler(handler)
	require.NoError(t, err)
}

// --- Worker.Group returns same group ---

func TestWorkerGroupReturnsSame(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig())
	require.NoError(t, err)

	g1 := worker.Group("email")
	g2 := worker.Group("email")
	assert.Same(t, g1, g2)
}
