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
job := queue.NewJob(
    "job_type",
    payload,
    queue.WithRetention(48*time.Hour),
)

id, err := client.EnqueueJob(job)
if err != nil {
    log.Fatalf("Error enqueueing job: %v", err)
}
```

## Managing Job Results

### Writing Results

Use the `WriteResult` method within job handlers to attach execution details, output data, or metrics directly to jobs.

```go
func YourJobHandler(ctx context.Context, job *queue.Job) error {
    result := map[string]interface{}{
        "status":  "success",
        "details": "Job executed successfully.",
    }

    if err := job.WriteResult(result); err != nil {
        return err
    }

    return nil
}
```

### Accessing Results

Retrieve job results with the `GetJobInfo` method for comprehensive job analysis.

```go
jobInfo, err := manager.GetJobInfo("queue_name", "jobID")
if err != nil {
    // Handle error
}

fmt.Println("Job Result:", jobInfo.Result)
```