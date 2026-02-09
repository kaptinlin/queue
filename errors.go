package queue

import (
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Job errors are returned when job validation or processing fails.
var (
	ErrNoJobQueueSpecified   = errors.New("job requires a specified queue")
	ErrNoJobTypeSpecified    = errors.New("job requires a specified type")
	ErrJobExceededDeadline   = errors.New("job failed to complete by deadline")
	ErrJobExceededMaxRetries = errors.New("job exceeded maximum retry attempts")
	ErrJobProcessingTimeout  = errors.New("job processing exceeded timeout")
	ErrScheduledTimeInPast   = errors.New("scheduled time must be in the future")
	ErrSerializationFailure  = errors.New("failure in serialization process")
	ErrResultWriterNotSet    = errors.New("result writer is not set for the job")
	ErrFailedToWriteResult   = errors.New("failed to write job result")
	ErrEnqueueJob            = errors.New("unable to enqueue job")
	ErrTransientIssue        = errors.New("transient issue, job will retry without affecting retry count")
	ErrInvalidJobState       = errors.New("invalid job state")
	ErrJobNotFound           = errors.New("job not found")
)

// Worker errors are returned when worker configuration or lifecycle operations fail.
var (
	ErrInvalidWorkerConfig      = errors.New("invalid worker configuration")
	ErrInvalidWorkerQueues      = errors.New("worker requires at least one queue")
	ErrInvalidWorkerConcurrency = errors.New("worker requires a positive concurrency value")
	ErrWorkerAlreadyStarted     = errors.New("worker already started")
	ErrHandlerAlreadyRegistered = errors.New("handler already registered")
	ErrWorkerNotFound           = errors.New("worker not found")
)

// Redis errors are returned when Redis configuration validation fails.
var (
	ErrInvalidRedisConfig      = errors.New("invalid redis configuration")
	ErrRedisEmptyAddress       = errors.New("address cannot be empty")
	ErrRedisUnsupportedNetwork = errors.New("unsupported network type")
	ErrRedisInvalidAddress     = errors.New("invalid address format")
	ErrRedisTLSRequired        = errors.New("TLS configuration required for rediss:// connections")
	ErrRedisClientNotSupported = errors.New("redis client type not supported")
)

// Manager errors are returned when queue management operations fail.
var (
	ErrOperationNotSupported        = errors.New("operation not supported for the given job state")
	ErrArchivingActiveJobs          = errors.New("cannot archive active jobs directly, cancel first")
	ErrGroupRequiredForAggregation  = errors.New("group identifier required for aggregating jobs")
	ErrUnsupportedJobStateForAction = errors.New("unsupported job state for the requested action")
	ErrQueueNotFound                = errors.New("queue not found")
	ErrQueueNotEmpty                = errors.New("queue is not empty")
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
	var rateLimitErr *ErrRateLimit
	return errors.As(err, &rateLimitErr)
}
