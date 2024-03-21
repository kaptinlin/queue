# Scheduler Documentation

The `Scheduler` enables scheduled job execution at specified intervals or according to cron expressions, facilitating routine operations such as nightly backups, weekly report generation, or periodic data synchronization tasks.

## Setup

Start with a valid Redis configuration. Initialize the Scheduler with necessary options like the time zone:

```go
redisConfig := queue.NewRedisConfig(
    queue.WithRedisAddress("localhost:6379"), // Specify your Redis server address
    // Additional configuration options as needed...
)

scheduler, err := queue.NewScheduler(redisConfig,
    queue.WithSchedulerLocation(time.UTC), // Adjust the time zone as needed
)
if err != nil {
    log.Fatal("Scheduler creation failed:", err)
}
```

## Configuring Hooks for Job Lifecycle Management

### Pre-Enqueue Hook

Incorporate custom logic prior to job enqueuing:

```go
scheduler.WithPreEnqueueFunc(func(job *queue.Job) {
    // Insert pre-enqueue operations here
    log.Printf("Job preparation: %s\n", job.Type)
})
```

### Post-Enqueue Hook

Execute follow-up actions after job enqueuing, especially for error handling:

```go
scheduler.WithPostEnqueueFunc(func(job *queue.JobInfo, err error) {
    if err != nil {
        log.Printf("Enqueue failed for job: %s, error: %v\n", job.Type, err)
    } else {
        log.Printf("Job enqueued successfully: %s\n", job.Type)
    }
})
```

## Job Scheduling

### Cron Jobs

For scheduling jobs based on cron expressions:

```go
jobType := "report:generate"
payload := map[string]interface{}{"reportType": "weekly"}
cronExpression := "0 9 * * 1" // Example: Every Monday at 9:00 AM

_, err = scheduler.RegisterCron(cronExpression, jobType, payload)
if err != nil {
    log.Fatalf("Cron job scheduling failed: %v", err)
}
```

### Periodic Jobs

For interval-based job scheduling:

```go
jobType := "status:check"
payload := map[string]interface{}{"target": "database"}
interval := 15 * time.Minute // Example: Every 15 minutes

_, err = scheduler.RegisterPeriodic(interval, jobType, payload)
if err != nil {
    log.Fatalf("Periodic job scheduling failed: %v", err)
}
```

## Scheduler Operations

**Starting the Scheduler:**

```go
if err := scheduler.Start(); err != nil {
    log.Fatal("Starting scheduler failed:", err)
}
```

**Stopping the Scheduler:**

```go
if err := scheduler.Stop(); err != nil {
    log.Println("Scheduler shutdown error:", err)
}
```
