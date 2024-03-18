# Rate Limiting

The `queue` library introduces rate limiting capabilities at both the global worker and individual handler levels. This feature is essential for maintaining manageable system loads and preventing the exhaustion of resources or external API limits.

## Global Rate Limiting

Applies a uniform rate limit across all tasks handled by a worker, facilitating overall system load management.

### Configuring Global Rate Limit

Implement a global rate limit using the `WithWorkerRateLimiter` option during worker setup.

```go
import (
    "github.com/kaptinlin/queue"
    "golang.org/x/time/rate"
)

// Define a global rate limiter: 10 tasks per second, with bursts of up to 5 tasks.
limiter := rate.NewLimiter(10, 5)

// Apply the global rate limiter to the worker.
worker, err := queue.NewWorker(redisConfig, queue.WithWorkerRateLimiter(limiter))
if err != nil {
    log.Fatalf("Worker initialization failed: %v", err)
}
```

## Handler Level Rate Limiting

Enables specific rate limits for distinct task types, providing precision control over task execution rates.

### Setting a Handler's Rate Limit

Define a handler-specific rate limit with the `WithRateLimiter` option.

```go
import (
    "context"
    "github.com/kaptinlin/queue"
    "golang.org/x/time/rate"
    "time"
)

func ProcessEmailJob(ctx context.Context, job *queue.Job) error {
    // Implement task logic here.
}

// Establish a rate limit for the handler: 5 tasks per minute.
limiter := rate.NewLimiter(rate.Every(1*time.Minute), 5)

// Apply the rate limiter to the handler for targeted execution control.
handler := queue.NewHandler("send_email", ProcessEmailJob, queue.WithRateLimiter(limiter))

// Incorporate the handler into the worker configuration.
if err := worker.RegisterHandler(handler); err != nil {
    log.Fatalf("Handler registration failed: %v", err)
}

// Initiate job processing with the defined rate limits.
worker.Start()
```

These configurations allow you to effectively manage the execution rate of tasks, ensuring your application operates efficiently without overwhelming external services or system resources. This approach helps maintain application responsiveness and reliability.