# Asynq Scheduler Hook Identity

## Dependency

- Package: `github.com/hibiken/asynq`
- Version: `v0.26.0`

## Trigger

`queue.Scheduler` uses `asynq.PeriodicTaskManager` with explicit schedule IDs in its `ConfigProvider`. The underlying Asynq scheduler hooks expose:

- `PreEnqueueFunc(task *asynq.Task, opts []asynq.Option)`
- `PostEnqueueFunc(info *asynq.TaskInfo, err error)`

Neither callback receives the periodic task config, config hash, scheduler entry ID, or caller-provided schedule ID.

## Expected

A periodic enqueue hook should expose the schedule identity that produced the enqueue attempt, plus the runtime task ID when enqueue succeeds.

For `queue`, the stable public event should be able to report:

- schedule ID
- runtime task ID
- job type
- queue
- enqueue error

## Actual

The hook inputs expose task type, payload, enqueue options, and the runtime `TaskInfo` after enqueue. They do not identify which periodic config produced the enqueue.

If two schedule IDs use the same job type, payload, and enqueue options, the hook cannot reliably distinguish them. Reconstructing the schedule ID from task data would be ambiguous and would reintroduce the identity confusion the queue API is removing.

## Suggested Upstream Shape

Expose periodic config identity in Asynq's periodic enqueue callbacks, for example:

```go
type PeriodicEnqueueInfo struct {
    EntryID  string
    Cronspec string
    Task     *Task
    Opts     []Option
}
```

Then call pre/post hooks with that metadata alongside the runtime `TaskInfo` and enqueue error.

## Queue Workaround Policy

Do not infer schedule ID from task type, payload, options, or content digest. Until Asynq exposes periodic config identity, queue hooks must either omit schedule ID or avoid exposing a hook contract that pretends schedule identity is reliable.
