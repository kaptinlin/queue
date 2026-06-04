package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
)

// --- Group.RegisterHandler ---

func TestGroupRegisterHandler(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig())
	require.NoError(t, err)

	mwCalled := false
	mw := func(next queue.HandlerFunc) queue.HandlerFunc {
		return func(ctx context.Context, delivery *queue.Delivery) error {
			mwCalled = true
			return next(ctx, delivery)
		}
	}

	group := worker.Group("email")
	group.Use(mw)

	handler := newHandler(t, "email:send",
		func(_ context.Context, _ *queue.Delivery) error {
			return nil
		},
	)
	err = group.RegisterHandler(handler)
	require.NoError(t, err)

	// Group.RegisterHandler registers a clone, leaving the original handler untouched.
	err = handler.Process(context.Background(), nil)
	require.NoError(t, err)
	assert.False(t, mwCalled)
}

func TestGroupRegisterHandler_NoMiddleware(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig())
	require.NoError(t, err)

	group := worker.Group("plain")

	handler := newHandler(t, "plain:job",
		func(_ context.Context, _ *queue.Delivery) error {
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
