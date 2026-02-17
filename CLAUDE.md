# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Distributed job queue processing library** built on [Asynq](https://github.com/hibiken/asynq) and Redis. Provides automatic error logging, custom error handling, retries, priority queues, rate limiting, job retention, and distributed worker architecture for Go applications.

**Module:** `github.com/kaptinlin/queue`
**Go Version:** 1.26+
**Key Dependencies:** asynq, go-redis/v9, go-json-experiment/json, netresearch/go-cron, golang.org/x/time

## Commands

```bash
# Testing (recommended workflow)
make test-with-redis    # Start Redis, run tests, cleanup automatically
make redis              # Start Redis via docker-compose
make test               # Run tests (requires Redis running)
make redis-stop         # Stop Redis service

# Code quality
make lint               # Run golangci-lint + go mod tidy verification
make all                # Run lint + test-with-redis

# Development
make clean              # Remove bin/ artifacts
go get -u all && go mod tidy  # Update dependencies
```

## Architecture

### Component Separation

```
Client (enqueue) → Redis Queue → Worker (process) → Result/Error
                                      ↓
                                  Manager (inspect/manage)
                                  Scheduler (cron jobs)
```

**Client:** Enqueues jobs to Redis. Runs anywhere (API servers, CLIs).
**Worker:** Processes jobs from Redis. Distributed across machines.
**Manager:** Inspection and management APIs for web UIs and monitoring.
**Scheduler:** Cron-based periodic job scheduling with dynamic configuration.

### Core Design Patterns

#### 1. Functional Options Pattern
All components use functional options:

```go
client, _ := queue.NewClient(redisConfig,
    queue.WithClientLogger(logger),
    queue.WithClientErrorHandler(handler))

worker, _ := queue.NewWorker(redisConfig,
    queue.WithWorkerConcurrency(10),
    queue.WithWorkerQueue("critical", 10))

handler := queue.NewHandler("email:send", handleFunc,
    queue.WithJobTimeout(30*time.Second),
    queue.WithRateLimiter(limiter))
```

#### 2. Dual Error Handling System
- **Automatic logging:** Always logs errors via `Logger` interface (defaults to slog)
- **Optional custom handlers:** Implement `ClientErrorHandler` or `WorkerErrorHandler` for metrics/alerts
- Both happen independently—logging is guaranteed, custom handlers are optional

```go
if err != nil {
    logger.Error(...)           // Always happens
    if handler != nil {
        handler.HandleError(...)  // Optional metrics/alerts
    }
}
```

#### 3. Middleware Composition
Three levels of middleware:
- **Worker-level:** `worker.Use(middleware)` applies to all handlers
- **Group-level:** `group.Use(middleware)` applies to grouped handlers
- **Handler-level:** `handler.Use(middleware)` applies to specific handler

Middleware signature: `func(HandlerFunc) HandlerFunc`

#### 4. Job Fingerprinting
Every job gets a unique MD5 hash based on type, payload, and options. Used for deduplication detection, debugging, tracing, and job identification in logs.

#### 5. Asynq Integration Layer
Core types wrap Asynq primitives:
- `Job` ↔ `asynq.Task` via `ConvertToAsynqTask()`
- `HandlerFunc` wraps `asynq.TaskHandler`
- `Worker` manages `asynq.Server` and `asynq.ServeMux`
- `Manager` uses `asynq.Inspector` for introspection

### Job State Flow

```
pending → scheduled → active → [retry] → completed/archived
                           ↓
                      aggregating (grouped jobs)
```

States defined in `state.go`:
- `StatePending`: Queued and ready
- `StateScheduled`: Scheduled for future
- `StateActive`: Currently processing
- `StateRetry`: Failed, awaiting retry
- `StateCompleted`: Successfully finished
- `StateArchived`: Failed permanently
- `StateAggregating`: Grouped jobs awaiting aggregation

### Handler Processing Flow

```
Worker.makeHandlerFunc() wraps:
  1. Global rate limiting check (Worker.limiter)
  2. Task → Job reconstruction (using Inspector.GetTaskInfo)
  3. Middleware chain application
  4. Handler.Process():
     a. Handler-level rate limiting check
     b. Timeout context creation (if configured)
     c. Handler.Handle() execution
  5. Error handling and logging
```

### Manager Operations

Batch operations on jobs by state:
- **Run operations:** Move jobs to pending (scheduled/retry/archived → pending)
- **Archive operations:** Move jobs to archived (pending/scheduled/retry → archived)
- **Delete operations:** Permanently remove jobs
- **Cancel operations:** Cancel active jobs
- **Batch operations:** Process multiple jobs, return (succeeded, failed) lists

**Constraints:**
- Cannot archive already archived jobs
- Cannot run/delete active jobs (must cancel first)
- Aggregating operations require group identifier

## Key Types and Interfaces

**Core Types:**
- `Job`: Task with type, payload, options, fingerprint
- `Handler`: Processes specific job types with middleware support
- `Worker`: Processes jobs from queues with concurrency control
- `Client`: Enqueues jobs to Redis queues
- `Manager`: Inspects and manages jobs/queues
- `Scheduler`: Handles cron-based periodic jobs

**Configuration:**
- `RedisConfig`: Redis connection settings
- `WorkerConfig`: Worker concurrency, queues, error handling
- `ClientConfig`: Client error handling, retention
- `JobOptions`: MaxRetries, Queue, Delay, ScheduleAt, Deadline, Retention

**Interfaces:**
- `Logger`: Structured logging interface (default: slog)
- `ClientErrorHandler`: Custom client error handling
- `WorkerErrorHandler`: Custom worker error handling
- `HandlerFunc`: Job processing function signature
- `MiddlewareFunc`: Middleware function signature

## Coding Rules

### Job Payload Encoding
Use `github.com/go-json-experiment/json`, not standard library:

```go
import "github.com/go-json-experiment/json"

payloadBytes, err := json.Marshal(j.Payload)
var payload EmailPayload
err := job.DecodePayload(&payload)
```

### Handler Registration
Always specify queue when registering handlers:

```go
// Good - explicit queue
worker.Register("email:send", handleFunc, queue.WithJobQueue("critical"))

// Default queue if not specified
handler := queue.NewHandler("task:type", handleFunc)  // Uses "default" queue
```

Queue must match one configured in `WithWorkerQueue()` options.

### Rate Limiting Hierarchy
Rate limiting checked at two levels (both must pass):
1. **Worker-level:** Global rate limit via `WithWorkerRateLimiter()`
2. **Handler-level:** Per-handler rate limit via `WithRateLimiter()`

Rate limit failures return `ErrRateLimit` with `RetryAfter` duration.

### Job Result Writing
Jobs can write results back to Redis:

```go
func handler(ctx context.Context, job *queue.Job) error {
    result := processData(job)
    return job.WriteResult(result)  // Requires ResultWriter to be set
}
```

Results require `WithRetention()` option on job to persist.

### Error Definitions
Add new errors to `errors.go` with descriptive names. Use sentinel errors for known failure modes.

### Worker Lifecycle
- `Worker.Start()` blocks until shutdown
- `Worker.Stop()` performs graceful shutdown
- Always handle graceful shutdown in production

## Testing

Tests require Redis and live in `tests/` directory:
- Each test file focuses on specific component (`client_test.go`, `worker_test.go`, etc.)
- Uses `testify` for assertions
- Redis connection required at `localhost:6379` (DB 0)
- Tests should clean up their own jobs/queues

Run tests:

```bash
make test-with-redis  # Recommended: automatic Redis setup/cleanup
make redis && make test && make redis-stop  # Manual Redis management
```

## Code Organization

```
queue/
├── client.go           # Client for enqueueing jobs
├── worker.go           # Worker for processing jobs
├── job.go              # Job type and options
├── handler.go          # Handler type and processing
├── manager.go          # Manager for inspection/management
├── scheduler.go        # Scheduler for cron jobs
├── redis.go            # Redis configuration
├── configs.go          # Configuration types
├── logger.go           # Logger interface
├── default_logger.go   # Default slog-based logger
├── errors.go           # Error definitions
├── state.go            # Job state types
├── info.go             # Job info types
├── middleware.go       # Middleware support
├── group.go            # Handler grouping
├── tests/              # Integration tests (requires Redis)
├── docs/               # Feature-specific documentation
└── examples/           # Usage examples
```

## Important Gotchas

1. **Fingerprint Stability:** MD5 hash includes options, so changing options changes fingerprint
2. **Active Job Archiving:** Cannot archive active jobs directly—must cancel first
3. **Queue Deletion:** Queue must be empty unless `force=true`
4. **Handler Context:** Context respects deadlines from both job options and handler timeout
5. **Inspector Queue Names:** Manager operations require exact queue name matching

## Environment Requirements

- **Go:** 1.26+
- **Redis:** Any version compatible with go-redis v9 and Asynq
- **golangci-lint:** 2.9.0 (managed via `.golangci.version`)

## Advanced Features

See `docs/` for detailed documentation:
- [Priority Queues](./docs/priorities.md)
- [Rate Limiting](./docs/rate_limiting.md)
- [Job Retention and Results](./docs/retention_results.md)
- [Job Retries](./docs/retries.md)
- [Timeouts and Deadlines](./docs/timeouts_deadlines.md)
- [Scheduler](./docs/scheduler.md)
- [Config Provider for Scheduler](./docs/config_provider.md)
- [Using Middleware](./docs/middleware.md)
- [Error Handling](./docs/error_handling.md)
- [Manager for Web UI Development](./docs/manager.md)

## Agent Skills

Package-local skills available in `.agents/skills/`:

- **agent-md-creating:** Generate CLAUDE.md for Go projects
- **code-simplifying:** Refine recently written Go code for clarity
- **committing:** Create conventional commits for Go packages
- **dependency-selecting:** Select Go dependencies from kaptinlin/agentable ecosystem
- **go-best-practices:** Google Go coding best practices and style guide
- **linting:** Set up and run golangci-lint v2
- **modernizing:** Go 1.20-1.26 modernization guide
- **ralphy-initializing:** Initialize Ralphy AI coding loop configuration
- **ralphy-todo-creating:** Create Ralphy TODO.yaml task files
- **readme-creating:** Generate README.md for Go libraries
- **releasing:** Guide release process for Go packages
- **testing:** Write Go tests with testify and Go 1.25+ features

Use the Skill tool to invoke these skills when relevant to your task.
