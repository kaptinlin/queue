package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
)

func TestNewSkipRetryError(t *testing.T) {
	err := queue.NewSkipRetryError("invalid payload")

	assert.Error(t, err)
	assert.ErrorIs(t, err, queue.ErrSkipRetry)
	assert.NotErrorIs(t, err, asynq.SkipRetry)
}

func TestNewRetryWithoutFailureError(t *testing.T) {
	err := queue.NewRetryWithoutFailureError(assert.AnError)

	assert.ErrorIs(t, err, queue.ErrRetryWithoutFailure)
	assert.ErrorIs(t, err, assert.AnError)
	assert.ErrorIs(t, queue.NewRetryWithoutFailureError(nil), queue.ErrRetryWithoutFailure)
}

func TestRateLimitErrorError(t *testing.T) {
	err := queue.NewRateLimitError(5 * time.Second)

	var rateLimitErr *queue.RateLimitError
	require.ErrorAs(t, err, &rateLimitErr)
	assert.Equal(t, 5*time.Second, rateLimitErr.RetryAfter)
	assert.ErrorIs(t, err, queue.ErrRetryWithoutFailure)
}

func TestIsRateLimitError_Wrapped(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", queue.NewRateLimitError(5*time.Second))

	assert.True(t, queue.IsRateLimitError(err))
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "direct RateLimitError",
			err:  queue.NewRateLimitError(time.Second),
			want: true,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "unrelated error",
			err:  queue.ErrEnqueueJob,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := queue.IsRateLimitError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}
