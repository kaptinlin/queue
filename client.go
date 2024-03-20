package queue

import (
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

// Client encapsulates an Asynq client instance with custom error handling and job retention settings.
type Client struct {
	asynqClient  *asynq.Client
	errorHandler ClientErrorHandler
	retention    time.Duration
}

// ClientConfig defines the configuration options for the Client.
type ClientConfig struct {
	ErrorHandler ClientErrorHandler // Custom handler for enqueue errors.
	Retention    time.Duration      // Default retention duration for jobs.
}

// ClientErrorHandler provides an interface for handling enqueue errors.
type ClientErrorHandler interface {
	HandleError(err error, job *Job)
}

// DefaultClientErrorHandler logs errors encountered during job enqueue operations.
type DefaultClientErrorHandler struct{}

func (h *DefaultClientErrorHandler) HandleError(err error, job *Job) {
	log.Printf("Error enqueuing job: %v, job: %v\n", err, job)
}

// NewClient initializes a new Client with specified Redis configuration and client options.
func NewClient(redisConfig *RedisConfig, opts ...ClientOption) (*Client, error) {
	if redisConfig == nil {
		return nil, ErrInvalidRedisConfig
	}
	if err := redisConfig.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRedisConfig, err)
	}

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Network:      redisConfig.Network,
		Addr:         redisConfig.Addr,
		Username:     redisConfig.Username,
		Password:     redisConfig.Password,
		DB:           redisConfig.DB,
		DialTimeout:  redisConfig.DialTimeout,
		ReadTimeout:  redisConfig.ReadTimeout,
		WriteTimeout: redisConfig.WriteTimeout,
		PoolSize:     redisConfig.PoolSize,
		TLSConfig:    redisConfig.TLSConfig,
	})

	config := &ClientConfig{
		ErrorHandler: &DefaultClientErrorHandler{}, // Default error handler.
		Retention:    0,                            // No retention by default.
	}

	// Apply client options to configure the instance.
	for _, opt := range opts {
		opt(config)
	}

	return &Client{
		asynqClient:  asynqClient,
		errorHandler: config.ErrorHandler,
		retention:    config.Retention,
	}, nil
}

// ClientOption defines a function signature for configuring the Client.
type ClientOption func(*ClientConfig)

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
func (c *Client) Enqueue(jobType string, payload interface{}, opts ...JobOption) (string, error) {
	job := NewJob(jobType, payload, opts...)
	return c.EnqueueJob(job)
}

// EnqueueJob adds a job to the queue based on the provided Job instance.
func (c *Client) EnqueueJob(job *Job) (string, error) {
	task, err := job.ConvertToAsynqTask()
	if err != nil {
		c.errorHandler.HandleError(err, job)
		return "", err
	}

	// Determine the appropriate retention period
	retention := job.Options.Retention
	if retention <= 0 {
		retention = c.retention
	}

	// Prepare task options with retention if applicable
	var taskOpts []asynq.Option
	if retention > 0 {
		taskOpts = append(taskOpts, asynq.Retention(retention))
	}
	// taskOpts = append(taskOpts, asynq.TaskID(job.Fingerprint))

	// Enqueue the task with the prepared options
	result, err := c.asynqClient.Enqueue(task, taskOpts...)
	if err != nil {
		c.errorHandler.HandleError(err, job)
		return "", fmt.Errorf("%w: %v", ErrEnqueueJob, err)
	}

	return result.ID, nil
}

// Stop terminates the Asynq client connection, releasing resources.
func (c *Client) Stop() error {
	return c.asynqClient.Close()
}
