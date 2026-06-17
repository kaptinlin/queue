# Optimizing Job Retries

Implementing strategic job retry mechanisms enhances the fault tolerance of your job processing system, ensuring efficiency even in the face of failures.

## Customizing Retry Delays

Adopting an exponential backoff strategy for retry delays helps balance between immediate retries and waiting too long, effectively managing both transient and persistent failures.

### Exponential Backoff Strategy

```go
import "math"

// Exponential backoff increases delay between retries, optimizing for temporary outage recovery.
func ExponentialBackoffDelay(attempts int, _ error) time.Duration {
    return time.Duration(math.Pow(float64(attempts), 2)) * time.Second
}
```

### Applying to a Job Handler

```go
handler, err := queue.NewHandler(
    "process_job",
    ProcessJobHandler,
    queue.WithRetryDelayFunc(ExponentialBackoffDelay), // Implement custom retry delay
)
if err != nil {
    return err
}
```

This setup applies escalating delay times for retries, efficiently spacing out retry attempts.

## Distinguishing Retry Accounting

Some errors should retry a job without increasing failure counters. Use this for conditions where the job did not actually fail, such as a dependency asking the worker to come back later.

### Retrying Without Recording Failure

Return `ErrRetryWithoutFailure` directly when no underlying cause exists. When you have a cause, wrap it with `NewRetryWithoutFailureError` so callers and logs keep the original reason.

```go
func ProcessJobHandler(ctx context.Context, delivery *queue.Delivery) error {
    if err := dependencyNotReady(); err != nil {
        return queue.NewRetryWithoutFailureError(err)
    }
    // Job processing logic
    return nil
}
```

### Addressing Permanent Failures

For errors unlikely to be resolved with retries, prevent further attempts to save resources.
Return queue's skip-retry error; the worker translates it to the underlying engine at the package boundary.

```go
func ProcessJobHandler(ctx context.Context, delivery *queue.Delivery) error {
    if irreversibleError() {
        return queue.NewSkipRetryError("Unrecoverable error identified") // Cease retries
    }
    // Job processing logic
    return nil
}
```
