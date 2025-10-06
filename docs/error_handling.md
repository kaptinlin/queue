# Error Handling

Proper error handling ensures the resilience and reliability of job processing systems. This section provides a concise guide on managing errors for both `Client` and `Worker` components.

## Default Error Logging

By default, the `Client` and `Worker` automatically log all errors using the built-in logger. No additional configuration is required for basic error logging:

```go
// Errors are automatically logged with structured logging
redisConfig := queue.NewRedisConfig(
    queue.WithRedisAddress("localhost:6379"),
)
client, _ := queue.NewClient(redisConfig)
worker, _ := queue.NewWorker(redisConfig)

// All enqueue and processing errors are logged automatically
```

## Custom Logger

You can provide your own logger implementation to customize the logging format:

```go
type MyLogger struct{}

func (l *MyLogger) Error(args ...interface{}) {
    // Custom error logging implementation
    log.Println("[ERROR]", fmt.Sprint(args...))
}
// ... implement other Logger interface methods

// Use custom logger
client, _ := queue.NewClient(redisConfig, queue.WithClientLogger(&MyLogger{}))
worker, _ := queue.NewWorker(redisConfig, queue.WithWorkerLogger(&MyLogger{}))
```

## Client Error Handling

For advanced error handling beyond logging (e.g., metrics, alerts, recovery), implement the `ClientErrorHandler` interface:

```go
type MetricsErrorHandler struct {
    metrics *prometheus.Registry
}

func (h *MetricsErrorHandler) HandleError(err error, job *queue.Job) {
    // Record metrics
    h.metrics.WithLabelValues(job.Type, "failed").Inc()

    // Send to error tracking service
    sentry.CaptureError(err)

    // Trigger alerts for critical errors
    if errors.Is(err, queue.ErrRedisConnection) {
        alerts.Send("Redis connection failed!")
    }
}

// Using the custom handler
client, _ := queue.NewClient(redisConfig,
    queue.WithClientErrorHandler(&MetricsErrorHandler{metrics: myMetrics}))
```

**Note:** Errors are always logged by the Client's logger. The custom error handler is called **after** logging for additional business logic.

## Worker Error Handling

For advanced error handling during job processing (e.g., metrics, alerts, recovery), implement the `WorkerErrorHandler` interface:

```go
type MonitoringErrorHandler struct {
    metrics *prometheus.Registry
}

func (h *MonitoringErrorHandler) HandleError(err error, job *queue.Job) {
    // Record processing errors
    h.metrics.WithLabelValues(job.Type, job.Options.Queue, "failed").Inc()

    // Send to error tracking
    sentry.CaptureException(err, &sentry.EventHint{
        Context: map[string]interface{}{
            "job_id":   job.ID,
            "job_type": job.Type,
            "queue":    job.Options.Queue,
        },
    })

    // Trigger alerts for critical job types
    if job.Type == "payment:process" {
        alerts.Critical("Payment processing failed!", err)
    }
}

// Configuring the worker
worker, _ := queue.NewWorker(redisConfig,
    queue.WithWorkerErrorHandler(&MonitoringErrorHandler{metrics: myMetrics}))
```

**Note:** Errors are always logged by the Worker's logger. The custom error handler is called **after** logging for additional business logic.

## Combining Custom Logger and Error Handler

You can combine both custom logger and error handler for complete control:

```go
client, _ := queue.NewClient(redisConfig,
    queue.WithClientLogger(myLogger),           // Custom logging format
    queue.WithClientErrorHandler(myHandler))    // Custom business logic

worker, _ := queue.NewWorker(redisConfig,
    queue.WithWorkerLogger(myLogger),           // Custom logging format
    queue.WithWorkerErrorHandler(myHandler))    // Custom business logic
```

These strategies ensure that both enqueuing and processing errors are handled appropriately, maintaining system performance and stability.