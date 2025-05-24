package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hibiken/asynq"
	"golang.org/x/time/rate"
)

const DefaultQueue = "default"

// DefaultQueues defines default queue names and their priorities.
var DefaultQueues = map[string]int{DefaultQueue: 1}

// Worker represents a worker that processes tasks using the asynq package.
type Worker struct {
	asynqServer  *asynq.Server
	inspector    *asynq.Inspector
	handlers     map[string]*Handler
	groups       map[string]*Group
	mu           sync.Mutex
	started      atomic.Bool
	errorHandler WorkerErrorHandler
	limiter      *rate.Limiter
	middlewares  []MiddlewareFunc
}

// WorkerErrorHandler defines an interface for handling errors that occur during job processing.
type WorkerErrorHandler interface {
	HandleError(err error, job *Job)
}

// DefaultWorkerErrorHandler is a default implementation of WorkerErrorHandler that logs errors.
type DefaultWorkerErrorHandler struct{}

func (h *DefaultWorkerErrorHandler) HandleError(err error, job *Job) {
	log.Printf("Error processing job: %v, job: %v\n", err, job)
}

// WorkerConfig holds configuration parameters for a worker, including concurrency, queue priorities, and error handling.
type WorkerConfig struct {
	StopTimeout  time.Duration
	Concurrency  int
	Queues       map[string]int
	ErrorHandler WorkerErrorHandler
	Limiter      *rate.Limiter
	Logger       Logger
}

// validate checks if the WorkerConfig's fields are correctly set, returning an error if any field is invalid.
func (wc *WorkerConfig) Validate() error {
	if wc.Concurrency <= 0 {
		return ErrInvalidWorkerConcurrency
	}

	if len(wc.Queues) == 0 {
		return ErrInvalidWorkerQueues
	}

	return nil
}

// NewWorker creates and returns a new Worker based on the given Redis configuration and WorkerConfig options.
func NewWorker(redisConfig *RedisConfig, opts ...WorkerOption) (*Worker, error) {
	// Initial validation of the provided Redis configuration.
	if redisConfig == nil {
		return nil, ErrInvalidRedisConfig
	}
	if err := redisConfig.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRedisConfig, err)
	}

	// Apply default configuration and options.
	config := &WorkerConfig{
		Concurrency:  runtime.NumCPU(),
		ErrorHandler: &DefaultWorkerErrorHandler{},
	}
	for _, opt := range opts {
		opt(config)
	}

	// Apply default queue configuration if none is provided.
	if len(config.Queues) == 0 {
		config.Queues = DefaultQueues
	}

	// Validate the WorkerConfig.
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidWorkerConfig, err)
	}

	// Setup the Worker instance.
	worker := &Worker{
		groups:       make(map[string]*Group),
		handlers:     make(map[string]*Handler),
		errorHandler: config.ErrorHandler,
		limiter:      config.Limiter,
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

	// Create a new group and store it in the worker's groups map.
	group := &Group{name: name, worker: w}
	w.groups[name] = group
	return group
}

// WorkerOption defines a function signature for configuring a Worker.
type WorkerOption func(*WorkerConfig)

// Register allows registering a handler function for a specific job type with additional options.
func (w *Worker) Register(jobType string, handle HandlerFunc, opts ...HandlerOption) error {
	handler := NewHandler(jobType, handle, opts...)

	return w.RegisterHandler(handler)
}

// RegisterHandler registers a task handler for a specific job type.
func (w *Worker) RegisterHandler(handler *Handler) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if handler.JobType == "" {
		return ErrNoJobTypeSpecified
	}

	if handler.JobQueue == "" {
		return ErrNoJobQueueSpecified
	}
	if _, exists := w.handlers[handler.JobType]; exists {
		return fmt.Errorf("%w: %s", ErrHandlerAlreadyRegistered, handler.JobType)
	}

	w.handlers[handler.JobType] = handler
	return nil
}

// Start initiates the worker to process tasks, ensuring it has not already been started.
func (w *Worker) Start() error {
	if !w.started.CompareAndSwap(false, true) {
		return ErrWorkerAlreadyStarted
	}

	mux := asynq.NewServeMux()
	w.setupHandlers(mux)

	return w.asynqServer.Run(mux)
}

// Stop gracefully shuts down the worker server, ensuring atomic update of the started status.
func (w *Worker) Stop() error {
	w.asynqServer.Shutdown()
	w.started.Store(false)

	return nil
}

// setupAsynqServer initializes the Asynq server and inspector based on the provided Redis configuration and worker configuration.
func (w *Worker) setupAsynqServer(redisConfig *RedisConfig, config *WorkerConfig) {
	asynqRedisOpt := redisConfig.ToAsynqRedisOpt()

	w.inspector = asynq.NewInspector(asynqRedisOpt)
	w.asynqServer = asynq.NewServer(asynqRedisOpt, asynq.Config{
		ShutdownTimeout: config.StopTimeout,
		Concurrency:     config.Concurrency,
		Queues:          config.Queues,
		RetryDelayFunc:  w.retryDelayFunc,
		IsFailure:       w.isFailure,
		Logger:          config.Logger,
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

// makeHandlerFunc creates a task handling function for the Asynq server, applying rate limiting and error handling.
func (w *Worker) makeHandlerFunc(handler *Handler) func(ctx context.Context, task *asynq.Task) error {
	finalHandler := handler.Process
	for i := len(w.middlewares) - 1; i >= 0; i-- {
		finalHandler = w.middlewares[i](finalHandler)
	}

	return func(ctx context.Context, task *asynq.Task) error {
		if w.limiter != nil && !w.limiter.Allow() {
			// Global rate limit exceeded
			return &ErrRateLimit{RetryAfter: 10 * time.Second}
		}

		// Extract payload from the task
		var payload map[string]interface{}
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return err
		}

		taskID := task.ResultWriter().TaskID()

		// Reconstructing Job object with task options
		taskInfo, err := w.inspector.GetTaskInfo(handler.JobQueue, taskID)
		if err != nil {
			return err
		}

		job := NewJob(task.Type(), payload,
			WithDelay(time.Until(taskInfo.NextProcessAt)),
			WithMaxRetries(taskInfo.MaxRetry),
			WithQueue(taskInfo.Queue),
			WithDeadline(&taskInfo.Deadline),
			WithScheduleAt(&taskInfo.NextProcessAt),
		)
		job.SetID(taskID).SetResultWriter(task.ResultWriter())

		// Process the job with the reconstructed Job object
		if err := finalHandler(ctx, job); err != nil {
			w.errorHandler.HandleError(err, job)
			return err
		}

		return nil
	}
}

// retryDelayFunc determines the delay before retrying a task after failure, using custom logic or falling back to Asynq's default.
func (w *Worker) retryDelayFunc(count int, err error, task *asynq.Task) time.Duration {
	var rateLimitErr *ErrRateLimit
	if errors.As(err, &rateLimitErr) {
		return rateLimitErr.RetryAfter
	}

	if handler, exists := w.handlers[task.Type()]; exists && handler.RetryDelayFunc != nil {
		return handler.RetryDelayFunc(count, err)
	}

	return asynq.DefaultRetryDelayFunc(count, err, task)
}

// isFailure determines whether a task failure should be considered final, based on custom logic.
func (w *Worker) isFailure(err error) bool {
	return !IsErrRateLimit(err) && !errors.Is(err, ErrTransientIssue)
}

// WorkerOption implementations for configuring various aspects of the Worker.

// WithWorkerStopTimeout configures the stop timeout for the worker.
func WithWorkerStopTimeout(timeout time.Duration) WorkerOption {
	return func(c *WorkerConfig) {
		c.StopTimeout = timeout
	}
}

// WithWorkerRateLimiter configures a global rate limiter for the worker.
func WithWorkerRateLimiter(limiter *rate.Limiter) WorkerOption {
	return func(c *WorkerConfig) {
		c.Limiter = limiter
	}
}

// WithWorkerConcurrency sets the number of concurrent workers.
func WithWorkerConcurrency(concurrency int) WorkerOption {
	return func(c *WorkerConfig) {
		c.Concurrency = concurrency
	}
}

// WithWorkerQueue adds a queue with the specified priority to the worker configuration.
func WithWorkerQueue(name string, priority int) WorkerOption {
	return func(c *WorkerConfig) {
		if c.Queues == nil {
			c.Queues = make(map[string]int)
		}
		c.Queues[name] = priority
	}
}

// WithWorkerQueues sets the queue names and their priorities for the worker.
func WithWorkerQueues(queues map[string]int) WorkerOption {
	return func(c *WorkerConfig) {
		if queues != nil {
			c.Queues = queues
		}
	}
}

// WithWorkerErrorHandler configures the error handler for the worker.
func WithWorkerErrorHandler(handler WorkerErrorHandler) WorkerOption {
	return func(c *WorkerConfig) {
		c.ErrorHandler = handler
	}
}

// WithWorkerLogger configures a custom logger for the worker.
func WithWorkerLogger(logger Logger) WorkerOption {
	return func(c *WorkerConfig) {
		c.Logger = logger
	}
}
