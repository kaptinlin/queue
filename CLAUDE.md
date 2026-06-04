# CLAUDE.md

AI development guide for `github.com/kaptinlin/queue`.

`queue` is a Redis-backed distributed job queue library built on Asynq. The package exposes small public concepts: immutable enqueue `Job`, runtime `Delivery`, `Client`, `Worker`, `Manager`, and `Scheduler`. Asynq and Redis stay behind the library boundary except at explicit adapter points.

## Commands

```bash
task test-with-redis    # Start Redis if needed, run integration tests
task test               # Run integration tests; requires Redis on localhost:6379
task lint               # golangci-lint + go mod tidy diff check
task vet                # Static analysis
task default            # lint + test-with-redis
task redis              # Start Redis through Docker Compose
task redis-stop         # Stop Redis
go test                 # Fast root package tests
go test ./... -run '^$'   # Compile all packages without Redis-heavy test execution
go build ./...          # Build all packages
```

`task test-with-redis` depends on Docker Compose. If Docker Compose is unavailable, run the non-Redis checks and report the integration gap clearly.

## Architecture

```text
Client -> Redis queue -> Worker -> Delivery handler -> result/error
                     \-> Manager inspection and operations
                     \-> Scheduler periodic enqueue
```

| Area | Files | Responsibility |
|------|-------|----------------|
| Enqueue | `client.go`, `job.go` | Validate immutable jobs, encode payloads, enqueue Asynq tasks |
| Runtime | `worker.go`, `delivery.go`, `handler.go`, `middleware.go`, `group.go` | Process deliveries, compose middleware, enforce rate limits and timeouts |
| Operations | `manager.go`, `info.go`, `state.go` | Inspect queues/jobs and perform state operations |
| Scheduling | `scheduler.go`, `configs.go` | Register explicit schedule IDs and enqueue jobs on cron/periodic cadence |
| Infrastructure | `redis.go`, `logger.go`, `default_logger.go`, `errors.go` | Redis config, logging adapters, stable error sentinels |

## Documentation Map

No `SPECS/` directory exists in the current tree. Treat source, tests, and feature docs as the current contract.

| Document | Purpose |
|----------|---------|
| [README.md](README.md) | User-facing installation and usage guide |
| [docs/priorities.md](docs/priorities.md) | Priority queue usage |
| [docs/rate_limiting.md](docs/rate_limiting.md) | Worker and handler rate limiting |
| [docs/retention_results.md](docs/retention_results.md) | Retention and delivery result writing |
| [docs/retries.md](docs/retries.md) | Retry policy and skip-retry behavior |
| [docs/timeouts_deadlines.md](docs/timeouts_deadlines.md) | Job deadlines and handler timeouts |
| [docs/scheduler.md](docs/scheduler.md) | Scheduler registration and hooks |
| [docs/config_provider.md](docs/config_provider.md) | Custom scheduler config providers |
| [docs/middleware.md](docs/middleware.md) | Middleware composition |
| [docs/error_handling.md](docs/error_handling.md) | Client and worker error handlers |
| [docs/manager.md](docs/manager.md) | Manager APIs for operational UIs |

## Agent Operating Rules

- Read the relevant source, tests, and docs before editing.
- Keep changes surgical and aligned with the package boundary.
- Prefer the simplest API that preserves a single clear concept.
- Verify the exact behavior you changed; do not treat compilation as full proof.
- Preserve user changes in dirty files and work with them.
- Fail loud on missing Redis, Docker Compose, dependency bugs, or unclear contracts.
- Do not add policy-only gate scripts that restate docs.
- Do not add spec mirror tests that prove prose instead of behavior.

## Design Philosophy

- **KISS** — Each public concept has one job: `Job` is enqueue intent, `Delivery` is runtime fact, `JobInfo` is inspection state.
- **DRY** — Queue state/action rules and error translations should live once and be reused by single and batch paths.
- **SRP** — Client, Worker, Manager, Scheduler, and ConfigProvider have separate responsibilities; do not leak one component's runtime facts into another.
- **ISP** — Avoid fat exported interfaces. Let consumers define the small interfaces their tests or services need.
- **Precision over cleverness** — Names must carry lifecycle meaning: schedule ID, content digest, runtime ID, delivery, snapshot.
- **Never:** workflow engine gravity, hidden process ownership, broad compatibility shims, abstraction theater.

## API Design Principles

- **Progressive disclosure**: Common enqueue/worker flows stay direct; lower-level Asynq conversion remains explicit.
- **Immutable input**: `Job` is built by `NewJob`, validates at construction, and exposes copies through accessors.
- **Handler intent**: `Handler` is built by `NewHandler`, validates type/function/queue at construction, and exposes read-only metadata.
- **Runtime separation**: Handlers receive `*Delivery`, not enqueue `*Job`; result writing belongs to delivery.
- **Explicit identity**: Content digest is diagnostic; schedule ID and runtime task ID are separate facts.

## Coding Rules

### Must Follow

- Go 1.26.4 — use modern stdlib features when they reduce code and preserve clarity.
- Use `github.com/go-json-experiment/json` for payload and result encoding.
- Return errors; do not panic in production code.
- Use `%w` only for errors callers should inspect with `errors.Is` or `errors.As`.
- Keep Asynq types inside adapter boundaries unless the API intentionally exposes an escape hatch.
- Keep OS signal handling in applications and examples, not in package internals.
- Close clients with `Client.Close`; Worker and Scheduler shutdown is context cancellation through `Run(ctx)`.
- Use explicit schedule IDs for scheduler registration.
- Copy mutable inputs such as maps, slices, payload bytes, and time pointers at boundaries.
- Build Manager values through `NewManager`; nil Redis clients and nil inspectors are invalid input.
- Use `t.Context()`, `b.Loop()`, `errors.AsType`, and `for range N` where the existing tests already do.

### Forbidden

- No public `Must*` APIs or panic wrappers.
- No reintroducing mutable public `Job` fields, `WithOptions`, job runtime ID setters, or result writers on `Job`.
- No mutable public `Handler` fields or `Handler.Use`; handler middleware belongs in `NewHandler(..., WithMiddleware(...))`.
- No using content digest as business dedupe, schedule identity, or runtime task identity.
- No exposing `asynq.SkipRetry` as the public skip-retry contract; use `ErrSkipRetry`.
- No worker handler reconstruction through `Inspector`; delivery comes from the task already in hand.
- No `Client.Stop`; resource cleanup is `Client.Close`.
- No raw payload or result bytes in `JobInfo` / `ActiveJobInfo`; use explicit Manager accessors.
- No scheduler enqueue hooks without reliable schedule identity; see `reports/asynq.md`.
- No `Logger.Fatal` or package-owned process exits.
- No working around dependency bugs by reimplementing dependency functionality; report them instead.

## Dependency Issue Reporting

When a dependency bug, limitation, or unexpected behavior blocks work:

1. Do not reimplement the dependency's functionality inside this package.
2. Create `reports/<dependency-name>.md`.
3. Include dependency name/version, trigger scenario, expected vs actual behavior, relevant errors, and a suggested workaround if known.
4. Continue with work that does not depend on the broken behavior.

## Error Handling

- Add stable sentinels to `errors.go` for known queue failure modes.
- Keep dependency errors as wrapped causes, not as the public semantic contract.
- Test public error behavior with `errors.Is` / `errors.As`.
- Avoid double logging: log at the component boundary that owns the context.

## Testing

- Root package tests should cover pure semantics without Redis when possible.
- Integration tests live in `tests/` and require Redis on `localhost:6379`.
- Use `testify` consistently where the suite already does; use `go-cmp` for structural diffs.
- Keep behavior tests focused on public contracts, regressions, and runtime effects.
- Avoid goroutine `t.Fatal`; report through channels or use `assert`/`require` on the owning goroutine.

## Agent Skills

Package-local skills in `.agents/skills/`:

| Skill | When to Use |
|-------|-------------|
| [agent-md-writing](.agents/skills/agent-md-writing/) | Updating `CLAUDE.md` / `AGENTS.md` from current project state |
| [readme-writing](.agents/skills/readme-writing/) | Updating `README.md` usage documentation |
| [go-best-practices](.agents/skills/go-best-practices/) | Go API, naming, error, concurrency, and test judgment |
| [modernizing](.agents/skills/modernizing/) | Adopting Go 1.20-1.26 idioms already accepted by the repo |
| [golangci-linting](.agents/skills/golangci-linting/) | Running or fixing golangci-lint v2 |
| [library-code-simplifying](.agents/skills/library-code-simplifying/) | Simplifying library internals while preserving behavior |
| [library-error-optimizing](.agents/skills/library-error-optimizing/) | Tightening sentinel errors, wrappers, and boundary translation |
| [library-legacy-pruning](.agents/skills/library-legacy-pruning/) | Removing deprecated aliases, compatibility shims, and legacy public surface |
| [library-test-covering](.agents/skills/library-test-covering/) | Expanding production-grade test coverage |
| [library-docs-maintaining](.agents/skills/library-docs-maintaining/) | Coordinating README and agent documentation refreshes |
| [committing](.agents/skills/committing/) | Creating conventional commits |
| [releasing](.agents/skills/releasing/) | Preparing version tags and release notes |
