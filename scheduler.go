package queue

import (
	"errors"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"github.com/robfig/cron/v3"
)

var ErrInvalidCronSpec = errors.New("invalid cron spec")

// Scheduler manages job scheduling with Asynq.
type Scheduler struct {
	taskManager    *asynq.PeriodicTaskManager
	configProvider ConfigProvider
	options        SchedulerOptions
}

// SchedulerOptions contains options for the Scheduler.
type SchedulerOptions struct {
	SyncInterval    time.Duration
	Location        *time.Location
	ConfigProvider  ConfigProvider
	Logger          Logger
	PreEnqueueFunc  func(job *Job)                // Pre-enqueue hook
	PostEnqueueFunc func(job *JobInfo, err error) // Post-enqueue hook
}

// SchedulerOption defines a function signature for configuring the Scheduler.
type SchedulerOption func(*SchedulerOptions)

// WithSyncInterval sets the sync interval for the Scheduler's task manager.
func WithSyncInterval(interval time.Duration) SchedulerOption {
	return func(opts *SchedulerOptions) {
		opts.SyncInterval = interval
	}
}

// WithSchedulerLocation sets the time location for the Scheduler.
func WithSchedulerLocation(loc *time.Location) SchedulerOption {
	return func(opts *SchedulerOptions) {
		opts.Location = loc
	}
}

// WithConfigProvider sets a custom config provider for the Scheduler.
func WithConfigProvider(provider ConfigProvider) SchedulerOption {
	return func(opts *SchedulerOptions) {
		opts.ConfigProvider = provider
	}
}

// WithSchedulerLogger sets a custom logger for the Scheduler.
func WithSchedulerLogger(logger Logger) SchedulerOption {
	return func(opts *SchedulerOptions) {
		opts.Logger = logger
	}
}

// WithPreEnqueueFunc sets a function to be called before enqueuing a job.
func WithPreEnqueueFunc(fn func(job *Job)) SchedulerOption {
	return func(opts *SchedulerOptions) {
		opts.PreEnqueueFunc = fn
	}
}

// WithPostEnqueueFunc sets a function to be called after enqueuing a job.
func WithPostEnqueueFunc(fn func(job *JobInfo, err error)) SchedulerOption {
	return func(opts *SchedulerOptions) {
		opts.PostEnqueueFunc = fn
	}
}

// NewScheduler creates a new Scheduler instance with the provided Redis configuration and options.
func NewScheduler(redisConfig *RedisConfig, opts ...SchedulerOption) (*Scheduler, error) {
	if redisConfig == nil {
		return nil, ErrInvalidRedisConfig
	}

	asynqClientOpt := redisConfig.ToAsynqRedisOpt()

	options := SchedulerOptions{
		Location:     time.UTC, // Default to UTC
		SyncInterval: 60 * time.Second,
	}
	for _, opt := range opts {
		opt(&options)
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
				Logger:   options.Logger,
				PreEnqueueFunc: func(task *asynq.Task, opts []asynq.Option) {
					if options.PreEnqueueFunc != nil {
						job, _ := NewJobFromAsynqTask(task)
						options.PreEnqueueFunc(job)
					}
				},
				PostEnqueueFunc: func(taskInfo *asynq.TaskInfo, err error) {
					log.Printf("Enqueued task: %v, err: %v", taskInfo, err)
					if options.PostEnqueueFunc != nil {
						jobInfo := toJobInfo(taskInfo, nil)
						options.PostEnqueueFunc(jobInfo, err)
					}
				},
			},
		})

	if err != nil {
		return nil, err
	}

	return &Scheduler{
		taskManager:    taskManager,
		configProvider: configProvider,
		options:        options,
	}, nil
}

// RegisterCron schedules a new cron job using the job type, payload, and options.
func (s *Scheduler) RegisterCron(spec string, jobType string, payload interface{}, opts ...JobOption) (string, error) {
	job := NewJob(jobType, payload, opts...)
	return s.RegisterCronJob(spec, job)
}

// RegisterCronJob schedules a new cron job using the job details.
func (s *Scheduler) RegisterCronJob(spec string, job *Job) (string, error) {
	// Use cron/v3 to parse the spec and check if it's a valid cron expression.
	_, err := cron.ParseStandard(spec)
	if err != nil {
		return "", ErrInvalidCronSpec
	}

	return s.configProvider.RegisterCronJob(spec, job)
}

// RegisterPeriodic schedules a new periodic job using the job type, payload, and options.
func (s *Scheduler) RegisterPeriodic(interval time.Duration, jobType string, payload interface{}, opts ...JobOption) (string, error) {
	job := NewJob(jobType, payload, opts...)
	return s.RegisterPeriodicJob(interval, job)
}

// RegisterPeriodicJob schedules a new periodic job using the job details and an interval.
func (s *Scheduler) RegisterPeriodicJob(interval time.Duration, job *Job) (string, error) {
	spec := "@every " + interval.String()
	return s.configProvider.RegisterCronJob(spec, job)
}

// Start begins the scheduler to enqueue tasks as per the schedule.
func (s *Scheduler) Start() error {
	return s.taskManager.Run()
}

// Stop gracefully shuts down the scheduler.
func (s *Scheduler) Stop() error {
	s.taskManager.Shutdown()
	return nil
}
