package tests

import (
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- WithWorkerLogger ---

func TestWithWorkerLogger(t *testing.T) {
	logger := &mockLogger{}
	worker, err := queue.NewWorker(getRedisConfig(),
		queue.WithWorkerLogger(logger),
	)
	require.NoError(t, err)
	assert.NotNil(t, worker)
}

// --- WithWorkerStopTimeout ---

func TestWithWorkerStopTimeout(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig(),
		queue.WithWorkerStopTimeout(30*time.Second),
	)
	require.NoError(t, err)
	assert.NotNil(t, worker)
}

// --- WithWorkerQueue ---

func TestWithWorkerQueue(t *testing.T) {
	worker, err := queue.NewWorker(getRedisConfig(),
		queue.WithWorkerQueue("critical", 10),
		queue.WithWorkerQueue("low", 1),
	)
	require.NoError(t, err)
	assert.NotNil(t, worker)
}

// --- WithWorkerQueues ---

func TestWithWorkerQueues(t *testing.T) {
	queues := map[string]int{
		"critical": 10,
		"default":  5,
		"low":      1,
	}
	worker, err := queue.NewWorker(getRedisConfig(),
		queue.WithWorkerQueues(queues),
	)
	require.NoError(t, err)
	assert.NotNil(t, worker)
}

func TestWithWorkerQueues_Nil(t *testing.T) {
	// Passing nil should use default queues.
	worker, err := queue.NewWorker(getRedisConfig(),
		queue.WithWorkerQueues(nil),
	)
	require.NoError(t, err)
	assert.NotNil(t, worker)
}
