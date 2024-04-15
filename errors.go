package queue

import (
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Define package-level error variables with descriptive names.
var (
	ErrNoJobQueueSpecified      = errors.New("job requires a specified queue")
	ErrNoJobTypeSpecified       = errors.New("job requires a specified type")
	ErrJobExceededDeadline      = errors.New("job failed to complete by deadline")
	ErrJobExceededMaxRetries    = errors.New("job exceeded maximum retry attempts")
	ErrInvalidRedisConfig       = errors.New("redis configuration is invalid")
	ErrInvalidWorkerConfig      = errors.New("worker configuration is invalid")
	ErrInvalidWorkerQueues      = errors.New("worker configuration must specify at least one queue")
	ErrInvalidWorkerConcurrency = errors.New("worker configuration must specify a positive concurrency value")
	ErrSerializationFailure     = errors.New("failure in serialization process")
	ErrEnqueueJob               = errors.New("unable to enqueue job")
	ErrScheduledTimeInPast      = errors.New("scheduled time must be in the future")
	ErrJobProcessingTimeout     = errors.New("job processing exceeded timeout")
	ErrTransientIssue           = errors.New("temporary issue detected, job will retry without affecting retry count")
	ErrResultWriterNotSet       = errors.New("result writer is not set for the job")
	ErrFailedToWriteResult      = errors.New("failed to write job result")
	ErrWorkerAlreadyStarted     = errors.New("worker is already started")
	ErrHandlerAlreadyRegistered = errors.New("handler is already registered")
	ErrRedisEmptyAddress        = errors.New("address cannot be empty")
	ErrRedisUnsupportedNetwork  = errors.New("unsupported network type")
	ErrRedisInvalidAddress      = errors.New("invalid address format")
	ErrRedisTLSRequired         = errors.New("TLS config is required for secure Redis connections")
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
