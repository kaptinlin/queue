package queue

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/go-json-experiment/json"
	"github.com/hibiken/asynq"
)

// Job is an immutable enqueue specification.
type Job struct {
	jobType       string
	payloadBytes  []byte
	options       JobOptions
	contentDigest string
}

// JobOptions encapsulates settings that control job execution.
type JobOptions struct {
	MaxRetries int           `json:"max_retries"` // Maximum number of retries.
	Queue      string        `json:"queue"`       // Queue name to which the job is dispatched.
	Delay      time.Duration `json:"delay"`       // Initial delay before processing the job.
	ScheduleAt *time.Time    `json:"schedule_at"` // Specific time at which the job should be processed.
	Deadline   *time.Time    `json:"deadline"`    // Time by which the job must complete.
	Retention  time.Duration `json:"retention"`   // Duration to retain the job data after completion.
}

// NewJob validates and initializes an immutable Job.
func NewJob(jobType string, payload any, opts ...JobOption) (*Job, error) {
	if jobType == "" {
		return nil, ErrNoJobTypeSpecified
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize payload: %w: %w", ErrSerializationFailure, err)
	}

	options := JobOptions{Queue: DefaultQueue}
	for _, opt := range opts {
		opt.applyJobOption(&options)
	}
	if err := validateJobOptions(options); err != nil {
		return nil, err
	}

	payloadBytes = append([]byte{}, payloadBytes...)
	return &Job{
		jobType:       jobType,
		payloadBytes:  payloadBytes,
		options:       cloneJobOptions(options),
		contentDigest: contentDigest(jobType, payloadBytes),
	}, nil
}

// Type returns the job type used for handler routing.
func (j *Job) Type() string {
	if j == nil {
		return ""
	}
	return j.jobType
}

// Options returns a copy of the job execution options.
func (j *Job) Options() JobOptions {
	if j == nil {
		return JobOptions{}
	}
	return cloneJobOptions(j.options)
}

// ContentDigest returns a stable digest derived from the job type and payload.
func (j *Job) ContentDigest() string {
	if j == nil {
		return ""
	}
	return j.contentDigest
}

// PayloadBytes returns a copy of the encoded job payload.
func (j *Job) PayloadBytes() []byte {
	if j == nil {
		return nil
	}
	return append([]byte{}, j.payloadBytes...)
}

// JobOption configures a Job.
type JobOption interface {
	applyJobOption(*JobOptions)
}

type jobOption func(*JobOptions)

func (f jobOption) applyJobOption(options *JobOptions) {
	f(options)
}

// WithDelay sets the initial delay before the job is processed.
func WithDelay(delay time.Duration) JobOption {
	return jobOption(func(opts *JobOptions) { opts.Delay = delay })
}

// WithMaxRetries sets the maximum number of retry attempts for the job.
func WithMaxRetries(maxRetries int) JobOption {
	return jobOption(func(opts *JobOptions) { opts.MaxRetries = maxRetries })
}

// WithQueue sets the queue name to which the job is dispatched.
func WithQueue(queue string) JobOption {
	return jobOption(func(opts *JobOptions) { opts.Queue = queue })
}

// WithScheduleAt sets a specific time at which the job should be processed.
func WithScheduleAt(scheduleAt *time.Time) JobOption {
	return jobOption(func(opts *JobOptions) {
		if scheduleAt == nil {
			opts.ScheduleAt = nil
			return
		}
		snapshot := *scheduleAt
		opts.ScheduleAt = &snapshot
	})
}

// WithRetention sets the duration to retain the job data after completion.
func WithRetention(retention time.Duration) JobOption {
	return jobOption(func(opts *JobOptions) { opts.Retention = retention })
}

// WithDeadline sets the time by which the job must complete.
func WithDeadline(deadline *time.Time) JobOption {
	return jobOption(func(opts *JobOptions) {
		if deadline == nil {
			opts.Deadline = nil
			return
		}
		snapshot := *deadline
		opts.Deadline = &snapshot
	})
}

// ConvertToAsynqTask converts the Job into an asynq.Task, ready for enqueueing.
func (j *Job) ConvertToAsynqTask() (*asynq.Task, []asynq.Option, error) {
	if j == nil {
		return nil, nil, ErrInvalidJob
	}

	opts := j.ConvertToAsynqOptions()
	return asynq.NewTask(j.jobType, j.PayloadBytes()), opts, nil
}

// ConvertToAsynqOptions converts the Job's options into asynq.Option slice.
func (j *Job) ConvertToAsynqOptions() []asynq.Option {
	if j == nil {
		return nil
	}

	opts := make([]asynq.Option, 0, 6)
	options := j.options

	if options.Queue != "" {
		opts = append(opts, asynq.Queue(options.Queue))
	}
	if options.Delay > 0 {
		opts = append(opts, asynq.ProcessIn(options.Delay))
	}
	if options.ScheduleAt != nil && !options.ScheduleAt.IsZero() {
		opts = append(opts, asynq.ProcessAt(*options.ScheduleAt))
	}
	if options.MaxRetries > 0 {
		opts = append(opts, asynq.MaxRetry(options.MaxRetries))
	}
	if options.Deadline != nil && !options.Deadline.IsZero() {
		opts = append(opts, asynq.Deadline(*options.Deadline))
	}
	if options.Retention > 0 {
		opts = append(opts, asynq.Retention(options.Retention))
	}

	return opts
}

func validateJobOptions(options JobOptions) error {
	if options.Queue == "" {
		return ErrNoJobQueueSpecified
	}
	if options.MaxRetries < 0 {
		return fmt.Errorf("%w: max retries cannot be negative", ErrInvalidJobOptions)
	}
	if options.Delay < 0 {
		return fmt.Errorf("%w: delay cannot be negative", ErrInvalidJobOptions)
	}
	if options.Retention < 0 {
		return fmt.Errorf("%w: retention cannot be negative", ErrInvalidJobOptions)
	}
	return nil
}

func cloneJobOptions(options JobOptions) JobOptions {
	clone := options
	if options.ScheduleAt != nil {
		scheduleAt := *options.ScheduleAt
		clone.ScheduleAt = &scheduleAt
	}
	if options.Deadline != nil {
		deadline := *options.Deadline
		clone.Deadline = &deadline
	}
	return clone
}

func contentDigest(jobType string, payloadBytes []byte) string {
	hash := sha256.New()
	hash.Write([]byte(jobType))
	hash.Write([]byte{0})
	hash.Write(payloadBytes)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// DecodePayload decodes the job payload into a given struct.
func (j *Job) DecodePayload(v any) error {
	if j == nil {
		return ErrInvalidJob
	}
	return json.Unmarshal(j.payloadBytes, v)
}
