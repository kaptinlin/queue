// Package queue provides a simple and flexible job queue implementation for Go applications.
package queue

import (
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Client encapsulates an asynq.Client instance with custom error handling and job retention settings.
type Client struct {
	asynqClient  *asynq.Client
	errorHandler ClientErrorHandler
	retention    time.Duration
	logger       Logger
}

// clientConfig defines the configuration options for the Client.
type clientConfig struct {
	ErrorHandler ClientErrorHandler // Custom handler for enqueue errors.
	Retention    time.Duration      // Default retention duration for jobs.
	Logger       Logger
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

	config := &clientConfig{
		Logger:    NewDefaultLogger(),
		Retention: 0,
	}

	for _, opt := range opts {
		opt.applyClientOption(config)
	}
	if config.Logger == nil {
		config.Logger = NewDefaultLogger()
	}

	return &Client{
		asynqClient:  asynqClient,
		errorHandler: config.ErrorHandler,
		retention:    config.Retention,
		logger:       config.Logger,
	}, nil
}

// ClientOption configures a Client.
type ClientOption interface {
	applyClientOption(*clientConfig)
}

type clientOption func(*clientConfig)

func (f clientOption) applyClientOption(config *clientConfig) {
	f(config)
}

// WithClientLogger sets a custom logger for the client.
func WithClientLogger(logger Logger) ClientOption {
	return clientOption(func(c *clientConfig) {
		c.Logger = logger
	})
}

// WithClientErrorHandler sets a custom error handler for the client.
func WithClientErrorHandler(handler ClientErrorHandler) ClientOption {
	return clientOption(func(c *clientConfig) {
		c.ErrorHandler = handler
	})
}

// WithClientRetention sets a default retention duration for jobs.
func WithClientRetention(retention time.Duration) ClientOption {
	return clientOption(func(c *clientConfig) {
		c.Retention = retention
	})
}

// Enqueue wraps the process of creating a job and enqueueing it with the Asynq client.
func (c *Client) Enqueue(jobType string, payload any, opts ...JobOption) (string, error) {
	job, err := NewJob(jobType, payload, opts...)
	if err != nil {
		c.handleJobError(err, nil, "failed to create job")
		return "", err
	}
	return c.EnqueueJob(job)
}

// EnqueueJob adds a job to the queue.
func (c *Client) EnqueueJob(job *Job) (string, error) {
	task, opts, err := job.ConvertToAsynqTask()
	if err != nil {
		c.handleJobError(err, job, "failed to convert job to task")
		return "", err
	}

	retention := job.Options().Retention
	if retention <= 0 {
		retention = c.retention
	}
	if retention > 0 {
		opts = append(opts, asynq.Retention(retention))
	}

	result, err := c.asynqClient.Enqueue(task, opts...)
	if err != nil {
		c.handleJobError(err, job, "failed to enqueue job")
		return "", fmt.Errorf("%w: %w", ErrEnqueueJob, err)
	}

	return result.ID, nil
}

// handleJobError logs the error and calls the custom error handler if one is registered.
func (c *Client) handleJobError(err error, job *Job, msg string) {
	fields := []any{"error", err}
	if job != nil {
		fields = append(fields,
			"job_type", job.Type(),
			"content_digest", job.ContentDigest(),
		)
	}
	c.logger.Error(append([]any{msg}, fields...)...)

	if c.errorHandler != nil {
		c.errorHandler.HandleError(err, job)
	}
}

// Close closes the client connection and releases resources.
func (c *Client) Close() error {
	return c.asynqClient.Close()
}
