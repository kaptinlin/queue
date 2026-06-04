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

## Schedule Identity

Every registration uses an explicit schedule ID. The schedule ID identifies the schedule definition; it is separate from the runtime task ID returned when a job is enqueued.

Scheduler enqueue hooks are not exposed by `queue` because Asynq does not provide the periodic schedule ID to its enqueue callbacks. Guessing the schedule ID from task type, payload, or options is ambiguous when two schedules use the same job spec.

## Job Scheduling

### Cron Jobs

For scheduling jobs based on cron expressions:

```go
jobType := "report:generate"
payload := map[string]interface{}{"reportType": "weekly"}
cronExpression := "0 9 * * 1" // Example: Every Monday at 9:00 AM

_, err = scheduler.RegisterCron("daily-report", cronExpression, jobType, payload)
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

_, err = scheduler.RegisterPeriodic("heartbeat", interval, jobType, payload)
if err != nil {
    log.Fatalf("Periodic job scheduling failed: %v", err)
}
```

## Scheduler Operations

**Running the Scheduler:**

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

if err := scheduler.Run(ctx); err != nil {
    log.Fatal("Starting scheduler failed:", err)
}
```

Cancel the context to shut the scheduler down gracefully. Hosts that run the
scheduler from another goroutine should still keep cancellation ownership at the
application boundary.
