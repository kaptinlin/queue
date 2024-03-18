package queue

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
)

// HandlerFunc defines a function signature for job processing functions.
type HandlerFunc func(ctx context.Context, job *Job) error

// Handler encapsulates the configuration and logic needed to process jobs of a specific type.
type Handler struct {
	Handle         HandlerFunc
	JobType        string
	JobQueue       string
	JobTimeout     time.Duration
	RetryDelayFunc func(int, error) time.Duration
	Limiter        *rate.Limiter
}

// NewHandler creates a new Handler with the specified job type, processing function, and options.
func NewHandler(jobType string, handle HandlerFunc, opts ...HandlerOption) *Handler {
	h := &Handler{
		JobType: jobType,
		Handle:  handle,
		// Default to the default queue unless overridden by options.
		JobQueue: DefaultQueue,
	}

	// Apply provided options to configure the handler.
	for _, opt := range opts {
		opt(h)
	}

	return h
}

// HandlerOption defines a function signature for configuring a Handler.
type HandlerOption func(*Handler)

// WithRateLimiter configures a rate limiter for the handler to control the rate of job processing.
func WithRateLimiter(limiter *rate.Limiter) HandlerOption {
	return func(h *Handler) {
		h.Limiter = limiter
	}
}

// WithJobTimeout sets a timeout for job processing, terminating the job if it exceeds this duration.
func WithJobTimeout(d time.Duration) HandlerOption {
	return func(h *Handler) {
		h.JobTimeout = d
	}
}

// WithJobQueue specifies the queue that the handler will process jobs from.
func WithJobQueue(queue string) HandlerOption {
	return func(h *Handler) {
		h.JobQueue = queue
	}
}

// WithRetryDelayFunc sets a custom function to determine the delay before retrying a failed job.
func WithRetryDelayFunc(f func(int, error) time.Duration) HandlerOption {
	return func(h *Handler) {
		h.RetryDelayFunc = f
	}
}

// Process executes the handler's job processing function, applying rate limiting and timeouts as configured.
func (h *Handler) Process(ctx context.Context, job *Job) error {
	if h.JobTimeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, h.JobTimeout)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			if h.Limiter != nil && !h.Limiter.Allow() {
				done <- &ErrRateLimit{RetryAfter: 10 * time.Second}
			} else {
				done <- h.Handle(ctx, job)
			}
		}()

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("%w: %v", ErrJobProcessingTimeout, ctx.Err())
			}

			return ctx.Err()
		}
	} else {
		if h.Limiter != nil && !h.Limiter.Allow() {
			return &ErrRateLimit{RetryAfter: 10 * time.Second}
		}
		return h.Handle(ctx, job)
	}
}
