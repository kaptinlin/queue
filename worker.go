package queue

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hibiken/asynq"
	"golang.org/x/time/rate"
)

const (
	// DefaultQueue is the default queue name used when no queue is specified.
	DefaultQueue = "default"
	// DefaultRateLimitRetryAfter is the default duration to wait before
	// retrying a rate-limited job.
	DefaultRateLimitRetryAfter = 10 * time.Second
	// DefaultHandlerChannelBuffer is the default buffer size for the
	// handler's internal done channel used in timeout processing.
	DefaultHandlerChannelBuffer = 1
)

func defaultQueues() map[string]int {
	return map[string]int{DefaultQueue: 1}
}

// Worker processes tasks from queues.
type Worker struct {
	asynqServer  *asynq.Server
	handlers     map[string]*Handler
	groups       map[string]*Group
	mu           sync.Mutex
	started      atomic.Bool
	stopped      atomic.Bool
	errorHandler WorkerErrorHandler
	limiter      *rate.Limiter
	middlewares  []MiddlewareFunc
	logger       Logger
}

// WorkerErrorHandler defines an interface for handling errors that occur during job processing.
type WorkerErrorHandler interface {
	HandleError(err error, delivery *Delivery)
}

// workerConfig holds configuration parameters for a worker, including concurrency, queue priorities, and error handling.
type workerConfig struct {
	StopTimeout  time.Duration
	Concurrency  int
	Queues       map[string]int
	ErrorHandler WorkerErrorHandler
	Limiter      *rate.Limiter
	Logger       Logger
}

// validate checks whether the worker configuration is usable.
func (wc *workerConfig) validate() error {
	if wc.Concurrency <= 0 {
		return ErrInvalidWorkerConcurrency
	}

	if len(wc.Queues) == 0 {
		return ErrInvalidWorkerQueues
	}
	for name, priority := range wc.Queues {
		if name == "" || priority <= 0 {
			return ErrInvalidWorkerQueues
		}
	}

	return nil
}

// NewWorker creates and returns a new Worker based on the given Redis configuration and options.
func NewWorker(redisConfig *RedisConfig, opts ...WorkerOption) (*Worker, error) {
	if redisConfig == nil {
		return nil, ErrInvalidRedisConfig
	}
	if err := redisConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid redis config: %w", err)
	}

	config := &workerConfig{
		Concurrency: max(1, runtime.NumCPU()),
		Logger:      NewDefaultLogger(),
	}
	for _, opt := range opts {
		opt.applyWorkerOption(config)
	}
	if config.Logger == nil {
		config.Logger = NewDefaultLogger()
	}

	if len(config.Queues) == 0 {
		config.Queues = defaultQueues()
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid worker config: %w", err)
	}

	worker := &Worker{
		groups:       make(map[string]*Group),
		handlers:     make(map[string]*Handler),
		errorHandler: config.ErrorHandler,
		limiter:      config.Limiter,
		logger:       config.Logger,
	}
	worker.setupAsynqServer(redisConfig, config)

	return worker, nil
}

// Use adds a global middleware to the worker.
func (w *Worker) Use(middleware MiddlewareFunc) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.middlewares = append(w.middlewares, middleware)
}

// Group retrieves an existing group by name or creates a new one if it doesn't exist.
func (w *Worker) Group(name string) *Group {
	w.mu.Lock()
	defer w.mu.Unlock()

	if group, exists := w.groups[name]; exists {
		return group
	}

	group := &Group{worker: w}
	w.groups[name] = group
	return group
}

// WorkerOption configures a Worker.
type WorkerOption interface {
	applyWorkerOption(*workerConfig)
}

type workerOption func(*workerConfig)

func (f workerOption) applyWorkerOption(config *workerConfig) {
	f(config)
}

// Register allows registering a handler function for a specific job type with additional options.
func (w *Worker) Register(jobType string, handle HandlerFunc, opts ...HandlerOption) error {
	handler, err := NewHandler(jobType, handle, opts...)
	if err != nil {
		return err
	}

	return w.RegisterHandler(handler)
}

// RegisterHandler registers a task handler for a specific job type.
func (w *Worker) RegisterHandler(handler *Handler) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if handler == nil {
		return ErrInvalidHandler
	}
	if err := handler.validate(); err != nil {
		return err
	}
	if _, exists := w.handlers[handler.jobType]; exists {
		return fmt.Errorf("%w: %s", ErrHandlerAlreadyRegistered, handler.jobType)
	}

	w.handlers[handler.jobType] = handler
	return nil
}

// Run starts the worker and blocks until ctx is canceled.
func (w *Worker) Run(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalidContext
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if w.stopped.Load() {
		return ErrWorkerStopped
	}
	if !w.started.CompareAndSwap(false, true) {
		return ErrWorkerAlreadyStarted
	}

	mux := asynq.NewServeMux()
	w.setupHandlers(mux)

	if err := w.asynqServer.Start(mux); err != nil {
		w.started.Store(false)
		return err
	}

	<-ctx.Done()
	if w.started.CompareAndSwap(true, false) {
		w.shutdown()
	}

	return nil
}

func (w *Worker) shutdown() {
	w.asynqServer.Shutdown()
	w.stopped.Store(true)
}

// setupAsynqServer initializes the asynq server.
func (w *Worker) setupAsynqServer(redisConfig *RedisConfig, config *workerConfig) {
	asynqRedisOpt := redisConfig.ToAsynqRedisOpt()

	w.asynqServer = asynq.NewServer(asynqRedisOpt, asynq.Config{
		ShutdownTimeout: config.StopTimeout,
		Concurrency:     config.Concurrency,
		Queues:          config.Queues,
		RetryDelayFunc:  w.retryDelayFunc,
		IsFailure:       w.isFailure,
		Logger:          newAsynqLogger(config.Logger),
	})
}

// setupHandlers configures the Asynq ServeMux with registered handlers, applying rate limiting and task preprocessing.
func (w *Worker) setupHandlers(mux *asynq.ServeMux) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for jobType, handler := range w.handlers {
		mux.HandleFunc(jobType, w.makeHandlerFunc(handler))
	}
}

// makeHandlerFunc creates a task handling function, applying rate limiting and error handling.
func (w *Worker) makeHandlerFunc(handler *Handler) func(ctx context.Context, task *asynq.Task) error {
	finalHandler := handler.Process
	for i := range len(w.middlewares) {
		finalHandler = w.middlewares[len(w.middlewares)-1-i](finalHandler)
	}

	return func(ctx context.Context, task *asynq.Task) error {
		if w.limiter != nil && !w.limiter.Allow() {
			return &ErrRateLimit{RetryAfter: DefaultRateLimitRetryAfter}
		}

		delivery, err := newDeliveryFromTask(ctx, task, handler.jobQueue)
		if err != nil {
			return err
		}

		if err := finalHandler(ctx, delivery); err != nil {
			w.logger.Error("failed to process job",
				"error", err,
				"job_id", delivery.ID(),
				"job_type", delivery.Type(),
				"queue", delivery.Queue(),
				"attempt", delivery.Attempt(),
				"retry_count", delivery.RetryCount(),
			)

			if w.errorHandler != nil {
				w.errorHandler.HandleError(err, delivery)
			}

			return asynqHandlerError(err)
		}

		return nil
	}
}

func asynqHandlerError(err error) error {
	if err == nil || errors.Is(err, asynq.SkipRetry) || !errors.Is(err, ErrSkipRetry) {
		return err
	}
	return fmt.Errorf("%w: %w", err, asynq.SkipRetry)
}

// retryDelayFunc determines the delay before retrying a task after failure.
func (w *Worker) retryDelayFunc(count int, err error, task *asynq.Task) time.Duration {
	if rateLimitErr, ok := errors.AsType[*ErrRateLimit](err); ok {
		return rateLimitErr.RetryAfter
	}

	if handler, exists := w.handlers[task.Type()]; exists && handler.retryDelayFunc != nil {
		return handler.retryDelayFunc(count, err)
	}

	return asynq.DefaultRetryDelayFunc(count, err, task)
}

// isFailure determines whether a task failure should be considered final, based on custom logic.
func (w *Worker) isFailure(err error) bool {
	return !IsErrRateLimit(err) && !errors.Is(err, ErrTransientIssue)
}

// WorkerOption implementations for configuring various aspects of the Worker.

// WithWorkerLogger sets a custom logger for the worker.
func WithWorkerLogger(logger Logger) WorkerOption {
	return workerOption(func(c *workerConfig) {
		c.Logger = logger
	})
}

// WithWorkerStopTimeout configures the stop timeout for the worker.
func WithWorkerStopTimeout(timeout time.Duration) WorkerOption {
	return workerOption(func(c *workerConfig) {
		c.StopTimeout = timeout
	})
}

// WithWorkerRateLimiter configures a global rate limiter for the worker.
func WithWorkerRateLimiter(limiter *rate.Limiter) WorkerOption {
	return workerOption(func(c *workerConfig) {
		c.Limiter = limiter
	})
}

// WithWorkerConcurrency sets the number of concurrent workers.
func WithWorkerConcurrency(concurrency int) WorkerOption {
	return workerOption(func(c *workerConfig) {
		c.Concurrency = concurrency
	})
}

// WithWorkerQueue adds a queue with the specified priority to the worker configuration.
func WithWorkerQueue(name string, priority int) WorkerOption {
	return workerOption(func(c *workerConfig) {
		if c.Queues == nil {
			c.Queues = make(map[string]int)
		}
		c.Queues[name] = priority
	})
}

// WithWorkerQueues sets the queue names and their priorities for the worker.
func WithWorkerQueues(queues map[string]int) WorkerOption {
	return workerOption(func(c *workerConfig) {
		if queues != nil {
			c.Queues = make(map[string]int, len(queues))
			for name, priority := range queues {
				c.Queues[name] = priority
			}
		}
	})
}

// WithWorkerErrorHandler configures the error handler for the worker.
func WithWorkerErrorHandler(handler WorkerErrorHandler) WorkerOption {
	return workerOption(func(c *workerConfig) {
		c.ErrorHandler = handler
	})
}
