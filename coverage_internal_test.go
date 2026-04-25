package queue

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

type recordingLogger struct {
	errors []any
}

func (l *recordingLogger) Debug(...any) {}
func (l *recordingLogger) Info(...any)  {}
func (l *recordingLogger) Warn(...any)  {}
func (l *recordingLogger) Fatal(...any) {}
func (l *recordingLogger) Error(args ...any) {
	l.errors = append(l.errors, args...)
}

type recordingErrorHandler struct {
	err error
	job *Job
}

func (h *recordingErrorHandler) HandleError(err error, job *Job) {
	h.err = err
	h.job = job
}

func TestNewClient_ValidatesConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		redisConfig *RedisConfig
		wantErr     error
	}{
		{
			name:        "nil config",
			redisConfig: nil,
			wantErr:     ErrInvalidRedisConfig,
		},
		{
			name: "invalid config",
			redisConfig: &RedisConfig{
				Network: "tcp",
				Addr:    "",
			},
			wantErr: ErrRedisEmptyAddress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := NewClient(tt.redisConfig)
			assert.Nil(t, client)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestClientEnqueue_ConversionErrorReportsHandler(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	handler := &recordingErrorHandler{}
	client, err := NewClient(DefaultRedisConfig(),
		WithClientLogger(logger),
		WithClientErrorHandler(handler),
		WithClientRetention(time.Hour),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, client.Stop())
	})

	id, err := client.Enqueue("", map[string]string{"k": "v"})

	assert.Empty(t, id)
	assert.ErrorIs(t, err, ErrNoJobTypeSpecified)
	require.NotNil(t, handler.job)
	assert.ErrorIs(t, handler.err, ErrNoJobTypeSpecified)
	assert.Equal(t, "", handler.job.Type)
	assert.NotEmpty(t, logger.errors)
}

func TestClientEnqueueJob_ConversionErrorReportsHandler(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	handler := &recordingErrorHandler{}
	client, err := NewClient(DefaultRedisConfig(),
		WithClientLogger(logger),
		WithClientErrorHandler(handler),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, client.Stop())
	})
	job := NewJob("test", make(chan int))

	id, err := client.EnqueueJob(job)

	assert.Empty(t, id)
	assert.ErrorIs(t, err, ErrSerializationFailure)
	assert.Same(t, job, handler.job)
	assert.ErrorIs(t, handler.err, ErrSerializationFailure)
	assert.NotEmpty(t, logger.errors)
}

func TestWorkerOptionsApplyToConfig(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	handler := &recordingErrorHandler{}
	limiter := rate.NewLimiter(rate.Limit(2), 3)
	queues := map[string]int{"critical": 10, "default": 1}
	config := &WorkerConfig{}

	WithWorkerLogger(logger)(config)
	WithWorkerStopTimeout(5 * time.Second)(config)
	WithWorkerRateLimiter(limiter)(config)
	WithWorkerConcurrency(4)(config)
	WithWorkerQueue("low", 1)(config)
	WithWorkerQueues(queues)(config)
	WithWorkerErrorHandler(handler)(config)

	assert.Same(t, logger, config.Logger)
	assert.Equal(t, 5*time.Second, config.StopTimeout)
	assert.Same(t, limiter, config.Limiter)
	assert.Equal(t, 4, config.Concurrency)
	if diff := cmp.Diff(queues, config.Queues); diff != "" {
		t.Errorf("worker queues mismatch (-want +got):\n%s", diff)
	}
	assert.Same(t, handler, config.ErrorHandler)
}

func TestNewWorker_ValidatesConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		redisConfig *RedisConfig
		options     []WorkerOption
		wantErr     error
	}{
		{
			name:        "nil redis config",
			redisConfig: nil,
			wantErr:     ErrInvalidRedisConfig,
		},
		{
			name: "invalid redis config",
			redisConfig: &RedisConfig{
				Network: "tcp",
				Addr:    "",
			},
			wantErr: ErrRedisEmptyAddress,
		},
		{
			name:        "invalid worker config",
			redisConfig: DefaultRedisConfig(),
			options:     []WorkerOption{WithWorkerConcurrency(0)},
			wantErr:     ErrInvalidWorkerConcurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			worker, err := NewWorker(tt.redisConfig, tt.options...)
			assert.Nil(t, worker)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestSchedulerOptionsApplyToConfig(t *testing.T) {
	t.Parallel()

	provider := NewMemoryConfigProvider()
	logger := &recordingLogger{}
	loc := time.FixedZone("test", 3600)
	pre := func(*Job) {}
	post := func(*JobInfo, error) {}
	options := &SchedulerOptions{}

	WithSyncInterval(5 * time.Second)(options)
	WithSchedulerLocation(loc)(options)
	WithConfigProvider(provider)(options)
	WithSchedulerLogger(logger)(options)
	WithPreEnqueueFunc(pre)(options)
	WithPostEnqueueFunc(post)(options)

	assert.Equal(t, 5*time.Second, options.SyncInterval)
	assert.Same(t, loc, options.Location)
	assert.Same(t, provider, options.ConfigProvider)
	assert.Same(t, logger, options.Logger)
	require.NotNil(t, options.PreEnqueueFunc)
	require.NotNil(t, options.PostEnqueueFunc)
}

func TestNewScheduler_ValidatesConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		redisConfig *RedisConfig
		wantErr     error
	}{
		{
			name:        "nil config",
			redisConfig: nil,
			wantErr:     ErrInvalidRedisConfig,
		},
		{
			name: "invalid config",
			redisConfig: &RedisConfig{
				Network: "tcp",
				Addr:    "",
			},
			wantErr: ErrRedisEmptyAddress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheduler, err := NewScheduler(tt.redisConfig)
			assert.Nil(t, scheduler)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestNewScheduler_AppliesConfigProvider(t *testing.T) {
	t.Parallel()

	provider := NewMemoryConfigProvider()
	scheduler, err := NewScheduler(DefaultRedisConfig(), WithConfigProvider(provider))
	require.NoError(t, err)

	id, err := scheduler.RegisterCron("*/5 * * * *", "cron:test", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	configs, err := provider.GetConfigs()
	require.NoError(t, err)
	assert.Len(t, configs, 1)
}

func TestSchedulerRegisterMethodsUseConfigProvider(t *testing.T) {
	t.Parallel()

	provider := NewMemoryConfigProvider()
	scheduler, err := NewScheduler(DefaultRedisConfig(), WithConfigProvider(provider))
	require.NoError(t, err)

	_, err = scheduler.RegisterCron("wrong cron", "bad", nil)
	assert.ErrorIs(t, err, ErrInvalidCronSpec)

	cronID, err := scheduler.RegisterCron("*/5 * * * *", "cron:test", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, cronID)

	periodicID, err := scheduler.RegisterPeriodic(2*time.Second, "periodic:test", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, periodicID)

	configs, err := provider.GetConfigs()
	require.NoError(t, err)
	assert.Len(t, configs, 2)

	require.NoError(t, scheduler.UnregisterCronJob(cronID))
	assert.ErrorIs(t, scheduler.UnregisterCronJob(cronID), ErrConfigJobNotFound)
}

func TestRedisConfigValidateBoundaries(t *testing.T) {
	t.Parallel()

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	tests := []struct {
		name    string
		config  RedisConfig
		wantErr error
	}{
		{
			name: "valid tcp address",
			config: RedisConfig{
				Network: "tcp",
				Addr:    "localhost:6379",
			},
		},
		{
			name: "empty address",
			config: RedisConfig{
				Network: "tcp",
				Addr:    "",
			},
			wantErr: ErrRedisEmptyAddress,
		},
		{
			name: "unsupported network",
			config: RedisConfig{
				Network: "udp",
				Addr:    "localhost:6379",
			},
			wantErr: ErrRedisUnsupportedNetwork,
		},
		{
			name: "rediss requires tls",
			config: RedisConfig{
				Network: "tcp",
				Addr:    "rediss://localhost:6379",
			},
			wantErr: ErrRedisTLSRequired,
		},
		{
			name: "rediss still requires host port syntax",
			config: RedisConfig{
				Network:   "tcp",
				Addr:      "rediss://localhost:6379",
				TLSConfig: tlsConfig,
			},
			wantErr: ErrRedisInvalidAddress,
		},
		{
			name: "unix socket skips host port validation",
			config: RedisConfig{
				Network: "unix",
				Addr:    "/tmp/redis.sock",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestRedisConfigOptionsAndAsynqConversion(t *testing.T) {
	t.Parallel()

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	config := NewRedisConfig(
		WithRedisAddress("redis.example:6380"),
		WithRedisUsername("user"),
		WithRedisPassword("secret"),
		WithRedisDB(3),
		WithRedisDialTimeout(2*time.Second),
		WithRedisReadTimeout(3*time.Second),
		WithRedisWriteTimeout(4*time.Second),
		WithRedisPoolSize(12),
		WithRedisTLSConfig(tlsConfig),
	)
	got := config.ToAsynqRedisOpt()
	type redisOptSnapshot struct {
		Network      string
		Addr         string
		Username     string
		Password     string
		DB           int
		DialTimeout  time.Duration
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		PoolSize     int
	}
	want := redisOptSnapshot{
		Network:      "tcp",
		Addr:         "redis.example:6380",
		Username:     "user",
		Password:     "secret",
		DB:           3,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 4 * time.Second,
		PoolSize:     12,
	}
	gotSnapshot := redisOptSnapshot{
		Network:      got.Network,
		Addr:         got.Addr,
		Username:     got.Username,
		Password:     got.Password,
		DB:           got.DB,
		DialTimeout:  got.DialTimeout,
		ReadTimeout:  got.ReadTimeout,
		WriteTimeout: got.WriteTimeout,
		PoolSize:     got.PoolSize,
	}
	if diff := cmp.Diff(want, gotSnapshot); diff != "" {
		t.Errorf("redis option mismatch (-want +got):\n%s", diff)
	}
	assert.Same(t, tlsConfig, got.TLSConfig)
}

func TestJobOptionsAndSetters(t *testing.T) {
	t.Parallel()

	scheduleAt := time.Now().Add(time.Hour)
	deadline := scheduleAt.Add(time.Hour)
	job := NewJob("job:test", nil)

	got := job.SetID("job-id").SetResultWriter(nil)
	job.WithOptions(
		WithDelay(time.Second),
		WithMaxRetries(3),
		WithQueue("critical"),
		WithScheduleAt(&scheduleAt),
		WithDeadline(&deadline),
		WithRetention(time.Hour),
	)

	assert.Same(t, job, got)
	assert.Equal(t, "job-id", job.ID)
	assert.Equal(t, time.Second, job.Options.Delay)
	assert.Equal(t, 3, job.Options.MaxRetries)
	assert.Equal(t, "critical", job.Options.Queue)
	assert.Same(t, &scheduleAt, job.Options.ScheduleAt)
	assert.Same(t, &deadline, job.Options.Deadline)
	assert.Equal(t, time.Hour, job.Options.Retention)
}

func TestConvertToAsynqTask_ValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		job     *Job
		wantErr error
	}{
		{
			name:    "missing type",
			job:     NewJob("", nil),
			wantErr: ErrNoJobTypeSpecified,
		},
		{
			name:    "missing queue",
			job:     NewJob("job:test", nil, WithQueue("")),
			wantErr: ErrNoJobQueueSpecified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			task, opts, err := tt.job.ConvertToAsynqTask()
			assert.Nil(t, task)
			assert.Nil(t, opts)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestConvertToAsynqOptions_AllOptions(t *testing.T) {
	t.Parallel()

	scheduleAt := time.Now().Add(time.Hour)
	deadline := scheduleAt.Add(time.Hour)
	job := NewJob("job:test", nil,
		WithQueue("critical"),
		WithDelay(time.Second),
		WithScheduleAt(&scheduleAt),
		WithMaxRetries(3),
		WithDeadline(&deadline),
		WithRetention(time.Hour),
	)

	opts := job.ConvertToAsynqOptions()

	assert.Len(t, opts, 6)
}

func TestHandlerOptionsAndMiddleware(t *testing.T) {
	t.Parallel()

	limiter := rate.NewLimiter(rate.Limit(1), 1)
	calls := make([]string, 0, 3)
	middleware := func(name string) MiddlewareFunc {
		return func(next HandlerFunc) HandlerFunc {
			return func(ctx context.Context, job *Job) error {
				calls = append(calls, name)
				return next(ctx, job)
			}
		}
	}
	retryDelay := func(count int, _ error) time.Duration {
		return time.Duration(count) * time.Second
	}
	handler := NewHandler("job:test",
		func(context.Context, *Job) error {
			calls = append(calls, "handler")
			return nil
		},
		WithRateLimiter(limiter),
		WithRetryDelayFunc(retryDelay),
		WithMiddleware(middleware("first")),
	)

	handler.WithOptions(WithJobQueue("critical"), WithJobTimeout(5*time.Second))
	handler.Use(middleware("second"))
	err := handler.Process(t.Context(), NewJob("job:test", nil))

	require.NoError(t, err)
	assert.Equal(t, "critical", handler.JobQueue)
	assert.Equal(t, 5*time.Second, handler.JobTimeout)
	assert.Same(t, limiter, handler.Limiter)
	assert.Equal(t, 2*time.Second, handler.RetryDelayFunc(2, assert.AnError))
	want := []string{"first", "second", "handler"}
	if diff := cmp.Diff(want, calls); diff != "" {
		t.Errorf("middleware calls mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertToAsynqTask_SerializationError(t *testing.T) {
	t.Parallel()

	payload := make(chan int)
	job := NewJob("test", payload)
	_, _, err := job.ConvertToAsynqTask()
	assert.ErrorIs(t, err, ErrSerializationFailure)
}

func TestDecodePayload_SerializationError(t *testing.T) {
	t.Parallel()

	payload := make(chan int)
	job := &Job{Type: "test", Payload: payload}
	var out string
	err := job.DecodePayload(&out)
	assert.ErrorIs(t, err, ErrSerializationFailure)
}

func TestProcessWithTimeout_ContextCanceled(t *testing.T) {
	t.Parallel()

	handler := NewHandler("test", func(ctx context.Context,
		_ *Job) error {
		<-ctx.Done()
		return ctx.Err()
	}, WithJobTimeout(100*time.Millisecond))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	job := &Job{Type: "test"}
	err := handler.Process(ctx, job)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestProcessWithTimeout_Success(t *testing.T) {
	t.Parallel()

	handler := NewHandler("test", func(_ context.Context,
		_ *Job) error {
		return nil
	}, WithJobTimeout(5*time.Second))

	job := &Job{Type: "test"}
	err := handler.Process(t.Context(), job)
	require.NoError(t, err)
}

func TestWriteResult_SerializationError(t *testing.T) {
	t.Parallel()

	job := NewJob("test", nil)
	err := job.WriteResult("data")
	assert.ErrorIs(t, err, ErrResultWriterNotSet)
}

func TestManagerListJobsByStateRejectsInvalidState(t *testing.T) {
	t.Parallel()

	manager := &Manager{}

	jobs, err := manager.ListJobsByState("default", JobState("unknown"), 10, 1)

	assert.Nil(t, jobs)
	assert.ErrorIs(t, err, ErrInvalidJobState)
}

func TestManagerRunJobsByStateEarlyErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   JobState
		wantErr error
	}{
		{name: "invalid state", state: JobState("unknown"), wantErr: ErrInvalidJobState},
		{name: "aggregating requires group", state: StateAggregating, wantErr: ErrGroupRequiredForAggregation},
		{name: "active not supported", state: StateActive, wantErr: ErrOperationNotSupported},
		{name: "pending not supported", state: StatePending, wantErr: ErrOperationNotSupported},
		{name: "completed not supported", state: StateCompleted, wantErr: ErrOperationNotSupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			count, err := (&Manager{}).RunJobsByState("default", tt.state)
			assert.Zero(t, count)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestManagerArchiveJobsByStateEarlyErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   JobState
		wantErr error
	}{
		{name: "invalid state", state: JobState("unknown"), wantErr: ErrInvalidJobState},
		{name: "archived not supported", state: StateArchived, wantErr: ErrOperationNotSupported},
		{name: "completed not supported", state: StateCompleted, wantErr: ErrOperationNotSupported},
		{name: "active requires cancel", state: StateActive, wantErr: ErrArchivingActiveJobs},
		{name: "aggregating requires group", state: StateAggregating, wantErr: ErrGroupRequiredForAggregation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			count, err := (&Manager{}).ArchiveJobsByState("default", tt.state)
			assert.Zero(t, count)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestManagerDeleteJobsByStateEarlyErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   JobState
		wantErr error
	}{
		{name: "invalid state", state: JobState("unknown"), wantErr: ErrInvalidJobState},
		{name: "active not supported", state: StateActive, wantErr: ErrOperationNotSupported},
		{name: "aggregating not supported", state: StateAggregating, wantErr: ErrOperationNotSupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			count, err := (&Manager{}).DeleteJobsByState("default", tt.state)
			assert.Zero(t, count)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestRedisInfo_UnsupportedClient(t *testing.T) {
	t.Parallel()

	m := &Manager{client: nil}
	_, err := m.RedisInfo(t.Context())
	assert.ErrorIs(t, err, ErrRedisClientNotSupported)
}

func TestRedisValidate_UnixNetwork(t *testing.T) {
	t.Parallel()

	cfg := &RedisConfig{
		Network: "unix",
		Addr:    "/tmp/redis.sock",
	}
	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestNewJobFromAsynqTask(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"key":"value"}`)
	task := asynq.NewTask("test:type", payload)
	job, err := NewJobFromAsynqTask(task)
	require.NoError(t, err)
	assert.Equal(t, "test:type", job.Type)
	if diff := cmp.Diff(payload, job.Payload); diff != "" {
		t.Errorf("job payload mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertToAsynqOptions_NoOptions(t *testing.T) {
	t.Parallel()

	job := &Job{Options: JobOptions{}}
	opts := job.ConvertToAsynqOptions()
	assert.Empty(t, opts)
}

func TestConvertToAsynqOptions_ZeroScheduleAt(t *testing.T) {
	t.Parallel()

	zero := time.Time{}
	job := &Job{Options: JobOptions{ScheduleAt: &zero}}
	opts := job.ConvertToAsynqOptions()
	assert.Empty(t, opts)
}

func TestConvertToAsynqOptions_ZeroDeadline(t *testing.T) {
	t.Parallel()

	zero := time.Time{}
	job := &Job{Options: JobOptions{Deadline: &zero}}
	opts := job.ConvertToAsynqOptions()
	assert.Empty(t, opts)
}

func TestEnqueueJob_WithJobRetention(t *testing.T) {
	t.Parallel()

	job := NewJob("test", map[string]string{"k": "v"},
		WithRetention(2*time.Hour),
	)
	assert.Equal(t, 2*time.Hour, job.Options.Retention)
}

func TestRedisInfo_StandardClient(t *testing.T) {
	t.Parallel()

	m := &Manager{client: nil}
	_, err := m.RedisInfo(t.Context())
	assert.ErrorIs(t, err, ErrRedisClientNotSupported)
}

func TestMemoryConfigProvider_RegisterDuplicate(t *testing.T) {
	t.Parallel()

	p := NewMemoryConfigProvider()
	job := NewJob("test", nil)
	_, err := p.RegisterCronJob("* * * * *", job)
	require.NoError(t, err)

	id, err := p.RegisterCronJob("* * * * *", job)

	assert.Empty(t, id)
	assert.ErrorIs(t, err, ErrJobAlreadyExists)
}

func TestGetConfigs_WithEntries(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	p := NewMemoryConfigProvider()
	_, err := p.RegisterCronJob("* * * * *", NewJob("", nil))
	require.NoError(t, err)

	_, err = p.GetConfigs()
	assert.ErrorIs(t, err, ErrNoJobTypeSpecified)
}

func TestGetConfigs_SerializationError(t *testing.T) {
	t.Parallel()

	p := NewMemoryConfigProvider()
	_, err := p.RegisterCronJob("* * * * *", NewJob("test", make(chan int)))
	require.NoError(t, err)

	_, err = p.GetConfigs()
	assert.ErrorIs(t, err, ErrSerializationFailure)
}
