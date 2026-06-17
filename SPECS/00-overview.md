# Queue Overview

## Scope

`queue` is a Redis-backed distributed job queue library for Go services. It accepts immutable enqueue intent, delivers runtime facts to handlers, schedules recurring work by explicit identity, and exposes operational inspection for queues and jobs.

The public model stays small: `RedisConfig`, `Job`, `Delivery`, `Client`, `Handler`, `Worker`, `Schedule`, `Scheduler`, `Manager`, and stable queue errors.

> **Why**: A queue library survives long-term use when each exported concept has one job and the backend remains a replaceable mechanism.
>
> **Rejected**: Workflow orchestration, product state ownership, broad compatibility shims, and public backend type plumbing.

## Boundaries

- Redis is the required runtime store. If Redis is unavailable, operations fail explicitly.
- Application code owns process lifecycle, signal handling, logging policy, and business retries.
- The package owns enqueue validation, handler delivery construction, schedule registration, queue inspection, and error translation.
- Backend-specific objects stay behind private adapters unless a future low-level escape hatch is intentionally designed.

## Design Rules

- Public APIs must speak in queue concepts, not backend concepts.
- Constructed values must be validated before use.
- Mutable caller input must be copied at package boundaries.
- Runtime identity, schedule identity, and content diagnostics must remain separate.
- Errors must be inspectable with `errors.Is` or `errors.As` when they express queue semantics.
- Usage docs may show examples; specs define the rules that examples must obey.

## Forbidden

- Do not turn `queue` into a workflow engine. Use explicit jobs and schedules.
- Do not use `queue` as an application state store. Persist product state in the application database.
- Do not add a local fallback executor. Redis unavailability is an operational error.
- Do not expose backend task, inspector, or scheduler types as the normal public API.
- Do not preserve removed concepts through compatibility aliases.
- Do not add public `Must*` constructors or panic-based control flow.

## Acceptance Criteria

- README examples can enqueue, process, schedule, and inspect jobs without importing backend packages.
- `SPECS/10-domain-model.md` defines each public concept exactly once.
- `SPECS/20-api-architecture.md` defines lifecycle, error, scheduling, and manager decisions.
- Feature docs under `docs/` describe usage and do not override these specs.
