package tests

import (
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
)

func TestNewSkipRetryError(t *testing.T) {
	err := queue.NewSkipRetryError("invalid payload")

	assert.Error(t, err)
	assert.ErrorIs(t, err, queue.ErrSkipRetry)
	assert.Contains(t, err.Error(), "invalid payload")
}

func TestErrRateLimitError(t *testing.T) {
	err := queue.NewErrRateLimit(5 * time.Second)

	assert.Equal(t, 5*time.Second, err.RetryAfter)
	assert.Contains(t, err.Error(), "rate limited")
	assert.Contains(t, err.Error(), "5s")
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
