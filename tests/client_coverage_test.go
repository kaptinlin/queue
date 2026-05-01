package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
)

// --- WithClientLogger ---

func TestWithClientLogger(t *testing.T) {
	logger := &mockLogger{}
	client, err := queue.NewClient(getRedisConfig(),
		queue.WithClientLogger(logger),
	)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NoError(t, client.Stop())
}

// --- NewClient validation ---

func TestNewClient_NilRedisConfig(t *testing.T) {
	_, err := queue.NewClient(nil)
	assert.ErrorIs(t, err, queue.ErrInvalidRedisConfig)
}

func TestNewClient_InvalidRedisConfig(t *testing.T) {
	cfg := &queue.RedisConfig{Network: "bad", Addr: ""}
	_, err := queue.NewClient(cfg)
	assert.Error(t, err)
}

// --- EnqueueJob with client retention ---

func TestClientEnqueueJob_WithClientRetention(t *testing.T) {
	client, err := queue.NewClient(getRedisConfig(),
		queue.WithClientRetention(1),
	)
	require.NoError(t, err)
	defer func() { assert.NoError(t, client.Stop()) }()

	id, err := client.Enqueue("retention_test",
		map[string]string{"k": "v"})
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestClientEnqueueJob_ReportsRedisFailure(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	handler := &recordingClientErrorHandler{}
	client, err := queue.NewClient(queue.NewRedisConfig(
		queue.WithRedisAddress("127.0.0.1:1"),
		queue.WithRedisDialTimeout(10*time.Millisecond),
		queue.WithRedisReadTimeout(10*time.Millisecond),
		queue.WithRedisWriteTimeout(10*time.Millisecond),
	),
		queue.WithClientLogger(logger),
		queue.WithClientErrorHandler(handler),
	)
	require.NoError(t, err)
	defer func() { assert.NoError(t, client.Stop()) }()

	job := queue.NewJob("enqueue_failure_test", map[string]string{"k": "v"})
	id, err := client.EnqueueJob(job)

	assert.Empty(t, id)
	assert.ErrorIs(t, err, queue.ErrEnqueueJob)
	assert.True(t, logger.hasLevel("error"))
	require.Len(t, handler.errors, 1)
	assert.NotErrorIs(t, handler.errors[0], queue.ErrEnqueueJob)
	assert.Same(t, job, handler.jobs[0])
}

type recordingClientErrorHandler struct {
	errors []error
	jobs   []*queue.Job
}

func (h *recordingClientErrorHandler) HandleError(err error, job *queue.Job) {
	h.errors = append(h.errors, err)
	h.jobs = append(h.jobs, job)
}
