package queue

import (
	"errors"
	"fmt"
	"time"
)

// Job errors are returned when job validation or processing fails.
var (
	// ErrNoJobQueueSpecified is returned when a job has no queue assigned.
	ErrNoJobQueueSpecified = errors.New("job requires a specified queue")
	// ErrNoJobTypeSpecified is returned when a job has no type assigned.
	ErrNoJobTypeSpecified = errors.New("job requires a specified type")
	// ErrInvalidJob is returned when a nil or otherwise invalid job is used.
	ErrInvalidJob = errors.New("invalid job")
	// ErrInvalidJobOptions is returned when job options contain invalid values.
	ErrInvalidJobOptions = errors.New("invalid job options")
	// ErrInvalidHandler is returned when a nil or otherwise invalid handler is used.
	ErrInvalidHandler = errors.New("invalid handler")
	// ErrNoHandlerFuncSpecified is returned when a handler has no processing function.
	ErrNoHandlerFuncSpecified = errors.New("handler requires a processing function")
	// ErrInvalidAsynqTask is returned when an Asynq task cannot produce a delivery.
	ErrInvalidAsynqTask = errors.New("invalid asynq task")
	// ErrJobProcessingTimeout is returned when a job exceeds its
	// configured processing timeout.
	ErrJobProcessingTimeout = errors.New("job processing exceeded timeout")
	// ErrSerializationFailure is returned when job payload serialization
	// or deserialization fails.
	ErrSerializationFailure = errors.New("failure in serialization process")
	// ErrResultWriterNotSet is returned when [Delivery.WriteResult] is called
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
	// ErrJobResultNotFound is returned when a retained job has no stored result.
	ErrJobResultNotFound = errors.New("job result not found")
	// ErrInvalidContext is returned when a required context is nil.
	ErrInvalidContext = errors.New("context cannot be nil")
)

// Worker errors are returned when worker configuration or lifecycle operations fail.
var (
	// ErrInvalidWorkerQueues is returned when no queues are configured
	// for the worker.
	ErrInvalidWorkerQueues = errors.New("worker requires at least one queue")
	// ErrInvalidWorkerConcurrency is returned when the worker concurrency
	// is set to zero or a negative value.
	ErrInvalidWorkerConcurrency = errors.New("worker requires a positive concurrency value")
	// ErrWorkerAlreadyStarted is returned when [Worker.Run] is called
	// on a worker that is already running.
	ErrWorkerAlreadyStarted = errors.New("worker already started")
	// ErrWorkerStopped is returned when a worker is started after it has
	// been shut down.
	ErrWorkerStopped = errors.New("worker already stopped")
	// ErrHandlerAlreadyRegistered is returned when a handler for the
	// same job type is registered more than once.
	ErrHandlerAlreadyRegistered = errors.New("handler already registered")
	// ErrWorkerNotFound is returned when a worker cannot be found by
	// its ID.
	ErrWorkerNotFound = errors.New("worker not found")
)

// Scheduler errors are returned when scheduler configuration or lifecycle operations fail.
var (
	// ErrSchedulerAlreadyStarted is returned when [Scheduler.Run] is called
	// on a scheduler that is already running.
	ErrSchedulerAlreadyStarted = errors.New("scheduler already started")
	// ErrSchedulerStopped is returned when a scheduler is started after
	// it has been shut down.
	ErrSchedulerStopped = errors.New("scheduler already stopped")
	// ErrInvalidPeriodicInterval is returned when a periodic schedule
	// is registered with a non-positive interval.
	ErrInvalidPeriodicInterval = errors.New("periodic interval must be positive")
	// ErrInvalidSyncInterval is returned when a scheduler sync interval
	// is zero or negative.
	ErrInvalidSyncInterval = errors.New("scheduler sync interval must be positive")
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
	// ErrRedisInvalidDB is returned when the Redis DB number is negative.
	ErrRedisInvalidDB = errors.New("redis db cannot be negative")
	// ErrRedisInvalidPoolSize is returned when Redis pool size is negative.
	ErrRedisInvalidPoolSize = errors.New("redis pool size cannot be negative")
	// ErrRedisInvalidTimeout is returned when a Redis timeout is negative.
	ErrRedisInvalidTimeout = errors.New("redis timeout cannot be negative")
	// ErrRedisClientNotSupported is returned when the Redis client type
	// is not supported by the manager.
	ErrRedisClientNotSupported = errors.New("redis client type not supported")
	// ErrRedisUnavailable is returned when Redis cannot serve an
	// operation because the server or connection is unavailable.
	ErrRedisUnavailable = errors.New("redis unavailable")
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
	// ErrInvalidManagerClient is returned when a manager is constructed
	// without a Redis client.
	ErrInvalidManagerClient = errors.New("manager requires redis client")
	// ErrInvalidManagerInspector is returned when a manager is constructed
	// without an Asynq inspector.
	ErrInvalidManagerInspector = errors.New("manager requires inspector")
	// ErrUnsupportedJobStateForAction is returned when the job state
	// does not support the requested action.
	ErrUnsupportedJobStateForAction = errors.New("unsupported job state for the requested action")
	// ErrQueueNotFound is returned when the specified queue does not exist.
	ErrQueueNotFound = errors.New("queue not found")
	// ErrQueueNotEmpty is returned when attempting to delete a queue
	// that still contains jobs without using force mode.
	ErrQueueNotEmpty = errors.New("queue is not empty")
)

// ErrSkipRetry indicates a condition to skip retries and move the job to the archive.
var ErrSkipRetry = errors.New("skip retry")

// NewSkipRetryError creates and wraps a SkipRetry error with a custom message.
func NewSkipRetryError(reason string) error {
	return fmt.Errorf("skip retry due to: %s: %w", reason, ErrSkipRetry)
}

// ErrRateLimit defines a custom error type for rate limiting scenarios.
//
//nolint:errname // Preserve the existing exported API for rate limit errors.
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
