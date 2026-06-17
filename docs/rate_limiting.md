# Rate Limiting

The `queue` library supports in-process rate limiting at both the worker and handler levels. These limiters apply backpressure by waiting on the processing context; they do not mark jobs as failed and they are not distributed across processes.

## Global Rate Limiting

Applies a uniform local rate limit across all tasks handled by one worker process.

### Configuring Global Rate Limit

Implement a global rate limit using the `WithWorkerLocalRateLimiter` option during worker setup.

```go
import (
    "github.com/kaptinlin/queue"
    "golang.org/x/time/rate"
)

// Define a global rate limiter: 10 tasks per second, with bursts of up to 5 tasks.
limiter := rate.NewLimiter(rate.Limit(10), 5)

// Apply the global rate limiter to the worker.
worker, err := queue.NewWorker(redisConfig, queue.WithWorkerLocalRateLimiter(limiter))
if err != nil {
    log.Fatalf("Worker initialization failed: %v", err)
}
```

## Handler Level Rate Limiting

Enables local rate limits for distinct task types.

### Setting a Handler's Rate Limit

Define a handler-specific rate limit with the `WithLocalRateLimiter` option.

```go
import (
    "context"
    "github.com/kaptinlin/queue"
    "golang.org/x/time/rate"
    "time"
)

func ProcessEmailJob(ctx context.Context, delivery *queue.Delivery) error {
    // Implement task logic here.
}

// Establish a rate limit for the handler: 5 tasks per minute.
limiter := rate.NewLimiter(rate.Every(1*time.Minute), 5)

// Apply the rate limiter to the handler for targeted execution control.
handler, err := queue.NewHandler("send_email", ProcessEmailJob, queue.WithLocalRateLimiter(limiter))
if err != nil {
    log.Fatalf("Handler creation failed: %v", err)
}

// Incorporate the handler into the worker configuration.
if err := worker.RegisterHandler(handler); err != nil {
    log.Fatalf("Handler registration failed: %v", err)
}

// Initiate job processing with the host-controlled context.
if err := worker.Run(ctx); err != nil {
    log.Fatalf("Worker stopped with error: %v", err)
}
```

Use a business-level `RateLimitError` when a handler learns from an external service that the job should retry later:

```go
return queue.NewRateLimitError(30 * time.Second)
```

The worker uses `RetryAfter` for retry delay and does not count `RateLimitError` as a job failure. `errors.Is(err, queue.ErrRetryWithoutFailure)` is true for rate-limit errors. Local `rate.Limiter` values are process-local; use a separate shared limiter if multiple worker processes need one global quota.
