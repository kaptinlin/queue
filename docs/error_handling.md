# Error Handling

Proper error handling ensures the resilience and reliability of job processing systems. This section provides a concise guide on managing errors for both `Client` and `Worker` components.

## Client Error Handling

Errors during job enqueueing, such as network issues or invalid job configurations, necessitate a custom error handler for effective management.

Implement the `ClientErrorHandler` interface for custom logic:

```go
type MyClientErrorHandler struct{}

func (h *MyClientErrorHandler) HandleError(err error, job *Job) {
    log.Printf("Enqueue error: %v for job: %v\n", err, job)
}

// Using the custom handler
redisConfig := &RedisConfig{Addr: "localhost:6379"}
client, _ := NewClient(redisConfig, WithClientErrorHandler(&MyClientErrorHandler{}))
```

## Worker Error Handling

For errors encountered during job processing, a similar approach using the `WorkerErrorHandler` interface allows for tailored error resolution strategies.

```go
type MyWorkerErrorHandler struct{}

func (h *MyWorkerErrorHandler) HandleError(err error, job *Job) {
    log.Printf("Processing error: %v for job: %v\n", err, job)
}

// Configuring the worker
redisConfig := &RedisConfig{Addr: "localhost:6379"}
worker, _ := NewWorker(redisConfig, WithWorkerErrorHandler(&MyWorkerErrorHandler{}))
```

These strategies ensure that both enqueuing and processing errors are handled appropriately, maintaining system performance and stability.