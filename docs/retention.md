# Job Retention

The `queue` library supports job retention, allowing completed jobs to be stored in the queue for a predefined duration. This functionality is crucial for creating audit trails, facilitating debugging, and conducting post-execution analysis. It provides a means for developers and system administrators to examine job results and performance metrics.

## Global Job Retention

Set a default retention period for all jobs managed by a client. This ensures a consistent retention policy across your application, simplifying management.

### Configuring Global Job Retention

To define a universal retention period for all jobs:

```go
import (
    "time"
    "github.com/kaptinlin/queue"
)

// Initialize a client with a 24-hour job retention period.
client, err := queue.NewClient(redisConfig, queue.WithClientRetention(24*time.Hour))
if err != nil {
    log.Fatalf("Client initialization error: %v", err)
}
```

With this setup, every job processed by the client will be retained for 24 hours after completion, ensuring uniform job retention behavior.

## Individual Job Retention

Override the global retention setting for specific jobs, allowing for tailored retention periods based on individual job requirements.

### Setting Retention for Individual Jobs

For jobs that require a distinct retention period:

```go
// Retain a specific job for 48 hours post-completion.
job := queue.NewJob(
    "job_type",
    payload,
    queue.WithRetention(48*time.Hour), // Define the job-specific retention period.
)

// Enqueue the job with the custom retention setting.
id, err := client.EnqueueJob(job)
if err != nil {
    log.Fatalf("Job enqueue error: %v", err)
}
```

This configuration demonstrates how to assign a 48-hour retention period to a particular job, providing flexibility for detailed analysis and review.
