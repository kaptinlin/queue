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

// --- NewHandler options ---

func TestNewHandlerWithOptions(t *testing.T) {
	handler := newHandler(t, "test",
		func(_ context.Context, _ *queue.Delivery) error {
			return nil
		},
		queue.WithJobQueue("critical"),
		queue.WithJobTimeout(5*time.Second),
	)
	assert.Equal(t, "critical", handler.Queue())
	assert.Equal(t, 5*time.Second, handler.Timeout())
}

// --- Handler middleware option ---

func TestHandlerWithSingleMiddleware(t *testing.T) {
	called := false
	mw := func(next queue.HandlerFunc) queue.HandlerFunc {
		return func(ctx context.Context, delivery *queue.Delivery) error {
			called = true
			return next(ctx, delivery)
		}
	}

	handler := newHandler(t, "test",
		func(_ context.Context, _ *queue.Delivery) error {
			return nil
		},
		queue.WithMiddleware(mw),
	)

	err := handler.Process(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, called)
}

// --- WithMiddleware option ---

func TestHandlerWithMiddlewareOption(t *testing.T) {
	called := false
	mw := func(next queue.HandlerFunc) queue.HandlerFunc {
		return func(ctx context.Context, delivery *queue.Delivery) error {
			called = true
			return next(ctx, delivery)
		}
	}

	handler := newHandler(t, "test",
		func(_ context.Context, _ *queue.Delivery) error {
			return nil
		},
		queue.WithMiddleware(mw),
	)

	err := handler.Process(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestHandlerMiddlewareOrder(t *testing.T) {
	t.Parallel()

	calls := make([]string, 0, 5)
	middleware := func(name string) queue.MiddlewareFunc {
		return func(next queue.HandlerFunc) queue.HandlerFunc {
			return func(ctx context.Context, delivery *queue.Delivery) error {
				calls = append(calls, name+":before")
				err := next(ctx, delivery)
				calls = append(calls, name+":after")
				return err
			}
		}
	}

	handler := newHandler(t, "test",
		func(_ context.Context, _ *queue.Delivery) error {
			calls = append(calls, "handler")
			return nil
		},
		queue.WithMiddleware(middleware("first"), middleware("second")),
	)

	err := handler.Process(context.Background(), nil)
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
