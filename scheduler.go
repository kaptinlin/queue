package queue

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/hibiken/asynq"
	"github.com/netresearch/go-cron"
)

// ErrInvalidCronSpec is returned when a cron specification string cannot be parsed.
var ErrInvalidCronSpec = errors.New("invalid cron spec")

// Scheduler manages periodic job scheduling.
type Scheduler struct {
	taskManager    *asynq.PeriodicTaskManager
	configProvider ConfigProvider
	started        atomic.Bool
	stopped        atomic.Bool
}

// schedulerOptions contains options for the Scheduler.
type schedulerOptions struct {
	SyncInterval   time.Duration
	Location       *time.Location
	ConfigProvider ConfigProvider
	Logger         Logger
}

// SchedulerOption configures a Scheduler.
type SchedulerOption interface {
	applySchedulerOption(*schedulerOptions)
}

type schedulerOption func(*schedulerOptions)

func (f schedulerOption) applySchedulerOption(options *schedulerOptions) {
	f(options)
}

// WithSyncInterval sets the sync interval for the Scheduler's task manager.
func WithSyncInterval(interval time.Duration) SchedulerOption {
	return schedulerOption(func(opts *schedulerOptions) {
		opts.SyncInterval = interval
	})
}

// WithSchedulerLocation sets the time location for the Scheduler.
func WithSchedulerLocation(loc *time.Location) SchedulerOption {
	return schedulerOption(func(opts *schedulerOptions) {
		opts.Location = loc
	})
}

// WithConfigProvider sets a custom config provider for the Scheduler.
func WithConfigProvider(provider ConfigProvider) SchedulerOption {
	return schedulerOption(func(opts *schedulerOptions) {
		opts.ConfigProvider = provider
	})
}

// WithSchedulerLogger sets a custom logger for the Scheduler.
func WithSchedulerLogger(logger Logger) SchedulerOption {
	return schedulerOption(func(opts *schedulerOptions) {
		opts.Logger = logger
	})
}

// NewScheduler creates a new Scheduler instance with the provided Redis configuration and options.
func NewScheduler(redisConfig *RedisConfig, opts ...SchedulerOption) (*Scheduler, error) {
	if redisConfig == nil {
		return nil, ErrInvalidRedisConfig
	}
	if err := redisConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid redis config: %w", err)
	}

	asynqClientOpt := redisConfig.ToAsynqRedisOpt()

	options := schedulerOptions{
		Location:     time.UTC,
		SyncInterval: 60 * time.Second,
	}
	for _, opt := range opts {
		opt.applySchedulerOption(&options)
	}
	if options.SyncInterval <= 0 {
		return nil, ErrInvalidSyncInterval
	}
	if options.Location == nil {
		options.Location = time.UTC
	}

	logger := options.Logger
	if logger == nil {
		logger = NewDefaultLogger()
	}

	configProvider := options.ConfigProvider
	if configProvider == nil {
		configProvider = NewMemoryConfigProvider()
	}

	taskManager, err := asynq.NewPeriodicTaskManager(
		asynq.PeriodicTaskManagerOpts{
			RedisConnOpt:               asynqClientOpt,
			PeriodicTaskConfigProvider: configProvider,
			SyncInterval:               options.SyncInterval,
			SchedulerOpts: &asynq.SchedulerOpts{
				Location: options.Location,
				Logger:   newAsynqLogger(logger),
				PostEnqueueFunc: func(taskInfo *asynq.TaskInfo, err error) {
					if err != nil {
						logger.Error("failed to enqueue scheduled task", "error", err)
						return
					}
					if taskInfo == nil {
						logger.Info("enqueued scheduled task")
						return
					}
					logger.Info("enqueued scheduled task",
						"job_id", taskInfo.ID,
						"job_type", taskInfo.Type,
						"queue", taskInfo.Queue,
					)
				},
			},
		})

	if err != nil {
		return nil, err
	}

	return &Scheduler{
		taskManager:    taskManager,
		configProvider: configProvider,
	}, nil
}

// RegisterCron schedules a new cron job using an explicit schedule identifier.
func (s *Scheduler) RegisterCron(identifier, spec, jobType string, payload any, opts ...JobOption) (string, error) {
	job, err := NewJob(jobType, payload, opts...)
	if err != nil {
		return "", err
	}
	return s.RegisterCronJob(identifier, spec, job)
}

// RegisterCronJob schedules a new cron job using an explicit schedule identifier.
func (s *Scheduler) RegisterCronJob(identifier, spec string, job *Job) (string, error) {
	if _, err := cron.ParseStandard(spec); err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidCronSpec, err)
	}
	return s.configProvider.RegisterCronJob(identifier, spec, job)
}

// RegisterPeriodic schedules a new periodic job using an explicit schedule identifier.
func (s *Scheduler) RegisterPeriodic(identifier string, interval time.Duration, jobType string, payload any, opts ...JobOption) (string, error) {
	job, err := NewJob(jobType, payload, opts...)
	if err != nil {
		return "", err
	}
	return s.RegisterPeriodicJob(identifier, interval, job)
}

// RegisterPeriodicJob schedules a new periodic job using an explicit schedule identifier.
func (s *Scheduler) RegisterPeriodicJob(identifier string, interval time.Duration, job *Job) (string, error) {
	if interval <= 0 {
		return "", ErrInvalidPeriodicInterval
	}
	spec := "@every " + interval.String()
	return s.configProvider.RegisterCronJob(identifier, spec, job)
}

// UnregisterCronJob removes a scheduled cron job using its identifier.
func (s *Scheduler) UnregisterCronJob(identifier string) error {
	return s.configProvider.UnregisterJob(identifier)
}

// Run starts the scheduler and blocks until ctx is canceled.
func (s *Scheduler) Run(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalidContext
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.stopped.Load() {
		return ErrSchedulerStopped
	}
	if !s.started.CompareAndSwap(false, true) {
		return ErrSchedulerAlreadyStarted
	}

	if err := s.taskManager.Start(); err != nil {
		s.started.Store(false)
		return err
	}

	<-ctx.Done()
	if s.started.CompareAndSwap(true, false) {
		s.shutdown()
	}

	return nil
}

func (s *Scheduler) shutdown() {
	s.taskManager.Shutdown()
	s.stopped.Store(true)
}
