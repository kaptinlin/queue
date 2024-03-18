# Timeouts and Deadlines

The `queue` library utilizes **Timeouts** and **Deadlines** to manage task execution effectively. Timeouts prevent tasks from running indefinitely, and Deadlines ensure tasks are executed within a specific timeframe or discarded.

## Timeouts

A timeout limits how long a task can run. If a task's execution time exceeds this limit, it is terminated and considered failed.

### Setting a Timeout

To enforce a maximum execution time on a task, specify a timeout when creating your handler.

```go
import (
    "context"
    "github.com/kaptinlin/queue"
    "time"
)

func ProcessEmailJob(ctx context.Context, job *queue.Job) error {
    // Insert task logic here
}

handler := queue.NewHandler("send_email", ProcessEmailJob, queue.WithJobTimeout(30*time.Second))
```

### Processing Jobs with Timeouts

Ensure your tasks adhere to the specified timeouts by registering the handler and starting the worker.

```go
ctx := context.Background()

// Register handler and start processing
if err := worker.RegisterHandler(handler); err != nil {
    log.Fatalf("Handler registration failed: %v", err)
}
worker.Start()
```

## Deadlines

Deadlines specify the latest time by which a task should start. Tasks not started by their deadline are skipped.

### Creating Jobs with a Deadline

When creating a job, you can set a deadline to ensure it does not start after a certain point in time.

```go
import (
    "time"
    "github.com/kaptinlin/queue"
)

deadline := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
jobOptions := queue.JobOptions{
    Deadline: deadline,
}
job := queue.NewJob("send_email", map[string]interface{}{"email": "example@example.com"}, jobOptions)
```

### Enqueueing Jobs with Deadlines

After setting a deadline for your job, enqueue it. The system will automatically discard the job if it's past the deadline when attempting to start.

```go
ctx := context.Background()

// Enqueue job with deadline consideration
client.Enqueue(ctx, job)
```
