# Manager

`Manager` exposes operational inspection and state changes for queues, workers, and jobs. It is intended for dashboards, admin tools, and maintenance jobs.

## Initialization

```go
redisConfig, err := queue.NewRedisConfig(queue.WithRedisAddress("localhost:6379"))
if err != nil {
	return err
}
manager, err := queue.NewManager(redisConfig)
if err != nil {
	return err
}
defer manager.Close()
```

## Workers and Queues

```go
workers, err := manager.ListWorkers()
if err != nil {
	return err
}

queueInfo, err := manager.QueueInfo("default")
if err != nil {
	return err
}

stats, err := manager.ListQueueStats("default", 7)
if err != nil {
	return err
}

_ = workers
_ = queueInfo
_ = stats
```

Queue operations return queue-owned sentinels such as `ErrQueueNotFound` and `ErrQueueNotEmpty`, so callers can branch with `errors.Is`. Redis inspection wraps transport failures with `ErrRedisUnavailable` while preserving the underlying cause.

```go
if err := manager.DeleteQueue("default", false); err != nil {
	if errors.Is(err, queue.ErrQueueNotEmpty) {
		return manager.DeleteQueue("default", true)
	}
	return err
}
```

## Redis Inspection

`RedisInfo` is an explicit operational endpoint. `Info` is parsed from Redis `INFO all`, while `RawInfo` keeps the original `INFO all` response for tools that need fields not modeled by `queue`. In cluster mode, `ClusterNodes` keeps the raw `CLUSTER NODES` response and `QueueLocations` reports queue key-slot placement.

Treat `RawInfo` and `ClusterNodes` as infrastructure diagnostics. They can include topology, memory, persistence, replication, module, client, and command statistics from the Redis server. Do not write these raw fields to user-facing logs or responses without your own redaction policy.

```go
info, err := manager.RedisInfo(ctx)
if err != nil {
	return err
}

fmt.Println(info.Address, info.IsCluster, info.Info["redis_version"])
```

## Job Snapshots

`JobInfo` and `ActiveJobInfo` are safe inspection snapshots. They include identity, state, timestamps, queue, retry counts, and payload/result presence with byte sizes. They do not include raw payload or result bytes by default.

```go
jobInfo, err := manager.JobInfo("default", jobID)
if err != nil {
	return err
}

fmt.Println(jobInfo.ID, jobInfo.State, jobInfo.HasPayload, jobInfo.PayloadSize)
```

Retrieve raw payload or result data explicitly when an admin workflow needs it.

```go
payload, err := manager.JobPayload("default", jobID)
if err != nil {
	return err
}

result, err := manager.JobResult("default", jobID)
if err != nil {
	return err
}

_ = payload
_ = result
```

`JobResult` returns `ErrJobResultNotFound` when the job exists but has no retained result.

## Listing Jobs

```go
jobs, err := manager.ListJobs(queue.JobQuery{
	Queue: "default",
	State: queue.StatePending,
	Page:  queue.Page{Size: 50, Number: 1},
})
if err != nil {
	return err
}

for _, job := range jobs {
	fmt.Println(job.ID, job.Type, job.State)
}
```

`Page` is one-based. The zero value uses the first page with 50 jobs; if either field is set, both `Size` and `Number` must be positive.

Use `ListActiveJobs` when a UI needs active worker timing fields.

```go
active, err := manager.ListActiveJobs("default", queue.Page{Size: 50, Number: 1})
if err != nil {
	return err
}
_ = active
```

Aggregating jobs require a group, so list them through the explicit group API.

```go
aggregating, err := manager.ListAggregatingJobs("default", "tenant-a", queue.Page{Size: 50, Number: 1})
if err != nil {
	return err
}
_ = aggregating
```

## State Operations

Single-job operations return stable queue errors.

```go
if err := manager.RunJob("default", jobID); err != nil {
	if errors.Is(err, queue.ErrJobNotFound) {
		return nil
	}
	return err
}
```

State-wide operations return the number of affected jobs.

```go
count, err := manager.ArchiveJobsByState("default", queue.StateRetry)
if err != nil {
	return err
}
fmt.Println("archived", count)
```

Active jobs cannot be archived directly; cancel them first.

```go
cancelled, err := manager.CancelActiveJobs("default", 100)
if err != nil {
	return err
}
fmt.Println("cancelled", cancelled)
```

Aggregating jobs require an explicit group identifier.

```go
count, err := manager.RunAggregatingJobs("default", "tenant-a")
if err != nil {
	return err
}
_ = count
```

## Batch Operations

Batch operations return a `BatchJobResult`. The result records successful job IDs and failed job IDs with their individual causes.

```go
result, err := manager.BatchRunJobs("default", []string{"job-1", "job-2"})
if err != nil {
	for _, failure := range result.Failed {
		if errors.Is(failure.Err, queue.ErrJobNotFound) {
			fmt.Println("missing job", failure.JobID)
			continue
		}
		return err
	}
}

fmt.Println("ran", len(result.Succeeded), "jobs")
```

The same result shape is used by `BatchArchiveJobs`, `BatchCancelJobs`, and `BatchDeleteJobs`.
