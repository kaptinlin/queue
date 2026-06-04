# Queue

![Go](https://img.shields.io/badge/go-1.26.4-00ADD8?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)

A Redis-backed distributed job queue library for Go with immutable jobs, runtime deliveries, retries, priorities, rate limiting, retention, scheduling, and operational inspection

## Features

- **Immutable jobs**: `NewJob` validates type, queue, payload encoding, and options before enqueue.
- **Runtime deliveries**: Workers process `Delivery` values with raw payload decoding and result writing.
- **Distributed workers**: Multiple workers can process the same Redis-backed queues across machines.
- **Priority queues**: Configure weighted queues such as `critical`, `default`, and `low`.
- **Retries and skip retry**: Use normal errors for retries and `NewSkipRetryError` for unrecoverable jobs.
- **Rate limiting**: Apply global worker limits and per-handler limits.
- **Retention and results**: Persist completed job metadata and handler-written results.
- **Scheduler and manager**: Register explicit schedule IDs and inspect or operate on queues for dashboards.

## Installation

```bash
go get github.com/kaptinlin/queue
```

Requires **Go 1.26.4+** and Redis.

## Quick Start

Enqueue a job from any process:

```go
package main

import (
	"log"
	"time"

	"github.com/kaptinlin/queue"
)

type EmailPayload struct {
	Email   string `json:"email"`
	Content string `json:"content"`
}

func main() {
	redisConfig := queue.NewRedisConfig(queue.WithRedisAddress("localhost:6379"))
	client, err := queue.NewClient(redisConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	payload := EmailPayload{Email: "user@example.com", Content: "Welcome"}
	_, err = client.Enqueue("email:send", payload,
		queue.WithQueue("critical"),
		queue.WithDelay(5*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}
}
```

Run a worker in another process:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kaptinlin/queue"
)

func handleEmail(ctx context.Context, delivery *queue.Delivery) error {
	var payload struct {
		Email   string `json:"email"`
		Content string `json:"content"`
	}
	if err := delivery.DecodePayload(&payload); err != nil {
		return err
	}

	fmt.Println("send email to", payload.Email)
	return nil
}

func main() {
	redisConfig := queue.NewRedisConfig(queue.WithRedisAddress("localhost:6379"))
	worker, err := queue.NewWorker(redisConfig, queue.WithWorkerQueue("critical", 10))
	if err != nil {
		log.Fatal(err)
	}
	if err := worker.Register("email:send", handleEmail, queue.WithJobQueue("critical")); err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := worker.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
```

## Core Concepts

| Concept | Purpose |
|---------|---------|
| `Job` | Immutable enqueue specification created by `NewJob` |
| `Delivery` | Runtime job delivery passed to handlers |
| `Client` | Enqueues jobs into Redis |
| `Worker` | Processes jobs by type and queue |
| `Manager` | Inspects queues/jobs and performs operational state changes |
| `Scheduler` | Enqueues jobs on cron or periodic schedules |

`Delivery` exposes runtime facts for the current processing attempt:

```go
func handle(ctx context.Context, delivery *queue.Delivery) error {
	deadline, hasDeadline := delivery.Deadline()
	log.Printf("job=%s attempt=%d retry=%d/%d deadline=%v present=%t",
		delivery.ID(), delivery.Attempt(), delivery.RetryCount(), delivery.MaxRetry(),
		deadline, hasDeadline)
	return nil
}
```

## Job Options

```go
job, err := queue.NewJob("report:generate", payload,
	queue.WithQueue("critical"),
	queue.WithMaxRetries(5),
	queue.WithRetention(24*time.Hour),
	queue.WithDeadline(&deadline),
)
if err != nil {
	return err
}
id, err := client.EnqueueJob(job)
```

| Option | Description | Default |
|--------|-------------|---------|
| `WithQueue(name)` | Dispatch job to a named queue | `default` |
| `WithDelay(duration)` | Delay processing after enqueue | none |
| `WithScheduleAt(time)` | Process at a specific time | none |
| `WithMaxRetries(n)` | Set retry attempts | Asynq default |
| `WithDeadline(time)` | Set completion deadline | none |
| `WithRetention(duration)` | Retain completed job data/results | client default |

## Middleware

```go
logging := func(next queue.HandlerFunc) queue.HandlerFunc {
	return func(ctx context.Context, delivery *queue.Delivery) error {
		log.Printf("start %s", delivery.Type())
		err := next(ctx, delivery)
		if err != nil {
			log.Printf("failed %s: %v", delivery.Type(), err)
		}
		return err
	}
}

worker.Use(logging)
```

Middleware can be registered at worker, group, or handler level.

## Results and Retention

Handlers can write results when the job has retention configured:

```go
func handleReport(ctx context.Context, delivery *queue.Delivery) error {
	result := map[string]string{"status": "done"}
	return delivery.WriteResult(result)
}
```

Use `WithRetention` when enqueueing the job so Redis keeps the completed metadata and result.

```go
info, err := manager.JobInfo("reports", jobID)
if err != nil {
	return err
}
if info.HasResult {
	result, err := manager.JobResult("reports", jobID)
	if err != nil {
		return err
	}
	_ = result
}
```

Manager snapshots report payload/result presence and byte sizes by default. Raw payload and result bytes are available only through explicit `JobPayload` and `JobResult` calls.

## Scheduler

Schedulers use explicit schedule IDs. The ID is the schedule identity; it is not derived from payload content.

```go
scheduler, err := queue.NewScheduler(redisConfig)
if err != nil {
	return err
}

_, err = scheduler.RegisterCron(
	"daily-report",
	"0 9 * * *",
	"report:generate",
	map[string]string{"kind": "daily"},
	queue.WithQueue("reports"),
)
```

Scheduler enqueue hooks are intentionally not exposed until the underlying scheduler can report reliable schedule identity to callbacks.

## Error Handling

The library returns stable queue errors and wraps dependency failures as causes.

```go
if errors.Is(err, queue.ErrNoJobTypeSpecified) {
	// Fix caller input.
}

if queue.IsErrRateLimit(err) {
	// Retry after the duration carried by ErrRateLimit.
}

if errors.Is(err, queue.ErrRedisUnavailable) {
	// Redis is unavailable; inspect the wrapped cause for transport details.
}

return queue.NewSkipRetryError("invalid payload")
```

Custom handlers can capture metrics or alerts:

```go
type WorkerErrors struct{}

func (WorkerErrors) HandleError(err error, delivery *queue.Delivery) {
	log.Printf("job %s failed: %v", delivery.Type(), err)
}

worker, err := queue.NewWorker(redisConfig, queue.WithWorkerErrorHandler(WorkerErrors{}))
```

## Documentation

- [Priority Queues](docs/priorities.md)
- [Rate Limiting](docs/rate_limiting.md)
- [Retention and Results](docs/retention_results.md)
- [Retries](docs/retries.md)
- [Timeouts and Deadlines](docs/timeouts_deadlines.md)
- [Scheduler](docs/scheduler.md)
- [Config Provider](docs/config_provider.md)
- [Middleware](docs/middleware.md)
- [Error Handling](docs/error_handling.md)
- [Manager](docs/manager.md)

For AI development guidelines, see [AGENTS.md](AGENTS.md).

## Development

```bash
task test-with-redis    # Start Redis if needed, run integration tests
task test               # Run tests; requires Redis on localhost:6379
task lint               # golangci-lint + go mod tidy diff check
task vet                # Static analysis
go test                 # Fast root package tests
go build ./...          # Build all packages
```

`task test-with-redis` requires Docker Compose when Redis is not already running.

## License

This project is licensed under the MIT License.
