package queue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  workerConfig
		wantErr error
	}{
		{
			name:    "valid config",
			config:  workerConfig{Concurrency: 4, Queues: map[string]int{"q": 1}},
			wantErr: nil,
		},
		{
			name:    "zero concurrency",
			config:  workerConfig{Concurrency: 0, Queues: map[string]int{"q": 1}},
			wantErr: ErrInvalidWorkerConcurrency,
		},
		{
			name:    "negative concurrency",
			config:  workerConfig{Concurrency: -1, Queues: map[string]int{"q": 1}},
			wantErr: ErrInvalidWorkerConcurrency,
		},
		{
			name:    "negative stop timeout",
			config:  workerConfig{Concurrency: 1, StopTimeout: -time.Second, Queues: map[string]int{"q": 1}},
			wantErr: ErrInvalidWorkerStopTimeout,
		},
		{
			name:    "empty queues",
			config:  workerConfig{Concurrency: 1, Queues: map[string]int{}},
			wantErr: ErrInvalidWorkerQueues,
		},
		{
			name:    "empty queue name",
			config:  workerConfig{Concurrency: 1, Queues: map[string]int{"": 1}},
			wantErr: ErrInvalidWorkerQueues,
		},
		{
			name:    "zero queue priority",
			config:  workerConfig{Concurrency: 1, Queues: map[string]int{"q": 0}},
			wantErr: ErrInvalidWorkerQueues,
		},
		{
			name:    "nil queues",
			config:  workerConfig{Concurrency: 1},
			wantErr: ErrInvalidWorkerQueues,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsFailure(t *testing.T) {
	w := &Worker{}

	assert.True(t, w.isFailure(assert.AnError))
	assert.False(t, w.isFailure(&RateLimitError{RetryAfter: time.Second}))
	assert.False(t, w.isFailure(ErrRetryWithoutFailure))
	assert.False(t, w.isFailure(NewRetryWithoutFailureError(assert.AnError)))
}

func TestAsynqHandlerErrorTranslatesSkipRetry(t *testing.T) {
	err := NewSkipRetryError("invalid payload")

	got := asynqHandlerError(err)

	assert.ErrorIs(t, got, ErrSkipRetry)
	assert.ErrorIs(t, got, asynq.SkipRetry)
}

func TestAsynqHandlerErrorLeavesNonSkipRetry(t *testing.T) {
	err := fmt.Errorf("plain error")

	got := asynqHandlerError(err)

	assert.Same(t, err, got)
	assert.NotErrorIs(t, got, asynq.SkipRetry)
}

func TestWorkerGroup_ReusesGroupByName(t *testing.T) {
	w := &Worker{groups: make(map[string]*Group)}

	email := w.Group("email")
	sameEmail := w.Group("email")
	other := w.Group("other")

	assert.Same(t, email, sameEmail)
	assert.NotSame(t, email, other)
	assert.Same(t, w, email.worker)
}

func TestWorkerAssemblyRejectsNilMiddleware(t *testing.T) {
	t.Parallel()

	worker, err := NewWorker(DefaultRedisConfig())
	require.NoError(t, err)

	assert.ErrorIs(t, worker.Use(nil), ErrInvalidMiddleware)

	group := worker.Group("email")
	assert.ErrorIs(t, group.Use(nil), ErrInvalidMiddleware)

	_, err = NewHandler(
		"email:send",
		func(context.Context, *Delivery) error { return nil },
		WithMiddleware(nil),
	)
	assert.ErrorIs(t, err, ErrInvalidMiddleware)
}

func TestWorkerAssemblyRejectsStartedWorker(t *testing.T) {
	t.Parallel()

	worker, err := NewWorker(DefaultRedisConfig())
	require.NoError(t, err)

	group := worker.Group("email")
	middleware := func(next HandlerFunc) HandlerFunc { return next }
	handler := func(context.Context, *Delivery) error { return nil }
	registeredHandler, err := NewHandler("email:registered", handler)
	require.NoError(t, err)

	worker.started.Store(true)

	assert.ErrorIs(t, worker.Use(middleware), ErrWorkerAlreadyStarted)
	assert.ErrorIs(t, worker.Register("email:send", handler), ErrWorkerAlreadyStarted)
	assert.ErrorIs(t, worker.RegisterHandler(registeredHandler), ErrWorkerAlreadyStarted)
	assert.ErrorIs(t, group.Use(middleware), ErrWorkerAlreadyStarted)
	assert.ErrorIs(t, group.Register("email:group", handler), ErrWorkerAlreadyStarted)
	assert.ErrorIs(t, group.RegisterHandler(registeredHandler), ErrWorkerAlreadyStarted)
}

func TestWorkerAssemblyRejectsStoppedWorker(t *testing.T) {
	t.Parallel()

	worker, err := NewWorker(DefaultRedisConfig())
	require.NoError(t, err)

	worker.stopped.Store(true)

	middleware := func(next HandlerFunc) HandlerFunc { return next }
	assert.ErrorIs(t, worker.Use(middleware), ErrWorkerStopped)
	assert.ErrorIs(t, worker.Register("email:send", func(context.Context, *Delivery) error {
		return nil
	}), ErrWorkerStopped)
}

func TestGroupRegister_ComposesGroupMiddlewareBeforeOptions(t *testing.T) {
	w := &Worker{
		groups:   make(map[string]*Group),
		handlers: make(map[string]*Handler),
	}
	calls := make([]string, 0, 2)
	middleware := func(name string) MiddlewareFunc {
		return func(next HandlerFunc) HandlerFunc {
			return func(ctx context.Context, delivery *Delivery) error {
				calls = append(calls, name)
				return next(ctx, delivery)
			}
		}
	}

	group := w.Group("email")
	assert.NoError(t, group.Use(middleware("group")))
	err := group.Register(
		"email:send",
		func(context.Context, *Delivery) error { return nil },
		WithMiddleware(middleware("handler")),
	)
	assert.NoError(t, err)

	handler := w.handlers["email:send"]
	assert.NotNil(t, handler)
	assert.NoError(t, handler.Process(context.Background(), &Delivery{jobType: "email:send"}))
	want := []string{"group", "handler"}
	if diff := cmp.Diff(want, calls); diff != "" {
		t.Errorf("middleware calls mismatch (-want +got):\n%s", diff)
	}
}
