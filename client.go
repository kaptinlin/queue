// Package queue provides a simple and flexible job queue implementation for Go applications.
package queue

import (
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Client encapsulates an Asynq client instance with custom error handling and job retention settings.
type Client struct {
	asynqClient  *asynq.Client
	errorHandler ClientErrorHandler
	retention    time.Duration
	logger       Logger
}

// ClientConfig defines the configuration options for the Client.
type ClientConfig struct {
	ErrorHandler ClientErrorHandler // Custom handler for enqueue errors.
	Retention    time.Duration      // Default retention duration for jobs.
	Logger       Logger             // Logger instance for logging.
}

// ClientErrorHandler provides an interface for handling enqueue errors.
type ClientErrorHandler interface {
	HandleError(err error, job *Job)
}

// NewClient initializes a new Client with specified Redis configuration and client options.
func NewClient(redisConfig *RedisConfig, opts ...ClientOption) (*Client, error) {
	if redisConfig == nil {
		return nil, ErrInvalidRedisConfig
	}
	if err := redisConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid redis config: %w", err)
	}

	asynqClient := asynq.NewClient(redisConfig.ToAsynqRedisOpt())

	config := &ClientConfig{
		Logger:    NewDefaultLogger(), // Default to slog-based logger.
		Retention: 0,                  // No retention by default.
		// ErrorHandler is nil by default - no default error handler
	}

	// Apply client options to configure the instance.
	for _, opt := range opts {
		opt(config)
	}

	return &Client{
		asynqClient:  asynqClient,
		errorHandler: config.ErrorHandler, // May be nil
		retention:    config.Retention,
		logger:       config.Logger,
	}, nil
}

// ClientOption defines a function signature for configuring the Client.
type ClientOption func(*ClientConfig)

// WithClientLogger sets a custom logger for the client.
func WithClientLogger(logger Logger) ClientOption {
	return func(c *ClientConfig) {
		c.Logger = logger
	}
}

// WithClientErrorHandler sets a custom error handler for the client.
func WithClientErrorHandler(handler ClientErrorHandler) ClientOption {
	return func(c *ClientConfig) {
		c.ErrorHandler = handler
	}
}

// WithClientRetention sets a default retention duration for jobs.
func WithClientRetention(retention time.Duration) ClientOption {
	return func(c *ClientConfig) {
		c.Retention = retention
	}
}

// Enqueue wraps the process of creating a job and enqueueing it with the Asynq client.
func (c *Client) Enqueue(jobType string, payload any, opts ...JobOption) (string, error) {
	job := NewJob(jobType, payload, opts...)
	return c.EnqueueJob(job)
}

// EnqueueJob adds a job to the queue based on the provided Job instance.
func (c *Client) EnqueueJob(job *Job) (string, error) {
	task, opts, err := job.ConvertToAsynqTask()
	if err != nil {
		c.handleJobError(err, job, "failed to convert job to task")
		return "", err
	}

	// Determine the appropriate retention period
	retention := job.Options.Retention
	if retention <= 0 {
		retention = c.retention
	}

	// Prepare task options with retention if applicable
	if retention > 0 {
		opts = append(opts, asynq.Retention(retention))
	}

	// Enqueue the task with the prepared options
	result, err := c.asynqClient.Enqueue(task, opts...)
	if err != nil {
		c.handleJobError(err, job, "failed to enqueue job")
		return "", fmt.Errorf("%w: %w", ErrEnqueueJob, err)
	}

	return result.ID, nil
}

// handleJobError logs the error and calls the custom error handler if one is registered.
func (c *Client) handleJobError(err error, job *Job, msg string) {
	// Always log
	c.logger.Error(fmt.Sprintf("%s: %v, job_id=%s, job_type=%s, fingerprint=%s",
		msg, err, job.ID, job.Type, job.Fingerprint))

	// Optional: call custom error handler if provided
	if c.errorHandler != nil {
		c.errorHandler.HandleError(err, job)
	}
}

// Stop terminates the Asynq client connection, releasing resources.
func (c *Client) Stop() error {
	return c.asynqClient.Close()
}
