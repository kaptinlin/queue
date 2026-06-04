package queue

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
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

type recordingWorkerErrorHandler struct {
	err      error
	delivery *Delivery
}

func (h *recordingWorkerErrorHandler) HandleError(err error, delivery *Delivery) {
	h.err = err
	h.delivery = delivery
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
		assert.NoError(t, client.Close())
	})

	id, err := client.Enqueue("", map[string]string{"k": "v"})

	assert.Empty(t, id)
	assert.ErrorIs(t, err, ErrNoJobTypeSpecified)
	assert.Nil(t, handler.job)
	assert.ErrorIs(t, handler.err, ErrNoJobTypeSpecified)
	assert.NotEmpty(t, logger.errors)
}

func TestNewClient_NilLoggerUsesDefault(t *testing.T) {
	t.Parallel()

	client, err := NewClient(DefaultRedisConfig(), WithClientLogger(nil))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})

	id, err := client.Enqueue("", nil)
	assert.Empty(t, id)
	assert.ErrorIs(t, err, ErrNoJobTypeSpecified)
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
		assert.NoError(t, client.Close())
	})
	id, err := client.Enqueue("test", make(chan int))

	assert.Empty(t, id)
	assert.ErrorIs(t, err, ErrSerializationFailure)
	assert.Nil(t, handler.job)
	assert.ErrorIs(t, handler.err, ErrSerializationFailure)
	assert.NotEmpty(t, logger.errors)
}

func TestWorkerOptionsApplyToConfig(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	handler := &recordingWorkerErrorHandler{}
	limiter := rate.NewLimiter(rate.Limit(2), 3)
	queues := map[string]int{"critical": 10, "default": 1}
	wantQueues := map[string]int{"critical": 10, "default": 1}
	config := &workerConfig{}

	WithWorkerLogger(logger).applyWorkerOption(config)
	WithWorkerStopTimeout(5 * time.Second).applyWorkerOption(config)
	WithWorkerRateLimiter(limiter).applyWorkerOption(config)
	WithWorkerConcurrency(4).applyWorkerOption(config)
	WithWorkerQueue("low", 1).applyWorkerOption(config)
	WithWorkerQueues(queues).applyWorkerOption(config)
	WithWorkerErrorHandler(handler).applyWorkerOption(config)
	queues["critical"] = 99

	assert.Same(t, logger, config.Logger)
	assert.Equal(t, 5*time.Second, config.StopTimeout)
	assert.Same(t, limiter, config.Limiter)
	assert.Equal(t, 4, config.Concurrency)
	if diff := cmp.Diff(wantQueues, config.Queues); diff != "" {
		t.Errorf("worker queues mismatch (-want +got):\n%s", diff)
	}
	assert.Same(t, handler, config.ErrorHandler)
}

func TestWorkerRunCanceledContext(t *testing.T) {
	t.Parallel()

	worker, err := NewWorker(DefaultRedisConfig())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = worker.Run(ctx)
	assert.ErrorIs(t, err, context.Canceled)
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

func TestNewWorker_NilLoggerUsesDefault(t *testing.T) {
	t.Parallel()

	worker, err := NewWorker(DefaultRedisConfig(), WithWorkerLogger(nil))
	require.NoError(t, err)
	assert.NotNil(t, worker)
}

func TestSchedulerOptionsApplyToConfig(t *testing.T) {
	t.Parallel()

	provider := NewMemoryConfigProvider()
	logger := &recordingLogger{}
	loc := time.FixedZone("test", 3600)
	options := &schedulerOptions{}

	WithSyncInterval(5 * time.Second).applySchedulerOption(options)
	WithSchedulerLocation(loc).applySchedulerOption(options)
	WithConfigProvider(provider).applySchedulerOption(options)
	WithSchedulerLogger(logger).applySchedulerOption(options)

	assert.Equal(t, 5*time.Second, options.SyncInterval)
	assert.Same(t, loc, options.Location)
	assert.Same(t, provider, options.ConfigProvider)
	assert.Same(t, logger, options.Logger)
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

func TestNewScheduler_NilLoggerUsesDefault(t *testing.T) {
	t.Parallel()

	scheduler, err := NewScheduler(DefaultRedisConfig(), WithSchedulerLogger(nil))
	require.NoError(t, err)
	assert.NotNil(t, scheduler)
}

func TestNewScheduler_InvalidSyncInterval(t *testing.T) {
	t.Parallel()

	scheduler, err := NewScheduler(DefaultRedisConfig(), WithSyncInterval(0))

	assert.Nil(t, scheduler)
	assert.ErrorIs(t, err, ErrInvalidSyncInterval)
}

func TestNewScheduler_AppliesConfigProvider(t *testing.T) {
	t.Parallel()

	provider := NewMemoryConfigProvider()
	scheduler, err := NewScheduler(DefaultRedisConfig(), WithConfigProvider(provider))
	require.NoError(t, err)

	id, err := scheduler.RegisterCron("cron:test", "*/5 * * * *", "cron:test", nil)
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

	_, err = scheduler.RegisterCron("bad", "wrong cron", "bad", nil)
	assert.ErrorIs(t, err, ErrInvalidCronSpec)

	cronID, err := scheduler.RegisterCron("cron:test", "*/5 * * * *", "cron:test", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, cronID)

	periodicID, err := scheduler.RegisterPeriodic("periodic:test", 2*time.Second, "periodic:test", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, periodicID)

	configs, err := provider.GetConfigs()
	require.NoError(t, err)
	assert.Len(t, configs, 2)

	require.NoError(t, scheduler.UnregisterCronJob(cronID))
	assert.ErrorIs(t, scheduler.UnregisterCronJob(cronID), ErrScheduleNotFound)
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
		{
			name: "negative db",
			config: RedisConfig{
				Network: "tcp",
				Addr:    "localhost:6379",
				DB:      -1,
			},
			wantErr: ErrRedisInvalidDB,
		},
		{
			name: "negative pool size",
			config: RedisConfig{
				Network:  "tcp",
				Addr:     "localhost:6379",
				PoolSize: -1,
			},
			wantErr: ErrRedisInvalidPoolSize,
		},
		{
			name: "negative dial timeout",
			config: RedisConfig{
				Network:     "tcp",
				Addr:        "localhost:6379",
				DialTimeout: -time.Second,
			},
			wantErr: ErrRedisInvalidTimeout,
		},
		{
			name: "negative read timeout",
			config: RedisConfig{
				Network:     "tcp",
				Addr:        "localhost:6379",
				ReadTimeout: -time.Second,
			},
			wantErr: ErrRedisInvalidTimeout,
		},
		{
			name: "negative write timeout",
			config: RedisConfig{
				Network:      "tcp",
				Addr:         "localhost:6379",
				WriteTimeout: -time.Second,
			},
			wantErr: ErrRedisInvalidTimeout,
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
	require.NotNil(t, got.TLSConfig)
	assert.NotSame(t, tlsConfig, got.TLSConfig)
	assert.Equal(t, uint16(tls.VersionTLS12), got.TLSConfig.MinVersion)
}

func TestJobOptions(t *testing.T) {
	t.Parallel()

	scheduleAt := time.Now().Add(time.Hour)
	deadline := scheduleAt.Add(time.Hour)
	job := mustNewJob(t, "job:test", nil,
		WithDelay(time.Second),
		WithMaxRetries(3),
		WithQueue("critical"),
		WithScheduleAt(&scheduleAt),
		WithDeadline(&deadline),
		WithRetention(time.Hour),
	)
	options := job.Options()

	assert.Equal(t, time.Second, options.Delay)
	assert.Equal(t, 3, options.MaxRetries)
	assert.Equal(t, "critical", options.Queue)
	assert.Equal(t, scheduleAt, *options.ScheduleAt)
	assert.Equal(t, deadline, *options.Deadline)
	assert.NotSame(t, &scheduleAt, options.ScheduleAt)
	assert.NotSame(t, &deadline, options.Deadline)
	assert.Equal(t, time.Hour, options.Retention)
}

func TestNewJob_ValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		jobType string
		opts    []JobOption
		wantErr error
	}{
		{
			name:    "missing type",
			jobType: "",
			wantErr: ErrNoJobTypeSpecified,
		},
		{
			name:    "missing queue",
			jobType: "job:test",
			opts:    []JobOption{WithQueue("")},
			wantErr: ErrNoJobQueueSpecified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			job, err := NewJob(tt.jobType, nil, tt.opts...)
			assert.Nil(t, job)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestConvertToAsynqTask_NilJob(t *testing.T) {
	t.Parallel()

	var job *Job
	task, opts, err := job.ConvertToAsynqTask()
	assert.Nil(t, task)
	assert.Nil(t, opts)
	assert.ErrorIs(t, err, ErrInvalidJob)
}

func TestConvertToAsynqOptions_AllOptions(t *testing.T) {
	t.Parallel()

	scheduleAt := time.Now().Add(time.Hour)
	deadline := scheduleAt.Add(time.Hour)
	job := mustNewJob(t, "job:test", nil,
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
			return func(ctx context.Context, delivery *Delivery) error {
				calls = append(calls, name)
				return next(ctx, delivery)
			}
		}
	}
	retryDelay := func(count int, _ error) time.Duration {
		return time.Duration(count) * time.Second
	}
	handler := mustNewHandler(t, "job:test",
		func(context.Context, *Delivery) error {
			calls = append(calls, "handler")
			return nil
		},
		WithRateLimiter(limiter),
		WithJobQueue("critical"),
		WithJobTimeout(5*time.Second),
		WithRetryDelayFunc(retryDelay),
		WithMiddleware(middleware("first"), middleware("second")),
	)

	err := handler.Process(t.Context(), &Delivery{jobType: "job:test"})

	require.NoError(t, err)
	assert.Equal(t, "critical", handler.Queue())
	assert.Equal(t, 5*time.Second, handler.Timeout())
	assert.Same(t, limiter, handler.limiter)
	assert.Equal(t, 2*time.Second, handler.retryDelayFunc(2, assert.AnError))
	want := []string{"first", "second", "handler"}
	if diff := cmp.Diff(want, calls); diff != "" {
		t.Errorf("middleware calls mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertToAsynqTask_SerializationError(t *testing.T) {
	t.Parallel()

	payload := make(chan int)
	job, err := NewJob("test", payload)
	assert.Nil(t, job)
	assert.ErrorIs(t, err, ErrSerializationFailure)
}

func TestDecodePayload_SerializationError(t *testing.T) {
	t.Parallel()

	var job *Job
	var out string
	err := job.DecodePayload(&out)
	assert.ErrorIs(t, err, ErrInvalidJob)
}

func TestProcessWithTimeout_ContextCanceled(t *testing.T) {
	t.Parallel()

	handler := mustNewHandler(t, "test", func(ctx context.Context,
		_ *Delivery) error {
		<-ctx.Done()
		return ctx.Err()
	}, WithJobTimeout(100*time.Millisecond))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	delivery := &Delivery{jobType: "test"}
	err := handler.Process(ctx, delivery)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestProcessWithTimeout_Success(t *testing.T) {
	t.Parallel()

	handler := mustNewHandler(t, "test", func(_ context.Context,
		_ *Delivery) error {
		return nil
	}, WithJobTimeout(5*time.Second))

	delivery := &Delivery{jobType: "test"}
	err := handler.Process(t.Context(), delivery)
	require.NoError(t, err)
}

func TestWriteResult_SerializationError(t *testing.T) {
	t.Parallel()

	delivery := &Delivery{resultWriter: &asynq.ResultWriter{}}
	err := delivery.WriteResult(func() {})
	assert.ErrorIs(t, err, ErrSerializationFailure)
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
		{name: "aggregating requires group", state: StateAggregating, wantErr: ErrGroupRequiredForAggregation},
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

	client := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{"shard": "localhost:6379"},
	})
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})
	inspector := asynq.NewInspector(DefaultRedisConfig().ToAsynqRedisOpt())
	m, err := NewManager(client, inspector)
	require.NoError(t, err)

	_, err = m.RedisInfo(t.Context())
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

func TestNewDeliveryFromTask(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"key":"value"}`)
	task := asynq.NewTask("test:type", payload)
	delivery, err := newDeliveryFromTask(t.Context(), task, "critical")
	require.NoError(t, err)
	assert.Equal(t, "test:type", delivery.Type())
	assert.Equal(t, "critical", delivery.Queue())
	assert.Equal(t, 1, delivery.Attempt())
	assert.Zero(t, delivery.RetryCount())
	assert.Zero(t, delivery.MaxRetry())
	_, ok := delivery.Deadline()
	assert.False(t, ok)
	if diff := cmp.Diff(payload, delivery.rawPayload); diff != "" {
		t.Errorf("delivery payload mismatch (-want +got):\n%s", diff)
	}
}

func TestNewDeliveryFromTask_UsesContextDeadline(t *testing.T) {
	t.Parallel()

	deadline := time.Now().Add(time.Minute).Round(0)
	ctx, cancel := context.WithDeadline(t.Context(), deadline)
	defer cancel()

	task := asynq.NewTask("test:type", []byte(`{}`))
	delivery, err := newDeliveryFromTask(ctx, task, "default")
	require.NoError(t, err)

	got, ok := delivery.Deadline()
	require.True(t, ok)
	assert.Equal(t, deadline, got)
}

func TestNewDeliveryFromTask_AllowsDecodePayload(t *testing.T) {
	t.Parallel()

	task := asynq.NewTask("test:type", []byte(`{"key":"value"}`))
	delivery, err := newDeliveryFromTask(t.Context(), task, "default")
	require.NoError(t, err)

	var got struct {
		Key string `json:"key"`
	}
	err = delivery.DecodePayload(&got)
	require.NoError(t, err)
	assert.Equal(t, "value", got.Key)
}

func TestNewDeliveryFromTask_NilTask(t *testing.T) {
	t.Parallel()

	delivery, err := newDeliveryFromTask(t.Context(), nil, "default")
	assert.Nil(t, delivery)
	assert.ErrorIs(t, err, ErrInvalidAsynqTask)
}

func TestConvertToAsynqOptions_NoOptions(t *testing.T) {
	t.Parallel()

	job := &Job{options: JobOptions{}}
	opts := job.ConvertToAsynqOptions()
	assert.Empty(t, opts)
}

func TestConvertToAsynqOptions_ZeroScheduleAt(t *testing.T) {
	t.Parallel()

	zero := time.Time{}
	job := &Job{options: JobOptions{ScheduleAt: &zero}}
	opts := job.ConvertToAsynqOptions()
	assert.Empty(t, opts)
}

func TestConvertToAsynqOptions_ZeroDeadline(t *testing.T) {
	t.Parallel()

	zero := time.Time{}
	job := &Job{options: JobOptions{Deadline: &zero}}
	opts := job.ConvertToAsynqOptions()
	assert.Empty(t, opts)
}

func TestEnqueueJob_WithJobRetention(t *testing.T) {
	t.Parallel()

	job := mustNewJob(t, "test", map[string]string{"k": "v"},
		WithRetention(2*time.Hour),
	)
	assert.Equal(t, 2*time.Hour, job.Options().Retention)
}

func TestRedisInfo_StandardClient(t *testing.T) {
	t.Parallel()

	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: time.Millisecond,
	})
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})
	inspector := asynq.NewInspector(DefaultRedisConfig().ToAsynqRedisOpt())
	m, err := NewManager(client, inspector)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = m.RedisInfo(ctx)
	assert.ErrorIs(t, err, context.Canceled)
	assert.NotErrorIs(t, err, ErrRedisClientNotSupported)
}

func TestMemoryConfigProvider_RegisterDuplicate(t *testing.T) {
	t.Parallel()

	p := NewMemoryConfigProvider()
	job := mustNewJob(t, "test", nil)
	_, err := p.RegisterCronJob("test-schedule", "* * * * *", job)
	require.NoError(t, err)

	id, err := p.RegisterCronJob("test-schedule", "* * * * *", job)

	assert.Empty(t, id)
	assert.ErrorIs(t, err, ErrScheduleAlreadyExists)
}

func TestGetConfigs_WithEntries(t *testing.T) {
	t.Parallel()

	p := NewMemoryConfigProvider()
	j := mustNewJob(t, "test", nil)
	_, err := p.RegisterCronJob("test-schedule", "* * * * *", j)
	require.NoError(t, err)

	configs, err := p.GetConfigs()
	require.NoError(t, err)
	assert.Len(t, configs, 1)
	assert.Equal(t, "* * * * *", configs[0].Cronspec)
	assert.Equal(t, "test", configs[0].Task.Type())
}

func TestRegisterCronJob_NilJob(t *testing.T) {
	t.Parallel()

	p := NewMemoryConfigProvider()
	_, err := p.RegisterCronJob("test-schedule", "* * * * *", nil)
	assert.ErrorIs(t, err, ErrInvalidJob)
}
