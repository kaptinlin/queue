package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
)

func TestNewSkipRetryError(t *testing.T) {
	err := queue.NewSkipRetryError("invalid payload")

	assert.Error(t, err)
	assert.ErrorIs(t, err, queue.ErrSkipRetry)
}

func TestErrRateLimitError(t *testing.T) {
	err := queue.NewErrRateLimit(5 * time.Second)

	var rateLimitErr *queue.ErrRateLimit
	require.ErrorAs(t, err, &rateLimitErr)
	assert.Equal(t, 5*time.Second, rateLimitErr.RetryAfter)
}

func TestIsErrRateLimit_Wrapped(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", queue.NewErrRateLimit(5*time.Second))

	assert.True(t, queue.IsErrRateLimit(err))
}

func TestIsErrRateLimit(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "direct ErrRateLimit",
			err:  queue.NewErrRateLimit(time.Second),
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
			got := queue.IsErrRateLimit(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}
