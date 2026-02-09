package tests

import (
	"context"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Handler.WithOptions ---

func TestHandlerWithOptions(t *testing.T) {
	handler := queue.NewHandler("test",
		func(_ context.Context, _ *queue.Job) error {
			return nil
		},
	)
	assert.Equal(t, queue.DefaultQueue, handler.JobQueue)

	handler.WithOptions(
		queue.WithJobQueue("critical"),
		queue.WithJobTimeout(5*time.Second),
	)
	assert.Equal(t, "critical", handler.JobQueue)
	assert.Equal(t, 5*time.Second, handler.JobTimeout)
}

// --- Handler.Use ---

func TestHandlerUse(t *testing.T) {
	called := false
	mw := func(next queue.HandlerFunc) queue.HandlerFunc {
		return func(ctx context.Context, job *queue.Job) error {
			called = true
			return next(ctx, job)
		}
	}

	handler := queue.NewHandler("test",
		func(_ context.Context, _ *queue.Job) error {
			return nil
		},
	)
	handler.Use(mw)

	job := &queue.Job{Type: "test", Payload: "data"}
	err := handler.Process(context.Background(), job)
	require.NoError(t, err)
	assert.True(t, called)
}

// --- WithMiddleware option ---

func TestHandlerWithMiddlewareOption(t *testing.T) {
	called := false
	mw := func(next queue.HandlerFunc) queue.HandlerFunc {
		return func(ctx context.Context, job *queue.Job) error {
			called = true
			return next(ctx, job)
		}
	}

	handler := queue.NewHandler("test",
		func(_ context.Context, _ *queue.Job) error {
			return nil
		},
		queue.WithMiddleware(mw),
	)

	job := &queue.Job{Type: "test", Payload: "data"}
	err := handler.Process(context.Background(), job)
	require.NoError(t, err)
	assert.True(t, called)
}
