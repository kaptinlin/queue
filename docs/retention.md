# Job Retention

The `queue` library incorporates job retention capabilities, enabling jobs to remain in the queue for a predetermined duration after their successful completion. This feature is instrumental for audit trails, debugging, and post-execution analysis, allowing developers and system administrators to review job outcomes and operational metrics.

## Features

- **Global Job Retention**: Configure a default retention period applicable to all jobs managed by a client, ensuring a uniform approach across your application.
- **Individual Job Retention**: Specify retention periods for individual jobs, granting the flexibility to cater to the unique requirements of each job.

## Configuring Job Retention

### Global Job Retention

Establish a default retention period for all jobs at the client level, simplifying management and ensuring consistency.

```go
import (
    "time"
    "github.com/kaptinlin/queue"
)

client, err := queue.NewClient(redisConfig, queue.WithClientRetention(24*time.Hour))
if err != nil {
    log.Fatalf("Error creating client: %v", err)
}
```

This configuration sets a universal 24-hour retention period for all jobs processed by the client.

### Individual Job Retention

For jobs needing specific retention settings, override the global default by specifying a retention period at the time of job creation.

```go
// Example: Retain a job for 48 hours after completion.
job := queue.NewJob(
    "task_type", 
    payload, 
    queue.WithRetention(48*time.Hour), // Specify the retention period for this job.
)

// Enqueue the job with specified retention.
id, err := client.EnqueueJob(job)
if err != nil {
    log.Fatalf("Error enqueueing job: %v", err)
}
```

This example configures a 48-hour retention period for a particular job, allowing for extended review and analysis.