package queue

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Job represents a task that will be executed by a worker.
type Job struct {
	Fingerprint string      `json:"fingerprint"` // Unique identifier for the job.
	Type        string      `json:"type"`        // Type of job, used for handler mapping.
	Payload     interface{} `json:"payload"`     // Job data.
	Options     JobOptions  `json:"options"`     // Execution options for the job.
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

// NewJob initializes a new Job with the provided type, payload, and configuration options.
func NewJob(jobType string, payload interface{}, opts ...JobOption) *Job {
	job := &Job{
		Type:    jobType,
		Payload: payload,
		Options: JobOptions{Queue: DefaultQueue}, // Use a default queue unless overridden.
	}

	// Apply provided configuration options to the job.
	for _, opt := range opts {
		opt(job)
	}

	job.fingerprint() // Generate a unique fingerprint for the job.

	return job
}

// JobOption defines a function signature for job configuration options.
type JobOption func(*Job)

// Job configuration options follow, allowing customization of the job's behavior.

func WithDelay(delay time.Duration) JobOption {
	return func(j *Job) { j.Options.Delay = delay }
}

func WithMaxRetries(maxRetries int) JobOption {
	return func(j *Job) { j.Options.MaxRetries = maxRetries }
}

func WithQueue(queue string) JobOption {
	return func(j *Job) { j.Options.Queue = queue }
}

func WithScheduleAt(scheduleAt *time.Time) JobOption {
	return func(j *Job) { j.Options.ScheduleAt = scheduleAt }
}

func WithRetention(retention time.Duration) JobOption {
	return func(j *Job) { j.Options.Retention = retention }
}

func WithDeadline(deadline *time.Time) JobOption {
	return func(j *Job) { j.Options.Deadline = deadline }
}

// ConvertToAsynqTask converts the Job into an Asynq task, ready for enqueueing.
func (j *Job) ConvertToAsynqTask() (*asynq.Task, error) {
	if j.Type == "" {
		return nil, ErrNoJobTypeSpecified
	}

	if j.Options.Queue == "" {
		return nil, ErrNoJobQueueSpecified
	}

	payloadBytes, err := json.Marshal(j.Payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSerializationFailure, err)
	}

	opts := make([]asynq.Option, 0)

	// Apply job options to the Asynq task.
	if j.Options.Delay > 0 {
		opts = append(opts, asynq.ProcessIn(j.Options.Delay))
	}
	if j.Options.ScheduleAt != nil && !j.Options.ScheduleAt.IsZero() {
		opts = append(opts, asynq.ProcessAt(*j.Options.ScheduleAt))
	}
	if j.Options.MaxRetries > 0 {
		opts = append(opts, asynq.MaxRetry(j.Options.MaxRetries))
	}
	if j.Options.Deadline != nil && !j.Options.Deadline.IsZero() {
		opts = append(opts, asynq.Deadline(*j.Options.Deadline))
	}
	if j.Options.Retention > 0 {
		opts = append(opts, asynq.Retention(j.Options.Retention))
	}

	return asynq.NewTask(j.Type, payloadBytes, opts...), nil
}

// fingerprint generates a unique hash for the job based on its type and payload.
func (j *Job) fingerprint() {
	if j.Fingerprint != "" {
		return // Fingerprint already set, no need to regenerate.
	}

	hash := md5.New()
	hash.Write([]byte(j.Type))
	payloadBytes, _ := json.Marshal(j.Payload)
	hash.Write(payloadBytes)

	j.Fingerprint = fmt.Sprintf("%x", hash.Sum(nil))
}

// DecodePayload decodes the job payload into a given struct.
func (j *Job) DecodePayload(v interface{}) error {
	payloadBytes, err := json.Marshal(j.Payload)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSerializationFailure, err)
	}
	return json.Unmarshal(payloadBytes, v)
}