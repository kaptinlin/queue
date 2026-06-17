# Timeouts and Deadlines

The `queue` library offers mechanisms for precise control over job execution times, enhancing resource efficiency and ensuring tasks complete within expected timeframes.

## Timeouts

Timeouts set a deadline on the handler context for one processing attempt. The library cannot forcibly stop arbitrary Go code; handlers must observe `ctx.Done()` and return. When a handler returns `context.DeadlineExceeded`, the error wraps `ErrJobProcessingTimeout`.

### Implementing Timeouts

To set a processing deadline, use the `WithJobTimeout` option when defining your job handler.

```go
import (
    "context"
    "github.com/kaptinlin/queue"
    "time"
)

func EmailJobHandler(ctx context.Context, delivery *queue.Delivery) error {
    // Define job logic here
    return nil
}

// Create a handler for email jobs with a 30-second processing context deadline.
handler, err := queue.NewHandler(
    "email_job",
    EmailJobHandler,
    queue.WithJobTimeout(30*time.Second),
)
if err != nil {
    log.Fatalf("Failed to create handler: %v", err)
}
```

## Deadlines

Deadlines determine the latest time a job can start. Jobs not initiated before their deadline are not processed.

### Setting Deadlines

Apply a deadline to a job using the `WithDeadline` option during job creation to enforce its start time.

```go
deadline := time.Now().Add(24 * time.Hour) // Set a 24-hour deadline

// Create a job with a specific deadline
job, err := queue.NewJob(
    "report_generation",
    map[string]interface{}{"reportId": 123},
    queue.WithDeadline(&deadline),
)
if err != nil {
    log.Fatalf("Failed to create job: %v", err)
}

// Enqueue the job, handling any errors
id, err := client.EnqueueJob(job)
if err != nil {
    log.Fatalf("Failed to enqueue job: %v", err)
}
```
