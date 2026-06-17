# Domain Model

## Core Concepts

`Job` is enqueue intent. It contains a job type, encoded payload bytes, execution options, and a diagnostic content digest. It is created through `NewJob` and exposes copies through accessors.

`Delivery` is runtime fact. It contains the backend task ID, job type, queue, attempt metadata, deadline, copied payload bytes, and result writer for the current processing attempt.

`Handler` is processing intent. It binds a job type, queue, function, middleware, timeout, retry delay function, and local backpressure into a validated immutable processing unit.

`Client` enqueues jobs. It merges client default retention with job options once before enqueue.

`Worker` processes deliveries. It is assembled before `Run(ctx)` and becomes structurally immutable once started.

`Schedule` is scheduler intent. It has an explicit schedule ID, kind, timing data, enabled flag, and `Job`.

`Manager` reports operational facts and performs state actions on jobs and queues.

> **Why**: A distributed queue has several identities in motion. Naming each one separately prevents accidental reuse of diagnostic data as business identity.
>
> **Rejected**: Mutable public job fields, handler mutation after registration, schedule IDs derived from payloads, and runtime IDs stored back on `Job`.

## Identity

- Job type routes work to handlers.
- Runtime task ID identifies one queued or delivered task.
- Schedule ID identifies one recurring registration.
- Content digest is diagnostic only and has the format `q1:sha256:<hex>`.
- Group identifies aggregating jobs for manager operations.

Content digest is derived from job type and current encoded payload bytes. It is not a business dedupe key, schedule ID, or runtime task ID.

## Payloads And Results

- Payloads and results use the package JSON encoder.
- Payload bytes are copied when a job or delivery crosses a boundary.
- `JobInfo` and `ActiveJobInfo` expose payload/result presence and sizes, not raw bytes.
- Raw payload and result bytes are available only through explicit manager accessors.
- Handler-written results require retained completed job metadata.

> **Why**: Operational views should be safe by default. Raw bytes are useful for debugging but may contain sensitive data, so access must be explicit.
>
> **Rejected**: Raw payload/result fields on manager snapshots and result writers on `Job`.

## Configuration

`RedisConfig` is constructed through `NewRedisConfig` or `DefaultRedisConfig`. Its fields are private, getters return values or copies, and TLS configuration is copied on input and output.

Components accept `*RedisConfig`; they reject nil and rely on constructor validation for non-nil configs.

## Scheduling Model

`ScheduleStore` stores queue-owned `Schedule` values through `Put`, `Delete`, and `List`. The store contract does not expose scheduler backend config.

Cron and interval schedules are different kinds. Cron schedules require a cron expression; interval schedules require a positive duration. Every schedule requires an explicit non-empty ID and non-nil job.

## Forbidden

- Do not use content digest for business dedupe, runtime lookup, or schedule identity.
- Do not expose mutable `Job`, `Handler`, or `RedisConfig` fields.
- Do not put raw payload or result bytes in snapshot structs.
- Do not make `ScheduleStore` return backend scheduler config.
- Do not add generic payload codec configuration without a real product requirement.

## Acceptance Criteria

- `NewJob` rejects empty type, empty queue, negative timing/retry options, and unserializable payloads.
- `NewRedisConfig` rejects invalid address, network, DB, pool, timeout, or TLS state.
- `Delivery` is created from the task currently being processed, not reconstructed through manager inspection.
- `ScheduleStore` implementations can persist schedules without importing backend packages.
