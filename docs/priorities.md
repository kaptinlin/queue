# Priority Queues

Priority queues are a powerful tool for managing jobs in your application, enabling the prioritization of important jobs over less critical ones. This approach ensures that high-priority jobs are processed first, improving your application's responsiveness and efficiency in resource utilization. Furthermore, priority queues offer the flexibility to run different jobs on different workers, enhancing scalability and control.

## Setting Up Priority Queues

To leverage priority queues, configure your workers to recognize multiple queues, each associated with a specific priority level. This configuration ensures that jobs in higher priority queues are attended to before those in lower priority queues.

### Worker Configuration for Priority Queues

Here’s how to configure your worker to recognize queues with different priorities:

```go
import "github.com/kaptinlin/queue"

func main() {
    redisConfig := queue.NewRedisConfig(queue.WithRedisAddress("localhost:6379"))

    worker, err := queue.NewWorker(redisConfig,
        queue.WithWorkerQueue("critical", 6), // High priority
        queue.WithWorkerQueue("default", 3),  // Medium priority
        queue.WithWorkerQueue("low", 1),      // Low priority
    )
    if err != nil {
        panic(err)
    }

    defer worker.Stop()
    if err := worker.Start(); err != nil {
        panic(err)
    }
}
```

This setup introduces three priority levels across the `critical`, `default`, and `low` queues, with respective priorities of 6, 3, and 1.

### Job Handling with Priority

To handle jobs with priority, ensure the worker is configured to process jobs from the specific queue. Register job handlers for each queue as follows to prevent `TaskNotFoundError` issues:

```go
err = worker.Register("email:send", handleEmailSendJob, queue.WithJobQueue("critical"))
if err != nil {
    panic(err)
}
```

### Enqueuing Jobs with Priority

When enqueuing jobs, explicitly specify the queue to ensure they are processed according to their priority:

```go
client, err := queue.NewClient(redisConfig)
if err != nil {
    panic(err)
}

jobPayload := map[string]interface{}{"to": "user@example.com"}
job := queue.NewJob("email:send", jobPayload, queue.WithQueue("critical"))

_, err = client.EnqueueJob(job)
if err != nil {
    panic(err)
}
```

This example enqueues an `email:send` job to the `critical` queue, giving it higher processing priority.
