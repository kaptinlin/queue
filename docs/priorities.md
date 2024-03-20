# Priority Queues

Priority queues are essential for managing background tasks efficiently, allowing critical tasks to be processed before less urgent ones. This system enhances responsiveness and resource allocation in applications.

## Configuring Queue Priorities

To implement queue priorities, configure your worker to recognize multiple queues, each with a distinct priority level. Higher priority queues get more processing attention than their lower priority counterparts.

### Worker Configuration

Assign priorities across different queues by setting up your worker as follows:

```go
import (
    "github.com/kaptinlin/queue"
)

func main() {
    // Initialize Redis configuration
    redisConfig := queue.NewRedisConfig(queue.WithRedisAddress("localhost:6379"))

    // Configure the worker with queues of varying priorities
    worker, err := queue.NewWorker(redisConfig,
        queue.WithWorkerQueue("critical", 6), // High priority
        queue.WithWorkerQueue("default", 3),  // Medium priority
        queue.WithWorkerQueue("low", 1),      // Low priority
    )
    if err != nil {
        panic(err)
    }

    // Start and ensure graceful shutdown of the worker
    defer worker.Stop()
    if err := worker.Start(); err != nil {
        panic(err)
    }
}
```

This configuration establishes three queues—`critical`, `default`, and `low`—with priority levels 6, 3, and 1, respectively.

### Enqueuing Tasks with Priority

To enqueue tasks with priority, specify the queue during job creation:

```go
client, err := queue.NewClient(redisConfig)
if err != nil {
    panic(err)
}

// Create a high-priority email notification job
jobPayload := map[string]interface{}{"to": "user@example.com"}
job := queue.NewJob("email_notification", jobPayload, queue.WithQueue("critical"))

// Enqueue the job
_, err = client.EnqueueJob(job)
if err != nil {
    panic(err)
}
```

This enqueues an `email_notification` job to the `critical` queue, ensuring it receives prompt attention.

## Best Practices

- **Task Prioritization:** Carefully consider each task's urgency. Prioritize tasks to optimize processing efficiency.
- **Graceful Shutdown:** Always stop workers gracefully to safeguard against task loss.
- **Performance Monitoring:** Regularly review and adjust queue priorities and configurations to align with your application's evolving needs.

Employing priority queues can significantly improve your application's processing dynamics, ensuring that critical tasks are handled promptly without sacrificing overall task management flexibility.