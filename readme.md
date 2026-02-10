# Golang Queue Processing Library

This library offers a robust and flexible solution for managing and processing queued jobs in Go applications. Built on top of the [Asynq](https://github.com/hibiken/asynq) task processing library, which uses Redis for storage, it provides advanced features like automatic error logging, custom error handling, retries, priority queues, rate limiting, and job retention. Whether you're building a simple task runner or a complex distributed system, this library is designed to meet your needs with efficiency and ease. It also supports the setup of multiple workers across different machines, allowing for scalable and distributed job processing.

## Getting Started

### Installation

Ensure your Go environment is ready (requires Go version 1.25 or higher), then install the library:

```bash
go get -u github.com/kaptinlin/queue
```

### Configuring Redis

Set up your Redis connection with minimal hassle:

```go
import "github.com/kaptinlin/queue"

redisConfig := queue.NewRedisConfig(
    queue.WithRedisAddress("localhost:6379"),
    queue.WithRedisDB(0),
    queue.WithRedisPassword("your_password"),
)
```

### Client Initialization

Create a client using the Redis configuration:

```go
client, err := queue.NewClient(redisConfig)
if err != nil {
    log.Fatalf("Error initializing client: %v", err)
}
```

The client automatically logs errors using structured logging. For custom logging or advanced error handling (metrics, alerts), see [Error Handling](./docs/error_handling.md).

```go
// Optional: Custom logger
client, err := queue.NewClient(redisConfig,
    queue.WithClientLogger(myLogger))

// Optional: Advanced error handling (metrics, alerts)
client, err := queue.NewClient(redisConfig,
    queue.WithClientErrorHandler(&MyErrorHandler{}))
```
### Job Enqueueing

Enqueue jobs by specifying their type and a structured payload for clear and concise data handling:

```go
type EmailPayload struct {
    Email   string `json:"email"`
    Content string `json:"content"`
}

jobType := "email:send"
payload := EmailPayload{Email: "user@example.com", Content: "Welcome to our service!"}

_, err = client.Enqueue(jobType, payload, queue.WithDelay(5*time.Second))
if err != nil {
    log.Printf("Failed to enqueue job: %v", err)
}
```

Alternatively, for direct control over job configuration, use a `Job` instance:

```go
job := queue.NewJob(jobType, payload, queue.WithDelay(5*time.Second))
if _, err := client.EnqueueJob(job); err != nil {
    log.Printf("Failed to enqueue job: %v", err)
}
```

This approach allows you to specify additional job options such as execution delay, directly within the `Job` object.

### Handling Jobs

Define a function to process jobs of a specific type. Utilize the `EmailPayload` struct for type-safe payload handling:

```go
func handleEmailSendJob(ctx context.Context, job *queue.Job) error {
    var payload EmailPayload
    if err := job.DecodePayload(&payload); err != nil {
        return fmt.Errorf("failed to decode payload: %w", err)
    }

    log.Printf("Sending email to: %s with content: %s", payload.Email, payload.Content)
    // Implement the email sending logic here.
    return nil
}
```

To achieve scalable and distributed job processing, you can register your function and start workers on different machines. Each worker independently processes jobs enqueued by the client:

```go
worker, err := queue.NewWorker(redisConfig, queue.WithWorkerQueue("default", 1))
if err != nil {
    log.Fatalf("Error creating worker: %v", err)
}

err = worker.Register("email:send", handleEmailSendJob)
if err != nil {
    log.Fatalf("Failed to register handler: %v", err)
}

if err := worker.Start(); err != nil {
    log.Fatalf("Failed to start worker: %v", err)
}
```

Workers automatically log processing errors. For custom logging or advanced error handling, use options:

```go
// Optional: Custom logger
worker, err := queue.NewWorker(redisConfig,
    queue.WithWorkerLogger(myLogger))

// Optional: Advanced error handling (metrics, alerts)
worker, err := queue.NewWorker(redisConfig,
    queue.WithWorkerErrorHandler(&MyErrorHandler{}))
```

### Graceful Shutdown

Ensure a clean shutdown process:

```go
func main() {
    // Initialization...

    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)
    <-c

    if err := client.Stop(); err != nil {
        log.Fatalf("Failed to stop client: %v", err)
    }
    worker.Stop()
}
```

## Advanced Features

Learn more about the library's advanced features by exploring our documentation on:

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

## Testing

We provide convenient Make targets for testing with Redis:

```bash
# Recommended: Run tests with automatic Redis setup and cleanup
make test-with-redis

# Or manually manage Redis and run tests
make redis          # Start Redis service
make test           # Run tests
make redis-stop     # Stop Redis service
```

For more detailed testing instructions, see [TEST.md](./TEST.md).

## Contributing

We welcome contributions! Please submit issues or pull requests on GitHub.

## License

This library is licensed under the [MIT License](https://opensource.org/licenses/MIT).

## Credits

Special thanks to the creators of [neoq](https://github.com/acaloiaro/neoq) and [Asynq](https://github.com/hibiken/asynq) for inspiring this library.
