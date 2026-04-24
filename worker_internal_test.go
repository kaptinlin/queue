package queue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  WorkerConfig
		wantErr error
	}{
		{
			name:    "valid config",
			config:  WorkerConfig{Concurrency: 4, Queues: map[string]int{"q": 1}},
			wantErr: nil,
		},
		{
			name:    "zero concurrency",
			config:  WorkerConfig{Concurrency: 0, Queues: map[string]int{"q": 1}},
			wantErr: ErrInvalidWorkerConcurrency,
		},
		{
			name:    "negative concurrency",
			config:  WorkerConfig{Concurrency: -1, Queues: map[string]int{"q": 1}},
			wantErr: ErrInvalidWorkerConcurrency,
		},
		{
			name:    "empty queues",
			config:  WorkerConfig{Concurrency: 1, Queues: map[string]int{}},
			wantErr: ErrInvalidWorkerQueues,
		},
		{
			name:    "nil queues",
			config:  WorkerConfig{Concurrency: 1},
			wantErr: ErrInvalidWorkerQueues,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
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
	assert.False(t, w.isFailure(&ErrRateLimit{RetryAfter: time.Second}))
	assert.False(t, w.isFailure(ErrTransientIssue))
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
