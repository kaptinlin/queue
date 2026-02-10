# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is a **distributed job queue processing library** for Go applications, built on top of [Asynq](https://github.com/hibiken/asynq) and Redis. It provides advanced features including automatic error logging, custom error handling, retries, priority queues, rate limiting, job retention, and distributed worker architecture.

## Common Commands

### Testing
```bash
# Recommended: Run tests with automatic Redis setup and cleanup
make test-with-redis

# Manual Redis management
make redis          # Start Redis service via docker-compose
make test           # Run tests (requires Redis to be running)
make redis-stop     # Stop Redis service

# Run specific test
cd tests && go test -v -run TestName
```

### Code Quality
```bash
# Run all linting checks (includes golangci-lint and go mod tidy verification)
make lint

# Run full quality pipeline (linting + tests with Redis)
make all
```

### Development
```bash
# Clean build artifacts
make clean

# Update dependencies
go get -u all && go mod tidy

# Verify golangci-lint configuration
make golangci-lint
```

## Architecture Overview

### Core Components

**Client-Worker Separation:**
- **Client**: Enqueues jobs to Redis queues. Can run anywhere (API servers, CLIs, etc.)
- **Worker**: Processes jobs from Redis queues. Can be distributed across multiple machines
- **Manager**: Provides inspection and management APIs for building web UIs and monitoring tools
- **Scheduler**: Handles cron-based periodic job scheduling with dynamic configuration

**Job Flow:**
```
Client.Enqueue(job) → Redis Queue → Worker.Process(job) → Result/Error
```

### Key Architectural Patterns

#### 1. Functional Options Pattern
All components use functional options for configuration:
```go
client, _ := queue.NewClient(redisConfig,
    queue.WithClientLogger(logger),
    queue.WithClientErrorHandler(handler),
)

worker, _ := queue.NewWorker(redisConfig,
    queue.WithWorkerConcurrency(10),
    queue.WithWorkerQueue("critical", 10),
)

handler := queue.NewHandler("email:send", handleFunc,
    queue.WithJobTimeout(30*time.Second),
    queue.WithRateLimiter(limiter),
)
```

#### 2. Dual Error Handling System
- **Automatic Logging**: Always logs errors via `Logger` interface (defaults to slog)
- **Optional Custom Handlers**: Implement `ClientErrorHandler` or `WorkerErrorHandler` for metrics/alerts
- Both happen independently - logging is guaranteed, custom handlers are optional

Pattern:
```go
if err != nil {
    logger.Error(...)           // Always happens
    if handler != nil {
        handler.HandleError(...)  // Optional metrics/alerts
    }
}
```

#### 3. Middleware Composition
Supports middleware at three levels:
- **Worker-level**: Applied to all handlers via `worker.Use(middleware)`
- **Group-level**: Applied to grouped handlers via `group.Use(middleware)`
- **Handler-level**: Applied to specific handler via `handler.Use(middleware)`

Middleware functions have signature: `func(HandlerFunc) HandlerFunc`

#### 4. Job Fingerprinting
Every job gets a unique fingerprint (MD5 hash) based on type, payload, and options. Used for:
- Deduplication detection
- Debugging and tracing
- Job identification in logs

#### 5. Asynq Integration Layer
Core types (`Job`, `Handler`, `Worker`) wrap Asynq primitives:
- `Job` ↔ `asynq.Task` conversion via `ConvertToAsynqTask()`
- `HandlerFunc` wraps `asynq.TaskHandler`
- `Worker` manages `asynq.Server` and `asynq.ServeMux`
- `Manager` uses `asynq.Inspector` for introspection

### State Management

Jobs progress through states:
```
pending → scheduled → active → [retry] → completed/archived
                           ↓
                      aggregating (for grouped jobs)
```

State types defined in `state.go`:
- `StatePending`: Queued and ready
- `StateScheduled`: Scheduled for future
- `StateActive`: Currently processing
- `StateRetry`: Failed, awaiting retry
- `StateCompleted`: Successfully finished
- `StateArchived`: Failed permanently
- `StateAggregating`: Grouped jobs awaiting aggregation

### Manager Operations

The `Manager` provides batch operations on jobs by state:
- **Run operations**: Move jobs to pending (scheduled/retry/archived → pending)
- **Archive operations**: Move jobs to archived state (pending/scheduled/retry → archived)
- **Delete operations**: Permanently remove jobs
- **Cancel operations**: Cancel active jobs
- **Batch operations**: Process multiple jobs, return (succeeded, failed) lists

Important constraints:
- Cannot archive already archived jobs
- Cannot run/delete active jobs (must cancel first)
- Aggregating operations require group identifier

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

### Testing Architecture

Tests require Redis and live in `tests/` directory:
- Each test file focuses on a specific component (`client_test.go`, `worker_test.go`, etc.)
- Tests use `testify` for assertions
- Redis connection required at `localhost:6379` (DB 0)
- Cleanup responsibilities: Tests should clean up their own jobs/queues

## Development Guidelines

### Adding New Features

1. **Error Definitions**: Add to `errors.go` with descriptive names
2. **Options Pattern**: Use functional options for all configurable components
3. **Logging**: Always log errors before returning, use structured logging
4. **Documentation**: Update relevant docs in `docs/` directory

### Code Organization

- **Core Types**: `client.go`, `worker.go`, `job.go`, `handler.go`, `manager.go`
- **Configuration**: `redis.go`, `configs.go`, `logger.go`, `default_logger.go`
- **Support**: `errors.go`, `state.go`, `info.go`, `middleware.go`, `group.go`
- **Scheduling**: `scheduler.go`
- **Tests**: `tests/*.go` (uses testify)
- **Docs**: `docs/*.md` (feature-specific documentation)

### Handler Registration Pattern

Always specify queue when registering handlers:
```go
// Good - explicit queue
worker.Register("email:send", handleFunc, queue.WithJobQueue("critical"))

// Default queue if not specified
handler := queue.NewHandler("task:type", handleFunc)  // Uses "default" queue
```

Queue must match one configured in `WithWorkerQueue()` options.

### Rate Limiting Hierarchy

Rate limiting is checked at two levels (both must pass):
1. **Worker-level**: Global rate limit via `WithWorkerRateLimiter()`
2. **Handler-level**: Per-handler rate limit via `WithRateLimiter()`

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

## Environment Requirements

- **Go Version**: 1.25 or higher
- **Redis**: Any version compatible with go-redis v9 and Asynq
- **golangci-lint**: Version 2.7.2 (managed via `.golangci.version`)

## Important Gotchas

1. **Job Payload Encoding**: Uses `github.com/go-json-experiment/json`, not standard library
2. **Fingerprint Stability**: MD5 hash includes options, so changing options changes fingerprint
3. **Active Job Archiving**: Cannot archive active jobs directly - must cancel first
4. **Queue Deletion**: Queue must be empty unless `force=true`
5. **Handler Context**: Context passed to handlers respects deadlines from both job options and handler timeout
6. **Worker Start/Stop**: `Worker.Start()` blocks, `Worker.Stop()` is graceful shutdown
7. **Inspector Queue Names**: Manager operations require exact queue name matching
