package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
)

// --- Job.WithOptions ---

func TestJobWithOptions(t *testing.T) {
	t.Parallel()

	job := queue.NewJob("test", map[string]string{"k": "v"})
	originalFingerprint := job.Fingerprint
	assert.Equal(t, queue.DefaultQueue, job.Options.Queue)

	job.WithOptions(
		queue.WithQueue("critical"),
		queue.WithMaxRetries(5),
	)
	assert.Equal(t, "critical", job.Options.Queue)
	assert.Equal(t, 5, job.Options.MaxRetries)
	assert.Equal(t, originalFingerprint, job.Fingerprint)
}

// --- ConvertToAsynqTask edge cases ---

func TestConvertToAsynqTask_EmptyType(t *testing.T) {
	t.Parallel()

	job := queue.NewJob("", nil)
	_, _, err := job.ConvertToAsynqTask()
	assert.ErrorIs(t, err, queue.ErrNoJobTypeSpecified)
}

func TestConvertToAsynqTask_EmptyQueue(t *testing.T) {
	t.Parallel()

	job := queue.NewJob("test", nil, queue.WithQueue(""))
	_, _, err := job.ConvertToAsynqTask()
	assert.ErrorIs(t, err, queue.ErrNoJobQueueSpecified)
}

func TestConvertToAsynqTask_SerializationFailure(t *testing.T) {
	t.Parallel()

	job := queue.NewJob("test", func() {})
	_, _, err := job.ConvertToAsynqTask()
	assert.ErrorIs(t, err, queue.ErrSerializationFailure)
}

// --- ConvertToAsynqOptions edge cases ---

func TestConvertToAsynqOptions_AllOptions(t *testing.T) {
	t.Parallel()

	now := time.Now()
	deadline := now.Add(time.Hour)
	job := queue.NewJob("test", nil,
		queue.WithQueue("q"),
		queue.WithDelay(5*time.Second),
		queue.WithScheduleAt(&now),
		queue.WithMaxRetries(3),
		queue.WithDeadline(&deadline),
		queue.WithRetention(24*time.Hour),
	)
	opts := job.ConvertToAsynqOptions()
	assert.Len(t, opts, 6, "all supported job options should be converted")
}

// --- WriteResult edge cases ---

func TestWriteResult_NoWriter(t *testing.T) {
	t.Parallel()

	job := queue.NewJob("test", nil)
	err := job.WriteResult("result")
	assert.ErrorIs(t, err, queue.ErrResultWriterNotSet)
}

// --- Fingerprint stability ---

func TestJobFingerprint_Stable(t *testing.T) {
	t.Parallel()

	j1 := queue.NewJob("t", map[string]string{"k": "v"})
	j2 := queue.NewJob("t", map[string]string{"k": "v"})
	assert.Equal(t, j1.Fingerprint, j2.Fingerprint)
}

func TestJobFingerprint_DiffersWithOptions(t *testing.T) {
	t.Parallel()

	j1 := queue.NewJob("t", nil)
	j2 := queue.NewJob("t", nil, queue.WithMaxRetries(5))
	assert.NotEqual(t, j1.Fingerprint, j2.Fingerprint)
}

func TestDecodePayload_InvalidDestination(t *testing.T) {
	t.Parallel()

	job := queue.NewJob("test", map[string]string{"k": "v"})
	err := job.DecodePayload(nil)
	require.Error(t, err)
	assert.False(t, errors.Is(err, queue.ErrSerializationFailure))
}

func TestDecodePayload_SerializationFailure(t *testing.T) {
	t.Parallel()

	job := queue.NewJob("test", func() {})
	err := job.DecodePayload(new(map[string]string))
	assert.ErrorIs(t, err, queue.ErrSerializationFailure)
}
