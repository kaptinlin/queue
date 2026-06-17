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
	taskManager *asynq.PeriodicTaskManager
	store       ScheduleStore
	started     atomic.Bool
	stopped     atomic.Bool
}

// schedulerOptions contains options for the Scheduler.
type schedulerOptions struct {
	SyncInterval time.Duration
	Location     *time.Location
	Store        ScheduleStore
	Logger       Logger
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

// WithScheduleStore sets a custom schedule store for the Scheduler.
func WithScheduleStore(store ScheduleStore) SchedulerOption {
	return schedulerOption(func(opts *schedulerOptions) {
		opts.Store = store
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

	redisOpt := asynqRedisOpt(redisConfig)

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

	store := options.Store
	if store == nil {
		store = NewMemoryScheduleStore()
	}

	configProvider := &asynqScheduleProvider{
		store:   store,
		timeout: options.SyncInterval,
	}
	taskManager, err := asynq.NewPeriodicTaskManager(
		asynq.PeriodicTaskManagerOpts{
			RedisConnOpt:               redisOpt,
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
		taskManager: taskManager,
		store:       store,
	}, nil
}

// RegisterCron schedules a job on a cron expression using an explicit schedule ID.
func (s *Scheduler) RegisterCron(ctx context.Context, id, spec string, job *Job) (string, error) {
	if err := validateContext(ctx); err != nil {
		return "", err
	}
	if _, err := cron.ParseStandard(spec); err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidCronSpec, err)
	}
	schedule := Schedule{
		ID:       id,
		Kind:     ScheduleCron,
		CronSpec: spec,
		Job:      job,
		Enabled:  true,
	}
	if err := s.store.Put(ctx, schedule); err != nil {
		return "", err
	}
	return id, nil
}

// RegisterInterval schedules a job at a fixed interval using an explicit schedule ID.
func (s *Scheduler) RegisterInterval(ctx context.Context, id string, interval time.Duration, job *Job) (string, error) {
	if err := validateContext(ctx); err != nil {
		return "", err
	}
	if interval <= 0 {
		return "", ErrInvalidPeriodicInterval
	}
	schedule := Schedule{
		ID:       id,
		Kind:     ScheduleInterval,
		Interval: interval,
		Job:      job,
		Enabled:  true,
	}
	if err := s.store.Put(ctx, schedule); err != nil {
		return "", err
	}
	return id, nil
}

// Unregister removes a schedule by ID.
func (s *Scheduler) Unregister(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
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

type asynqScheduleProvider struct {
	store   ScheduleStore
	timeout time.Duration
}

func (p *asynqScheduleProvider) GetConfigs() ([]*asynq.PeriodicTaskConfig, error) {
	ctx := context.Background()
	if p.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	schedules, err := p.store.List(ctx)
	if err != nil {
		return nil, err
	}

	configs := make([]*asynq.PeriodicTaskConfig, 0, len(schedules))
	for _, schedule := range schedules {
		if !schedule.Enabled {
			continue
		}
		spec, err := asynqScheduleSpec(schedule)
		if err != nil {
			return nil, err
		}
		task, opts, err := schedule.Job.convertToAsynqTask(schedule.Job.options)
		if err != nil {
			return nil, err
		}
		configs = append(configs, &asynq.PeriodicTaskConfig{
			Cronspec: spec,
			Task:     task,
			Opts:     opts,
		})
	}
	return configs, nil
}

func asynqScheduleSpec(schedule Schedule) (string, error) {
	switch schedule.Kind {
	case ScheduleCron:
		return schedule.CronSpec, nil
	case ScheduleInterval:
		if schedule.Interval <= 0 {
			return "", ErrInvalidPeriodicInterval
		}
		return "@every " + schedule.Interval.String(), nil
	default:
		return "", ErrInvalidScheduleKind
	}
}
