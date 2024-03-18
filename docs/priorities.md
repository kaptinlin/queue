# Queue Priority

In the realm of efficient background task processing, leveraging queue priorities is a crucial feature. It allows developers to ensure that critical tasks are attended to before those of lesser urgency. 

## Setting Up Queue Priorities

To utilize queue priorities, you must configure your worker with multiple queues, each assigned a specific priority level. Tasks in higher priority queues are processed more frequently than those in lower priority queues.

### Configuring Your Worker

First, ensure your worker recognizes multiple queues and assigns a priority to each. Here's a sample configuration:

```go
import (
    "github.com/kaptinlin/queue"
)

func main() {
    // Setting up Redis configuration
    redisConfig := queue.NewRedisConfig(queue.WithRedisAddress("localhost:6379"))

    // Creating and configuring the worker with queue priorities
    worker, err := queue.NewWorker(redisConfig,
        queue.WithWorkerQueue("critical", 6), // High-priority queue
        queue.WithWorkerQueue("default", 3),  // Medium-priority queue
        queue.WithWorkerQueue("low", 1),      // Low-priority queue
    )
    if err != nil {
        panic(err)
    }

    // Starting the worker
    if err := worker.Start(); err != nil {
        panic(err)
    }
    // Remember to stop the worker gracefully
    defer worker.Stop()
}
```

In this setup, we define three queues: `critical`, `default`, and `low`, with priorities 6, 3, and 1, respectively. A higher number indicates a higher priority.

### Enqueuing Tasks with Priorities

Once the worker is configured, you can enqueue tasks to specific queues, effectively setting their priority. Here’s an example of enqueuing a task to the `critical` queue, incorporating the `Job` structure for added context:

```go
client, err := queue.NewClient(redisConfig)
if err != nil {
    panic(err)
}

// Defining job options, including the queue name for priority
options := queue.JobOptions{Queue: "critical"}
jobPayload := map[string]interface{}{"to": "user@example.com"}
job := queue.NewJob("email_notification", jobPayload, options)

// Enqueuing the job
_, err = client.EnqueueJob(job)
if err != nil {
    panic(err)
}
```

This example adds an `email_notification` task to the `critical` queue, ensuring it is processed with high priority. The Job structure allows for a more detailed task definition, including payload and options like delay and max retries.

## Best Practices

- **Prioritize Wisely:** Evaluate the importance of each task. Not every task needs immediate processing. Use priorities to manage the load effectively.
- **Graceful Shutdown:** Ensure your workers are stopped gracefully to prevent loss of tasks.
- **Monitor Performance:** Adjust priorities and configurations based on your application's performance and needs.

By leveraging queue priorities, you can enhance your application's efficiency, ensuring critical tasks are processed in a timely manner while maintaining flexibility in task management.