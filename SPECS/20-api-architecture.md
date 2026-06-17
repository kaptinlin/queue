# API And Architecture

## Public Boundary

Daily usage must stay inside the `queue` package. Backend conversion, scheduler config adaptation, and manager error mapping are private implementation details.

`NewManager(redisConfig)` owns its Redis client and inspector resources and releases them through `Manager.Close`.

> **Why**: Users should learn the queue model once. Exposing backend setup in common constructors makes dependency upgrades and conceptual cleanup expensive.
>
> **Rejected**: Public task conversion helpers, public Redis-to-backend option conversion, and manager constructors that require backend inspectors.

## Lifecycle

Workers and schedulers run through `Run(ctx)`. A canceled context is the shutdown signal; OS signal handling belongs in applications and examples.

Workers are assembled before start. `Use`, `Register`, `RegisterHandler`, and group middleware registration return errors when called after start or after stop. Nil middleware and nil handlers are invalid input.

Schedulers reject nil or already-canceled contexts, cannot be started twice, and cannot be restarted after shutdown.

## Handler Execution

Handler timeout is a processing context deadline. The library does not terminate arbitrary Go code; handlers must observe `ctx.Done()` and return.

Local worker and handler rate limiters apply backpressure with context-aware waits. They are not business errors and must not be counted as job failures.

Business-level retry delay is represented by `RateLimitError`; it carries `RetryAfter` and matches `ErrRetryWithoutFailure`.

> **Why**: Timeout and throttle semantics must not create two realities where the queue records failure while business work continues in another goroutine.
>
> **Rejected**: Goroutine early-return timeout wrappers and local limiter errors masquerading as job failures.

## Error Contract

Public sentinels express queue semantics. Dependency failures remain wrapped causes when callers need diagnosis.

- Invalid delivery construction uses `ErrInvalidDelivery`.
- Missing jobs and queues map to queue sentinels while preserving the backend cause.
- Redis inspection transport failures map to `ErrRedisUnavailable` unless the error is context cancellation, deadline expiry, or an already translated queue error.
- Batch operations return `BatchJobResult` and a joined error for per-job failures.

## Manager API

Manager exposes direct methods for operational screens and automation. It does not use nested public facades.

`ListJobs(JobQuery)` is the general list entry point. `Page{Size, Number}` is normalized before backend calls; invalid pagination returns `ErrInvalidPage`.

State/action rules live once and are reused by single-state operations and batch paths. Active cancellation always rereads the first page after each batch to avoid skipping tasks when the active set shrinks.

`RedisInfo` may expose raw Redis `INFO` and cluster node output because it is an explicit operational API. Other manager snapshots remain summarized.

## Scheduler API

Schedules are registered by explicit ID:

- `RegisterCron(ctx, id, spec, job)`
- `RegisterInterval(ctx, id, interval, job)`
- `Unregister(ctx, id)`

Scheduler enqueue hooks are not public unless they can reliably report schedule identity.

## Forbidden

- Do not add `Client.Stop`; clients release resources through `Close`.
- Do not add package-owned process exits or fatal logging.
- Do not add manager sub-DSLs such as nested `Jobs()` or `Queues()` facades.
- Do not add cursor pagination unless the backend provides true cursor semantics.
- Do not translate dependency errors by string matching outside the manager/adapter boundary.
- Do not expose retry-skip backend sentinels as the public contract.

## Acceptance Criteria

- Public constructors reject nil required dependencies and invalid options.
- Worker mutation after start returns lifecycle errors.
- Handler timeout tests prove handlers must return from context cancellation.
- Local rate limiting waits before handler execution and does not record job failure.
- Manager list and state operations share validation and state/action rules.
- Scheduler registration uses schedule IDs supplied by the caller.
