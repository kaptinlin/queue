# Schedule Store

`Scheduler` stores schedule definitions through `ScheduleStore`. The default store is `MemoryScheduleStore`, which keeps entries only for the current process. Use a custom store when schedules must survive restarts.

`ScheduleStore` is a queue-owned API. It does not expose Asynq config types.

```go
type ScheduleStore interface {
    Put(ctx context.Context, schedule queue.Schedule) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context) ([]queue.Schedule, error)
}
```

## Custom Store

Persist schedules in your database using the schedule ID as the unique key. Store the schedule kind, cron spec or interval, enabled flag, job type, encoded payload, and job options needed to rebuild a `queue.Job`.

`Put` should return `queue.ErrScheduleAlreadyExists` when the ID already exists. `Delete` should return `queue.ErrScheduleNotFound` when the ID is missing. `List` should return enabled and disabled schedules; the scheduler adapter filters disabled entries before handing configs to the backend.

## Usage

```go
store := &SQLScheduleStore{db: db}
scheduler, err := queue.NewScheduler(redisConfig,
    queue.WithScheduleStore(store),
)
if err != nil {
    return err
}

job, err := queue.NewJob("report:generate", map[string]string{"kind": "daily"})
if err != nil {
    return err
}

_, err = scheduler.RegisterCron(ctx, "daily-report", "0 9 * * *", job)
```

Cron and interval schedules stay distinct:

```go
_, err = scheduler.RegisterInterval(ctx, "heartbeat", 15*time.Minute, job)
```
