package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/time/rate"
)

// HandlerFunc defines the signature for job processing functions.
type HandlerFunc func(ctx context.Context, job *Job) error

// HandlerOption defines the signature for configuring a Handler.
type HandlerOption func(*Handler)

// Handler encapsulates the configuration and logic for processing jobs.
type Handler struct {
	originalHandle HandlerFunc // The original job processing function.
	Handle         HandlerFunc // Composed function including middleware.
	JobType        string
	JobQueue       string
	JobTimeout     time.Duration
	RetryDelayFunc func(int, error) time.Duration
	Limiter        *rate.Limiter
	middlewares    []MiddlewareFunc // Middleware functions to apply.
}

// NewHandler initializes a new Handler with the provided job type, processing function, and options.
func NewHandler(jobType string, handle HandlerFunc, opts ...HandlerOption) *Handler {
	h := &Handler{
		originalHandle: handle,
		JobType:        jobType,
		JobQueue:       DefaultQueue, // Default queue unless overridden.
	}
	// Apply configuration options to the handler.
	for _, opt := range opts {
		opt(h)
	}
	// Compose middleware onto the handler's function.
	h.composeMiddleware()

	return h
}

// WithOptions dynamically updates the handler's options.
func (h *Handler) WithOptions(opts ...HandlerOption) {
	for _, opt := range opts {
		opt(h)
	}
}

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

// WithMiddleware returns a HandlerOption that appends provided middlewares to the handler and recomposes the middleware chain.
func WithMiddleware(middlewares ...MiddlewareFunc) HandlerOption {
	return func(h *Handler) {
		h.middlewares = append(h.middlewares, middlewares...)
		h.composeMiddleware()
	}
}

// Use appends provided middlewares to the handler and recomposes the middleware chain.
func (h *Handler) Use(middlewares ...MiddlewareFunc) {
	h.middlewares = append(h.middlewares, middlewares...)
	h.composeMiddleware()
}

// composeMiddleware composes all middleware into a single handler function.
func (h *Handler) composeMiddleware() {
	composed := h.originalHandle
	for i := len(h.middlewares) - 1; i >= 0; i-- {
		composed = h.middlewares[i](composed)
	}
	// Update the handler function to include the composed middleware chain.
	h.Handle = composed
}

// Process executes the handler's job processing function, applying rate limiting and timeouts as configured.
func (h *Handler) Process(ctx context.Context, job *Job) error {
	if h.JobTimeout > 0 {
		return h.processWithTimeout(ctx, job)
	}
	return h.processJob(ctx, job)
}

// processWithTimeout executes the handler's job processing function with a timeout.
func (h *Handler) processWithTimeout(ctx context.Context, job *Job) error {
	ctx, cancel := context.WithTimeout(ctx, h.JobTimeout)
	defer cancel()

	done := make(chan error, DefaultHandlerChannelBuffer)
	go func() {
		done <- h.processJob(ctx, job)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%w: %w", ErrJobProcessingTimeout, ctx.Err())
		}
		return ctx.Err()
	}
}

// processJob executes the handler's job processing function, applying rate limiting if configured.
func (h *Handler) processJob(ctx context.Context, job *Job) error {
	if h.Limiter != nil && !h.Limiter.Allow() {
		return &ErrRateLimit{RetryAfter: DefaultRateLimitRetryAfter}
	}
	return h.Handle(ctx, job)
}
