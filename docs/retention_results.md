# Job Retention and Results

The `queue` library streamlines job lifecycle management, emphasizing retention and result handling for in-depth job analysis and debugging.

## Setting Up Retention

Job retention allows completed jobs to be stored in the queue for a predefined duration. This feature is instrumental in creating detailed audit logs, simplifying the debugging process, and enabling thorough post-execution analysis.

### Global Retention

Define a universal retention duration for all jobs, ensuring a standardized approach across your application.

```go
import (
    "time"
    "github.com/kaptinlin/queue"
)

client, err := queue.NewClient(redisConfig, queue.WithClientRetention(24*time.Hour))
if err != nil {
    log.Fatalf("Error initializing client: %v", err)
}
```

This configuration sets a 24-hour retention period for all jobs, providing a consistent framework for job data retention.

### Specific Job Retention

Customize retention periods for individual jobs based on unique requirements:

```go
job, err := queue.NewJob(
    "job_type",
    payload,
    queue.WithRetention(48*time.Hour),
)
if err != nil {
    log.Fatalf("Error creating job: %v", err)
}

id, err := client.EnqueueJob(job)
if err != nil {
    log.Fatalf("Error enqueueing job: %v", err)
}
```

## Managing Job Results

### Writing Results

Use the `WriteResult` method within job handlers to attach execution details, output data, or metrics directly to runtime deliveries.

```go
func YourJobHandler(ctx context.Context, delivery *queue.Delivery) error {
    result := map[string]interface{}{
        "status":  "success",
        "details": "Job executed successfully.",
    }

    if err := delivery.WriteResult(result); err != nil {
        return err
    }

    return nil
}
```

### Accessing Results

`JobInfo` reports whether a retained result exists and its byte size. Retrieve the raw encoded result explicitly with `JobResult`.

```go
jobInfo, err := manager.JobInfo("queue_name", "jobID")
if err != nil {
    return err
}
if !jobInfo.HasResult {
    return queue.ErrJobResultNotFound
}

result, err := manager.JobResult("queue_name", "jobID")
if err != nil {
    return err
}

fmt.Println("Job Result:", string(result))
```
