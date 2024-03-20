# Manager for Web UI Development

The `Manager` in the `queue` package gives access to worker, queue, and job data, enabling the creation of web UIs for management and monitoring.

## Initialization

To begin, configure the `Manager` with a Redis client and an Asynq inspector instance:

```go
package main

import (
    "github.com/hibiken/asynq"
    "github.com/kaptinlin/queue"
    "github.com/redis/go-redis/v9"
)

func main() {
    redisConfig := queue.RedisConfig{
        Addr: "localhost:6379",
        DB:   0,
    }
    asynqRedisOpt := redisConfig.ToAsynqRedisOpt()
    inspector := asynq.NewInspector(asynqRedisOpt)
	redisClient := asynqRedisOpt.MakeRedisClient().(redis.UniversalClient)

    manager := queue.NewManager(redisClient, inspector)
}
```

## Operations

### Listing Workers

List all active workers:

```go
workers, err := manager.ListWorkers()
if err != nil {
    fmt.Printf("Error listing workers: %v\n", err)
    return
}
for _, worker := range workers {
    fmt.Printf("Worker ID: %s, Host: %s, Status: %s\n", worker.ID, worker.Host, worker.Status)
}
```

### Managing Queues

Retrieve all queues:

```go
queues, err := manager.ListQueues()
if err != nil {
    fmt.Printf("Error listing queues: %v\n", err)
    return
}
for _, queue := range queues {
    fmt.Printf("Queue Name: %s, Size: %d\n", queue.Queue, queue.Size)
}
```

Fetch detailed queue information:

```go
queueInfo, dailyStats, err := manager.GetQueueInfo("default")
if err != nil {
    fmt.Printf("Error getting queue info: %v\n", err)
    return
}
// Handle queueInfo and dailyStats as required
```

Pause and resume a queue:

```go
if err := manager.PauseQueue("default"); err != nil {
    fmt.Printf("Error pausing queue: %v\n", err)
}

if err := manager.ResumeQueue("default"); err != nil {
    fmt.Printf("Error resuming queue: %v\n", err)
}
```

### Job Management

Listing and managing jobs:

```go
jobs, err := manager.ListJobsByState("default", queue.JobStateActive, 10, 1)
if err != nil {
    fmt.Printf("Error listing jobs: %v\n", err)
    return
}

for _, job := range jobs {
    fmt.Printf("Job ID: %s, Type: %s\n", job.ID, job.Type)
    // Run a job immediately
    if err := manager.RunJob("default", job.ID); err != nil {
        fmt.Printf("Error running job: %v\n", err)
    }
    // Archive a job
    if err := manager.ArchiveJob("default", job.ID); err != nil {
        fmt.Printf("Error archiving job: %v\n", err)
    }
    // Delete a job
    if err := manager.DeleteJob("default", job.ID); err != nil {
        fmt.Printf("Error deleting job: %v\n", err)
    }
}
```

## Additional Operations

### Deleting a Queue

To delete a queue (force deletion if needed):

```go
if err := manager.DeleteQueue("default", true); err != nil {
    fmt.Printf("Error deleting queue: %v\n", err)
}
```

### Running Batch Jobs

For executing multiple jobs at once:

```go
successfulIDs, failedIDs, err := manager.BatchRunJobs("default", []string{"jobID1", "jobID2"})
if err != nil {
    fmt.Printf("Error running batch jobs: %v\n", err)
}
fmt.Printf("Successful Job IDs: %v\nFailed Job IDs: %v\n", successfulIDs, failedIDs)
```

### Cancelling Active Jobs

To cancel active jobs in a queue:

```go
cancelledCount, err := manager.CancelActiveJobs("default", 10, 1)
if err != nil {
    fmt.Printf("Error cancelling active jobs: %v\n", err)
}
fmt.Printf("Cancelled %d jobs.\n", cancelledCount)
```