package tests

import (
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
)

// --- Job.WithOptions ---

func TestJobWithOptions(t *testing.T) {
	job := queue.NewJob("test", map[string]string{"k": "v"})
	assert.Equal(t, queue.DefaultQueue, job.Options.Queue)

	job.WithOptions(
		queue.WithQueue("critical"),
		queue.WithMaxRetries(5),
	)
	assert.Equal(t, "critical", job.Options.Queue)
	assert.Equal(t, 5, job.Options.MaxRetries)
}

// --- ConvertToAsynqTask edge cases ---

func TestConvertToAsynqTask_EmptyType(t *testing.T) {
	job := queue.NewJob("", nil)
	_, _, err := job.ConvertToAsynqTask()
	assert.ErrorIs(t, err, queue.ErrNoJobTypeSpecified)
}

func TestConvertToAsynqTask_EmptyQueue(t *testing.T) {
	job := queue.NewJob("test", nil, queue.WithQueue(""))
	_, _, err := job.ConvertToAsynqTask()
	assert.ErrorIs(t, err, queue.ErrNoJobQueueSpecified)
}

// --- ConvertToAsynqOptions edge cases ---

func TestConvertToAsynqOptions_AllOptions(t *testing.T) {
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
	// Queue + Delay + ScheduleAt + MaxRetries + Deadline + Retention = 6
	assert.Len(t, opts, 6)
}

// --- WriteResult edge cases ---

func TestWriteResult_NoWriter(t *testing.T) {
	job := queue.NewJob("test", nil)
	err := job.WriteResult("result")
	assert.ErrorIs(t, err, queue.ErrResultWriterNotSet)
}

// --- Fingerprint stability ---

func TestJobFingerprint_Stable(t *testing.T) {
	j1 := queue.NewJob("t", map[string]string{"k": "v"})
	j2 := queue.NewJob("t", map[string]string{"k": "v"})
	assert.Equal(t, j1.Fingerprint, j2.Fingerprint)
}

func TestJobFingerprint_DiffersWithOptions(t *testing.T) {
	j1 := queue.NewJob("t", nil)
	j2 := queue.NewJob("t", nil, queue.WithMaxRetries(5))
	assert.NotEqual(t, j1.Fingerprint, j2.Fingerprint)
}
