package tests

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
)

// --- Job options snapshot ---

func TestJobOptionsSnapshot(t *testing.T) {
	t.Parallel()

	job := newJob(t, "test", map[string]string{"k": "v"},
		queue.WithQueue("critical"),
		queue.WithMaxRetries(5),
	)
	options := job.Options()
	options.Queue = "mutated"

	assert.Equal(t, "critical", job.Options().Queue)
	assert.Equal(t, 5, job.Options().MaxRetries)
}

// --- NewJob edge cases ---

func TestNewJob_EmptyType(t *testing.T) {
	t.Parallel()

	job, err := queue.NewJob("", nil)
	assert.Nil(t, job)
	assert.ErrorIs(t, err, queue.ErrNoJobTypeSpecified)
}

func TestNewJob_EmptyQueue(t *testing.T) {
	t.Parallel()

	job, err := queue.NewJob("test", nil, queue.WithQueue(""))
	assert.Nil(t, job)
	assert.ErrorIs(t, err, queue.ErrNoJobQueueSpecified)
}

func TestNewJob_SerializationFailure(t *testing.T) {
	t.Parallel()

	job, err := queue.NewJob("test", func() {})
	assert.Nil(t, job)
	assert.ErrorIs(t, err, queue.ErrSerializationFailure)
}

// --- Job options edge cases ---

func TestJobOptions_AllOptions(t *testing.T) {
	t.Parallel()

	now := time.Now()
	deadline := now.Add(time.Hour)
	job := newJob(t, "test", nil,
		queue.WithQueue("q"),
		queue.WithDelay(5*time.Second),
		queue.WithScheduleAt(&now),
		queue.WithMaxRetries(3),
		queue.WithDeadline(&deadline),
		queue.WithRetention(24*time.Hour),
	)
	options := job.Options()

	assert.Equal(t, "q", options.Queue)
	assert.Equal(t, 5*time.Second, options.Delay)
	assert.Equal(t, 3, options.MaxRetries)
	assert.Equal(t, 24*time.Hour, options.Retention)
	assert.True(t, options.ScheduleAt.Equal(now))
	assert.True(t, options.Deadline.Equal(deadline))
}

// --- WriteResult edge cases ---

func TestWriteResult_NoWriter(t *testing.T) {
	t.Parallel()

	var delivery *queue.Delivery
	err := delivery.WriteResult("result")
	assert.ErrorIs(t, err, queue.ErrResultWriterNotSet)
}

// --- ContentDigest stability ---

func TestJobContentDigest_Stable(t *testing.T) {
	t.Parallel()

	j1 := newJob(t, "t", map[string]string{"k": "v"})
	j2 := newJob(t, "t", map[string]string{"k": "v"})
	assert.Equal(t, j1.ContentDigest(), j2.ContentDigest())
	assert.True(t, strings.HasPrefix(j1.ContentDigest(), "q1:sha256:"))
	assert.Len(t, j1.ContentDigest(), len("q1:sha256:")+64)
}

func TestJobContentDigest_IgnoresOptions(t *testing.T) {
	t.Parallel()

	j1 := newJob(t, "t", nil)
	j2 := newJob(t, "t", nil, queue.WithMaxRetries(5))
	assert.Equal(t, j1.ContentDigest(), j2.ContentDigest())
}

func TestDecodePayload_InvalidDestination(t *testing.T) {
	t.Parallel()

	job := newJob(t, "test", map[string]string{"k": "v"})
	err := job.DecodePayload(nil)
	require.Error(t, err)
	assert.False(t, errors.Is(err, queue.ErrSerializationFailure))
}

func TestDecodePayload_SerializationFailure(t *testing.T) {
	t.Parallel()

	job, err := queue.NewJob("test", func() {})
	assert.Nil(t, job)
	assert.ErrorIs(t, err, queue.ErrSerializationFailure)
}
