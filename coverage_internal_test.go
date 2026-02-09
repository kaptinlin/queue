package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFingerprint_AlreadySet(t *testing.T) {
	job := &Job{
		Type:        "test",
		Fingerprint: "pre-set-fingerprint",
	}
	job.fingerprint()
	assert.Equal(t, "pre-set-fingerprint", job.Fingerprint)
}

func TestConvertToAsynqTask_SerializationError(t *testing.T) {
	job := NewJob("test", make(chan int))
	_, _, err := job.ConvertToAsynqTask()
	assert.ErrorIs(t, err, ErrSerializationFailure)
}

func TestDecodePayload_SerializationError(t *testing.T) {
	job := &Job{Type: "test", Payload: make(chan int)}
	var out string
	err := job.DecodePayload(&out)
	assert.ErrorIs(t, err, ErrSerializationFailure)
}

func TestProcessWithTimeout_ContextCanceled(t *testing.T) {
	handler := NewHandler("test", func(ctx context.Context,
		_ *Job) error {
		<-ctx.Done()
		return ctx.Err()
	}, WithJobTimeout(100*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	job := &Job{Type: "test"}
	err := handler.Process(ctx, job)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestProcessWithTimeout_Success(t *testing.T) {
	handler := NewHandler("test", func(_ context.Context,
		_ *Job) error {
		return nil
	}, WithJobTimeout(5*time.Second))

	job := &Job{Type: "test"}
	err := handler.Process(context.Background(), job)
	require.NoError(t, err)
}

func TestHandleQueueError_QueueNotFound(t *testing.T) {
	m := &Manager{}
	//nolint:err113
	err := m.handleQueueError(
		errors.New("queue does not exist"))
	assert.ErrorIs(t, err, ErrQueueNotFound)
}

func TestHandleQueueError_OtherError(t *testing.T) {
	m := &Manager{}
	//nolint:err113
	orig := errors.New("some other error")
	err := m.handleQueueError(orig)
	assert.Equal(t, orig, err)
}

func TestWriteResult_SerializationError(t *testing.T) {
	job := &Job{Type: "test", resultWriter: nil}
	err := job.WriteResult("data")
	assert.ErrorIs(t, err, ErrResultWriterNotSet)
}

func TestGetRedisInfo_UnsupportedClient(t *testing.T) {
	m := &Manager{client: nil}
	_, err := m.GetRedisInfo(context.Background())
	assert.ErrorIs(t, err, ErrRedisClientNotSupported)
}

func TestRedisValidate_UnixNetwork(t *testing.T) {
	cfg := &RedisConfig{
		Network: "unix",
		Addr:    "/tmp/redis.sock",
	}
	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestNewJobFromAsynqTask(t *testing.T) {
	task := asynq.NewTask("test:type",
		[]byte(`{"key":"value"}`))
	job, err := NewJobFromAsynqTask(task)
	require.NoError(t, err)
	assert.Equal(t, "test:type", job.Type)
}

func TestConvertToAsynqOptions_NoOptions(t *testing.T) {
	job := &Job{Options: JobOptions{}}
	opts := job.ConvertToAsynqOptions()
	assert.Empty(t, opts)
}

func TestConvertToAsynqOptions_ZeroScheduleAt(t *testing.T) {
	zero := time.Time{}
	job := &Job{Options: JobOptions{ScheduleAt: &zero}}
	opts := job.ConvertToAsynqOptions()
	assert.Empty(t, opts)
}

func TestConvertToAsynqOptions_ZeroDeadline(t *testing.T) {
	zero := time.Time{}
	job := &Job{Options: JobOptions{Deadline: &zero}}
	opts := job.ConvertToAsynqOptions()
	assert.Empty(t, opts)
}

func TestEnqueueJob_WithJobRetention(t *testing.T) {
	job := NewJob("test", map[string]string{"k": "v"},
		WithRetention(2*time.Hour),
	)
	assert.Equal(t, 2*time.Hour, job.Options.Retention)
}

func TestRetryDelayFunc_RateLimitError(t *testing.T) {
	w := &Worker{
		handlers: make(map[string]*Handler),
	}
	task := asynq.NewTask("test", nil)
	rlErr := &ErrRateLimit{RetryAfter: 5 * time.Second}
	d := w.retryDelayFunc(1, rlErr, task)
	assert.Equal(t, 5*time.Second, d)
}

func TestRetryDelayFunc_CustomHandler(t *testing.T) {
	w := &Worker{
		handlers: make(map[string]*Handler),
	}
	w.handlers["test"] = &Handler{
		RetryDelayFunc: func(n int, _ error) time.Duration {
			return time.Duration(n) * time.Second
		},
	}
	task := asynq.NewTask("test", nil)
	//nolint:err113
	d := w.retryDelayFunc(3, errors.New("fail"), task)
	assert.Equal(t, 3*time.Second, d)
}

func TestRetryDelayFunc_DefaultFallback(t *testing.T) {
	w := &Worker{
		handlers: make(map[string]*Handler),
	}
	task := asynq.NewTask("test", nil)
	//nolint:err113
	d := w.retryDelayFunc(1, errors.New("fail"), task)
	assert.Greater(t, d, time.Duration(0))
}

func TestGetRedisInfo_StandardClient(t *testing.T) {
	m := &Manager{client: nil}
	_, err := m.GetRedisInfo(context.Background())
	assert.ErrorIs(t, err, ErrRedisClientNotSupported)
}

func TestGetConfigs_WithEntries(t *testing.T) {
	p := NewMemoryConfigProvider()
	j := NewJob("test", nil)
	_, err := p.RegisterCronJob("* * * * *", j)
	require.NoError(t, err)

	configs, err := p.GetConfigs()
	require.NoError(t, err)
	assert.Len(t, configs, 1)
	assert.Equal(t, "* * * * *", configs[0].Cronspec)
	assert.Equal(t, "test", configs[0].Task.Type())
}

func TestGetConfigs_ConversionError(t *testing.T) {
	p := NewMemoryConfigProvider()
	p.mu.Lock()
	p.jobs["bad"] = JobConfig{
		Schedule: "* * * * *",
		Job:      NewJob("", nil),
	}
	p.mu.Unlock()

	_, err := p.GetConfigs()
	assert.Error(t, err)
}
