package queue

import (
	"time"

	"github.com/hibiken/asynq"
)

// Scheduler manages job scheduling with Asynq.
type Scheduler struct {
	scheduler *asynq.Scheduler
	options   SchedulerOptions
}

// SchedulerOptions contains options for the Scheduler.
type SchedulerOptions struct {
	Location            *time.Location
	EnqueueErrorHandler func(*Job, error)
}

// SchedulerOption defines a function signature for configuring the Scheduler.
type SchedulerOption func(*SchedulerOptions)

// WithSchedulerLocation sets the time location for the Scheduler.
func WithSchedulerLocation(loc *time.Location) SchedulerOption {
	return func(opts *SchedulerOptions) {
		opts.Location = loc
	}
}

// WithSchedulerEnqueueErrorHandler sets the enqueue error handler for the Scheduler.
func WithSchedulerEnqueueErrorHandler(handler func(*Job, error)) SchedulerOption {
	return func(opts *SchedulerOptions) {
		opts.EnqueueErrorHandler = handler
	}
}

// NewScheduler creates a new Scheduler instance with the provided Redis configuration and options.
func NewScheduler(redisConfig *RedisConfig, opts ...SchedulerOption) (*Scheduler, error) {
	if redisConfig == nil {
		return nil, ErrInvalidRedisConfig
	}

	asynqClientOpt := redisConfig.ToAsynqRedisOpt()

	options := SchedulerOptions{
		Location: time.UTC, // Default to UTC
	}
	for _, opt := range opts {
		opt(&options)
	}

	scheduler := asynq.NewScheduler(asynqClientOpt, &asynq.SchedulerOpts{
		Location: options.Location,
		EnqueueErrorHandler: func(task *asynq.Task, opts []asynq.Option, err error) {
			if options.EnqueueErrorHandler != nil {
				job, _ := NewJobFromAsynqTask(task)
				options.EnqueueErrorHandler(job, err)
			}
		},
	})

	return &Scheduler{
		scheduler: scheduler,
		options:   options,
	}, nil
}

// RegisterCron schedules a new cron job using the job type, payload, and options.
func (s *Scheduler) RegisterCron(spec string, jobType string, payload interface{}, opts ...JobOption) (string, error) {
	job := NewJob(jobType, payload, opts...)
	return s.RegisterCronJob(spec, job)
}

// RegisterCronJob schedules a new cron job using the job details.
func (s *Scheduler) RegisterCronJob(spec string, job *Job) (string, error) {
	task, err := job.ConvertToAsynqTask()
	if err != nil {
		return "", err
	}

	entryID, err := s.scheduler.Register(spec, task)
	if err != nil {
		if s.options.EnqueueErrorHandler != nil {
			s.options.EnqueueErrorHandler(job, err)
		}
		return "", err
	}

	return entryID, nil
}

// RegisterPeriodic schedules a new periodic job using the job type, payload, and options.
func (s *Scheduler) RegisterPeriodic(interval time.Duration, jobType string, payload interface{}, opts ...JobOption) (string, error) {
	job := NewJob(jobType, payload, opts...)
	return s.RegisterPeriodicJob(interval, job)
}

// RegisterPeriodicJob schedules a new periodic job using the job details and an interval.
func (s *Scheduler) RegisterPeriodicJob(interval time.Duration, job *Job) (string, error) {
	spec := "@every " + interval.String()
	return s.RegisterCronJob(spec, job)
}

// Start begins the scheduler to enqueue tasks as per the schedule.
func (s *Scheduler) Start() error {
	return s.scheduler.Run()
}

// Stop gracefully shuts down the scheduler.
func (s *Scheduler) Stop() error {
	s.scheduler.Shutdown()
	return nil
}
