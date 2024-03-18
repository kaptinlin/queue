# Timeouts and Deadlines

The `queue` library enables precise control over job execution times with **Timeouts** and **Deadlines**, optimizing resource use and ensuring timely task completion.

## Timeouts

Define the maximum duration for job execution to prevent indefinite running. If a job exceeds the specified timeout, it's terminated and marked as failed.

### Implementing Timeouts

Use the `WithJobTimeout` option when creating your job handler to set execution limits.

```go
import (
    "context"
    "github.com/kaptinlin/queue"
    "time"
)

func EmailJobHandler(ctx context.Context, job *queue.Job) error {
    // Job logic
    return nil
}

handler := queue.NewHandler(
    "email_job",
    EmailJobHandler,
    queue.WithJobTimeout(30*time.Second), // 30-second timeout
)
```

## Deadlines

Deadlines specify when a job must start by. Jobs not started before their deadline are skipped.

### Setting Deadlines

Use `WithDeadline` when creating a job to enforce start times.

```go
deadline := time.Now().Add(24 * time.Hour) // 24-hour deadline

job := queue.NewJob(
    "report_generation",
    map[string]interface{}{"reportId": 123},
    queue.WithDeadline(&deadline),
)

id, err := client.EnqueueJob(job)
if err != nil {
    log.Fatalf("Job enqueue failed: %v", err)
}
```