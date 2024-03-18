# Rate Limiting

The `queue` library provides mechanisms to enforce rate limits at both the worker and handler level, ensuring that tasks are processed at a controlled pace. This feature is critical for managing system load and preventing overuse of resources or external APIs.

## Global Rate Limiting (Worker Level)

Global rate limiting applies a single rate limit across all tasks processed by a worker, regardless of their type. This is useful for controlling overall system load.

### Configuring Global Rate Limit

To set a global rate limit for a worker, use the `WithWorkerRateLimiter` option when creating the worker.

```go
import (
    "github.com/kaptinlin/queue"
    "golang.org/x/time/rate"
)

// Create a limiter that allows 10 events per second with a burst size of 5.
limiter := rate.NewLimiter(10, 5)

// Configure worker with the global rate limiter
worker, err := queue.NewWorker(redisConfig, queue.WithWorkerRateLimiter(limiter))
if err != nil {
    log.Fatalf("Failed to create worker: %v", err)
}
```

## Handler Level Rate Limiting

Handler level rate limiting applies rate limits to specific task types, allowing fine-grained control over the processing rate of different tasks.

### Setting Rate Limit for a Handler

Specify a rate limit when creating a handler using the `WithRateLimiter` option.

```go
import (
    "context"
    "github.com/kaptinlin/queue"
    "golang.org/x/time/rate"
    "time"
)

func ProcessEmailJob(ctx context.Context, job *queue.Job) error {
    // Task logic here
}

// Create a limiter for this handler: 5 events per minute.
limiter := rate.NewLimiter(rate.Every(1*time.Minute), 5)

// Bind rate limiter to handler for optimal execution control.
handler := queue.NewHandler("send_email", ProcessEmailJob, queue.WithRateLimiter(limiter))

// Register the handler with the worker.
if err := worker.RegisterHandler(handler); err != nil {
    log.Fatalf("Failed to register handler: %v", err)
}

// Start processing jobs with rate limits in effect
worker.Start()
```

Using these configurations, you can effectively manage task execution rates, ensuring your application remains responsive and avoids overloading external dependencies or services.