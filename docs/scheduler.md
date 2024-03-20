# Scheduler

The `Scheduler` component allows for periodic task scheduling, enabling tasks to be enqueued at regular intervals or based on cron expressions. This is useful for recurring tasks like nightly data backups, weekly reports, or regular data synchronization.

### Scheduler Setup

First, ensure a valid Redis configuration. Initialize the Scheduler with options like time zone and error handling as needed:

```go
redisConfig := queue.NewRedisConfig(
    queue.WithRedisAddress("localhost:6379"),
    // Additional options...
)

scheduler, err := queue.NewScheduler(redisConfig,
    queue.WithSchedulerLocation(time.UTC), // Or your desired time zone
    queue.WithSchedulerEnqueueErrorHandler(func(job *queue.Job, err error) {
        log.Println("Failed to enqueue job:", err)
    }),
)
if err != nil {
    log.Fatal("Failed to create scheduler:", err)
}
```

### Scheduling Tasks

#### Cron Jobs

Schedule tasks with cron expressions:

```go
jobType := "report:generate"
payload := map[string]interface{}{"type": "weekly"}
cronExpression := "0 9 * * 1" // Every Monday at 9:00 AM

_, err = scheduler.RegisterCron(cronExpression, jobType, payload)
if err != nil {
    log.Fatalf("Failed to schedule cron job: %v", err)
}
```

#### Periodic Jobs

Schedule tasks at fixed intervals:

```go
jobType := "status:check"
payload := map[string]interface{}{"service": "database"}
interval := 15 * time.Minute // Every 15 minutes

_, err = scheduler.RegisterPeriodic(interval, jobType, payload)
if err != nil {
    log.Fatalf("Failed to schedule periodic job: %v", err)
}
```

### Starting and Stopping the Scheduler

**Start the Scheduler:**

```go
if err := scheduler.Start(); err != nil {
    log.Fatal("Failed to start scheduler:", err)
}
```

**Stop the Scheduler:**

```go
if err := scheduler.Stop(); err != nil {
    log.Println("Error shutting down scheduler:", err)
}
```
