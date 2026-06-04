package tests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hibiken/asynq"
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

// --- ConvertToAsynqOptions edge cases ---

func TestConvertToAsynqOptions_AllOptions(t *testing.T) {
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
	opts := job.ConvertToAsynqOptions()
	assert.Len(t, opts, 6, "all supported job options should be converted")
}

// --- WriteResult edge cases ---

func TestWriteResult_NoWriter(t *testing.T) {
	t.Parallel()

	var delivery *queue.Delivery
	err := delivery.WriteResult("result")
	assert.ErrorIs(t, err, queue.ErrResultWriterNotSet)
}

func TestWriteResult_WriterFailure(t *testing.T) {
	redisConfig := getRedisConfig()
	worker, err := queue.NewWorker(redisConfig,
		queue.WithWorkerQueue("write_result_failure", 1),
		queue.WithWorkerStopTimeout(100*time.Millisecond),
	)
	require.NoError(t, err)

	client := asynq.NewClient(redisConfig.ToAsynqRedisOpt())
	defer func() { assert.NoError(t, client.Close()) }()

	var once sync.Once
	started := make(chan struct{})
	errorsCh := make(chan error, 1)
	err = worker.Register("write_result_failure", func(ctx context.Context, delivery *queue.Delivery) error {
		once.Do(func() { close(started) })
		<-ctx.Done()
		err := delivery.WriteResult("result")
		errorsCh <- err
		return err
	}, queue.WithJobQueue("write_result_failure"))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- worker.Run(ctx)
	}()
	defer func() {
		cancel()
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(2 * time.Second):
			t.Error("timed out waiting for worker shutdown")
		}
	}()

	task := asynq.NewTask("write_result_failure", []byte(`{}`))
	_, err = client.Enqueue(task,
		asynq.Queue("write_result_failure"),
		asynq.MaxRetry(0),
		asynq.Retention(time.Hour),
		asynq.Timeout(time.Second),
	)
	require.NoError(t, err)

	select {
	case <-started:
		cancel()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for worker to start result writer test")
	}

	select {
	case err := <-errorsCh:
		assert.ErrorIs(t, err, queue.ErrFailedToWriteResult)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for result write failure")
	}
}

// --- ContentDigest stability ---

func TestJobContentDigest_Stable(t *testing.T) {
	t.Parallel()

	j1 := newJob(t, "t", map[string]string{"k": "v"})
	j2 := newJob(t, "t", map[string]string{"k": "v"})
	assert.Equal(t, j1.ContentDigest(), j2.ContentDigest())
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
