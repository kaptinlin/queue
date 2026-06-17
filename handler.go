package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/time/rate"
)

// HandlerFunc defines the signature for runtime job delivery processing.
type HandlerFunc func(ctx context.Context, delivery *Delivery) error

// HandlerOption configures a Handler.
type HandlerOption interface {
	applyHandlerOption(*Handler)
}

type handlerOption func(*Handler)

func (f handlerOption) applyHandlerOption(handler *Handler) {
	f(handler)
}

// Handler encapsulates the configuration and logic for processing jobs.
type Handler struct {
	originalHandle HandlerFunc
	handle         HandlerFunc
	jobType        string
	jobQueue       string
	jobTimeout     time.Duration
	retryDelayFunc func(int, error) time.Duration
	limiter        *rate.Limiter
	middlewares    []MiddlewareFunc
}

// NewHandler initializes a new Handler with the provided job type, processing function, and options.
func NewHandler(jobType string, handle HandlerFunc, opts ...HandlerOption) (*Handler, error) {
	if jobType == "" {
		return nil, ErrNoJobTypeSpecified
	}
	if handle == nil {
		return nil, ErrNoHandlerFuncSpecified
	}

	h := &Handler{
		originalHandle: handle,
		jobType:        jobType,
		jobQueue:       DefaultQueue,
	}
	for _, opt := range opts {
		opt.applyHandlerOption(h)
	}
	if err := h.validate(); err != nil {
		return nil, err
	}
	h.composeMiddleware()

	return h, nil
}

// Type returns the job type handled by h.
func (h *Handler) Type() string {
	if h == nil {
		return ""
	}
	return h.jobType
}

// Queue returns the queue handled by h.
func (h *Handler) Queue() string {
	if h == nil {
		return ""
	}
	return h.jobQueue
}

// Timeout returns the processing timeout configured for h.
func (h *Handler) Timeout() time.Duration {
	if h == nil {
		return 0
	}
	return h.jobTimeout
}

func (h *Handler) validate() error {
	if h.jobType == "" {
		return ErrNoJobTypeSpecified
	}
	if h.originalHandle == nil {
		return ErrNoHandlerFuncSpecified
	}
	if h.jobQueue == "" {
		return ErrNoJobQueueSpecified
	}
	if h.jobTimeout < 0 {
		return fmt.Errorf("%w: timeout cannot be negative", ErrInvalidJobOptions)
	}
	for _, middleware := range h.middlewares {
		if middleware == nil {
			return ErrInvalidMiddleware
		}
	}
	return nil
}

// WithLocalRateLimiter configures in-process backpressure before handler execution.
func WithLocalRateLimiter(limiter *rate.Limiter) HandlerOption {
	return handlerOption(func(h *Handler) {
		h.limiter = limiter
	})
}

// WithJobTimeout sets the processing context deadline for a handler attempt.
func WithJobTimeout(d time.Duration) HandlerOption {
	return handlerOption(func(h *Handler) {
		h.jobTimeout = d
	})
}

// WithJobQueue specifies the queue that the handler will process jobs from.
func WithJobQueue(queue string) HandlerOption {
	return handlerOption(func(h *Handler) {
		h.jobQueue = queue
	})
}

// WithRetryDelayFunc sets a custom function to determine the delay before retrying a failed job.
func WithRetryDelayFunc(f func(int, error) time.Duration) HandlerOption {
	return handlerOption(func(h *Handler) {
		h.retryDelayFunc = f
	})
}

// WithMiddleware returns a HandlerOption that appends provided middlewares to the handler and recomposes the middleware chain.
func WithMiddleware(middlewares ...MiddlewareFunc) HandlerOption {
	return handlerOption(func(h *Handler) {
		h.middlewares = append(h.middlewares, middlewares...)
	})
}

// composeMiddleware composes all middleware into a single handler function.
func (h *Handler) composeMiddleware() {
	composed := h.originalHandle
	for i := range len(h.middlewares) {
		composed = h.middlewares[len(h.middlewares)-1-i](composed)
	}
	h.handle = composed
}

// Process executes the handler's job processing function with rate limiting and optional timeout.
func (h *Handler) Process(ctx context.Context, delivery *Delivery) error {
	if h == nil {
		return ErrInvalidHandler
	}
	if h.jobTimeout > 0 {
		return h.processWithTimeout(ctx, delivery)
	}
	return h.processJob(ctx, delivery)
}

// processWithTimeout runs the handler with a bounded context.
func (h *Handler) processWithTimeout(ctx context.Context, delivery *Delivery) error {
	ctx, cancel := context.WithTimeout(ctx, h.jobTimeout)
	defer cancel()

	err := h.processJob(ctx, delivery)
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %w", ErrJobProcessingTimeout, err)
	}
	return err
}

// processJob executes the handler's job processing function, applying rate limiting if configured.
func (h *Handler) processJob(ctx context.Context, delivery *Delivery) error {
	if h.limiter != nil {
		if err := h.limiter.Wait(ctx); err != nil {
			return err
		}
	}
	return h.handle(ctx, delivery)
}
