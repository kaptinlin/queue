package queue

import (
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Job errors are returned when job validation or processing fails.
var (
	// ErrNoJobQueueSpecified is returned when a job has no queue assigned.
	ErrNoJobQueueSpecified = errors.New("job requires a specified queue")
	// ErrNoJobTypeSpecified is returned when a job has no type assigned.
	ErrNoJobTypeSpecified = errors.New("job requires a specified type")
	// ErrJobExceededDeadline is returned when a job fails to complete
	// before its deadline.
	ErrJobExceededDeadline = errors.New("job failed to complete by deadline")
	// ErrJobExceededMaxRetries is returned when a job has exhausted all
	// retry attempts.
	ErrJobExceededMaxRetries = errors.New("job exceeded maximum retry attempts")
	// ErrJobProcessingTimeout is returned when a job exceeds its
	// configured processing timeout.
	ErrJobProcessingTimeout = errors.New("job processing exceeded timeout")
	// ErrScheduledTimeInPast is returned when a job is scheduled for a
	// time that has already passed.
	ErrScheduledTimeInPast = errors.New("scheduled time must be in the future")
	// ErrSerializationFailure is returned when job payload serialization
	// or deserialization fails.
	ErrSerializationFailure = errors.New("failure in serialization process")
	// ErrResultWriterNotSet is returned when [Job.WriteResult] is called
	// but no result writer has been configured.
	ErrResultWriterNotSet = errors.New("result writer is not set for the job")
	// ErrFailedToWriteResult is returned when writing a job result to
	// Redis fails.
	ErrFailedToWriteResult = errors.New("failed to write job result")
	// ErrEnqueueJob is returned when the client fails to enqueue a job.
	ErrEnqueueJob = errors.New("unable to enqueue job")
	// ErrTransientIssue indicates a temporary failure that should be
	// retried without counting against the job's retry limit.
	ErrTransientIssue = errors.New("transient issue, job will retry without affecting retry count")
	// ErrInvalidJobState is returned when an invalid [JobState] is
	// provided to a manager operation.
	ErrInvalidJobState = errors.New("invalid job state")
	// ErrJobNotFound is returned when a job cannot be found by its ID.
	ErrJobNotFound = errors.New("job not found")
)

// Worker errors are returned when worker configuration or lifecycle operations fail.
var (
	// ErrInvalidWorkerConfig is returned when the worker configuration
	// fails validation.
	ErrInvalidWorkerConfig = errors.New("invalid worker configuration")
	// ErrInvalidWorkerQueues is returned when no queues are configured
	// for the worker.
	ErrInvalidWorkerQueues = errors.New("worker requires at least one queue")
	// ErrInvalidWorkerConcurrency is returned when the worker concurrency
	// is set to zero or a negative value.
	ErrInvalidWorkerConcurrency = errors.New("worker requires a positive concurrency value")
	// ErrWorkerAlreadyStarted is returned when [Worker.Start] is called
	// on a worker that is already running.
	ErrWorkerAlreadyStarted = errors.New("worker already started")
	// ErrHandlerAlreadyRegistered is returned when a handler for the
	// same job type is registered more than once.
	ErrHandlerAlreadyRegistered = errors.New("handler already registered")
	// ErrWorkerNotFound is returned when a worker cannot be found by
	// its ID.
	ErrWorkerNotFound = errors.New("worker not found")
)

// Redis errors are returned when Redis configuration validation fails.
var (
	// ErrInvalidRedisConfig is returned when the Redis configuration
	// fails validation.
	ErrInvalidRedisConfig = errors.New("invalid redis configuration")
	// ErrRedisEmptyAddress is returned when the Redis address is empty.
	ErrRedisEmptyAddress = errors.New("address cannot be empty")
	// ErrRedisUnsupportedNetwork is returned when an unsupported
	// network type is specified in the Redis configuration.
	ErrRedisUnsupportedNetwork = errors.New("unsupported network type")
	// ErrRedisInvalidAddress is returned when the Redis address format
	// is invalid.
	ErrRedisInvalidAddress = errors.New("invalid address format")
	// ErrRedisTLSRequired is returned when a rediss:// URL is used
	// without providing a TLS configuration.
	ErrRedisTLSRequired = errors.New("TLS configuration required for rediss:// connections")
	// ErrRedisClientNotSupported is returned when the Redis client type
	// is not supported by the manager.
	ErrRedisClientNotSupported = errors.New("redis client type not supported")
)

// Manager errors are returned when queue management operations fail.
var (
	// ErrOperationNotSupported is returned when the requested operation
	// is not supported for the given job state.
	ErrOperationNotSupported = errors.New("operation not supported for the given job state")
	// ErrArchivingActiveJobs is returned when attempting to archive
	// jobs that are currently being processed. Cancel them first.
	ErrArchivingActiveJobs = errors.New("cannot archive active jobs directly, cancel first")
	// ErrGroupRequiredForAggregation is returned when an aggregating
	// operation is attempted without specifying a group identifier.
	ErrGroupRequiredForAggregation = errors.New("group identifier required for aggregating jobs")
	// ErrUnsupportedJobStateForAction is returned when the job state
	// does not support the requested action.
	ErrUnsupportedJobStateForAction = errors.New("unsupported job state for the requested action")
	// ErrQueueNotFound is returned when the specified queue does not exist.
	ErrQueueNotFound = errors.New("queue not found")
	// ErrQueueNotEmpty is returned when attempting to delete a queue
	// that still contains jobs without using force mode.
	ErrQueueNotEmpty = errors.New("queue is not empty")
)

// ErrSkipRetry indicates a specific Asynq framework condition to skip retries and move the job to the archive.
var ErrSkipRetry = asynq.SkipRetry

// NewSkipRetryError creates and wraps a SkipRetry error with a custom message.
func NewSkipRetryError(reason string) error {
	return fmt.Errorf("skip retry due to: %s: %w", reason, ErrSkipRetry)
}

// ErrRateLimit defines a custom error type for rate limiting scenarios.
type ErrRateLimit struct {
	RetryAfter time.Duration // Suggested time to wait before retrying the operation.
}

// Error implements the error interface for ErrRateLimit.
func (e *ErrRateLimit) Error() string {
	return fmt.Sprintf("rate limited: retry after %v", e.RetryAfter)
}

// NewErrRateLimit constructs a new ErrRateLimit with a specified retry delay.
func NewErrRateLimit(retryAfter time.Duration) *ErrRateLimit {
	return &ErrRateLimit{
		RetryAfter: retryAfter,
	}
}

// IsErrRateLimit checks if the provided error is or wraps an ErrRateLimit error.
func IsErrRateLimit(err error) bool {
	_, ok := errors.AsType[*ErrRateLimit](err)
	return ok
}
