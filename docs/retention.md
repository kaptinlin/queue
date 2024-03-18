# Task Retention

The `queue` library includes functionality to retain completed tasks in the queue for inspection or debugging purposes. This feature allows you to specify a retention period for tasks, during which they remain accessible even after successful execution.

## Global Task Retention (Client Level)

Global task retention sets a default retention period for all tasks managed by a client. This setting is useful for maintaining a consistent retention policy across your application.

### Configuring Global Task Retention

To configure a global retention period for all tasks enqueued by a client, use the `WithClientRetention` option when creating the client.

```go
import (
    "github.com/kaptinlin/queue"
    "time"
)

// Specify a default retention period of 24 hours for all tasks.
client, err := queue.NewClient(redisConfig, queue.WithClientRetention(24*time.Hour))
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
```

## Task-Level Retention

Task-level retention allows specifying a retention period for individual tasks, offering flexibility for tasks with different inspection or debugging needs.

### Setting Retention for a Task

When enqueuing a task, you can override the global retention setting by specifying a retention period for that task.

```go
import (
    "github.com/kaptinlin/queue"
    "time"
)

// Create a job with specific options.
options := queue.JobOptions{
    // Other options...
    Retention: 48 * time.Hour, // Retain this job for 48 hours after completion.
}
job := queue.NewJob("my_task_type", payload, options)

// Enqueue the job with task-level retention.
id, err := client.EnqueueJob(job)
if err != nil {
    log.Fatalf("Failed to enqueue job: %v", err)
}
```

Using these settings, you can control how long completed tasks are retained in the queue, aiding in post-execution analysis and troubleshooting.