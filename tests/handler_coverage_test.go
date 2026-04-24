package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
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

func TestHandlerMiddlewareOrder(t *testing.T) {
	t.Parallel()

	calls := make([]string, 0, 5)
	middleware := func(name string) queue.MiddlewareFunc {
		return func(next queue.HandlerFunc) queue.HandlerFunc {
			return func(ctx context.Context, job *queue.Job) error {
				calls = append(calls, name+":before")
				err := next(ctx, job)
				calls = append(calls, name+":after")
				return err
			}
		}
	}

	handler := queue.NewHandler("test",
		func(_ context.Context, _ *queue.Job) error {
			calls = append(calls, "handler")
			return nil
		},
		queue.WithMiddleware(middleware("first"), middleware("second")),
	)

	err := handler.Process(context.Background(), queue.NewJob("test", nil))
	require.NoError(t, err)

	want := []string{
		"first:before",
		"second:before",
		"handler",
		"second:after",
		"first:after",
	}
	if diff := cmp.Diff(want, calls); diff != "" {
		t.Errorf("middleware call order mismatch (-want +got):\n%s", diff)
	}
}
