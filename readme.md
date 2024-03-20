# Golang Queue Processing Library

This library offers a robust and flexible solution for managing and processing queued jobs in Go applications. Built on top of the [Asynq](https://github.com/hibiken/asynq) task processing library, which uses Redis for storage, it provides advanced features like custom error handling, retries, priority queues, rate limiting, and job retention. Whether you're building a simple task runner or a complex distributed system, this library is designed to meet your needs with efficiency and ease. It also supports the setup of multiple workers across different machines, allowing for scalable and distributed job processing.

## Getting Started

### Installation

Ensure your Go environment is ready (requires Go version 1.21.4 or higher), then install the library:

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

### Job Enqueueing

Quickly enqueue jobs specifying their type and payload:

```go
jobType := "email:send"
payload := map[string]interface{}{"email": "user@example.com", "content": "Welcome!"}

_, err = client.Enqueue(jobType, payload, queue.WithDelay(5*time.Second))
if err != nil {
    log.Printf("Failed to enqueue job: %v", err)
}
```

For more control over job configuration, use a `Job` instance:

```go
job := queue.NewJob(jobType, payload, queue.WithDelay(5*time.Second))
if _, err := client.EnqueueJob(job); err != nil {
    log.Printf("Failed to enqueue job: %v", err)
}
```

### Handling Jobs

Define a function to process jobs of a specific type:

```go
func handleEmailSendJob(ctx context.Context, job *queue.Job) error {
    log.Printf("Sending email to: %s", job.Payload["email"])
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

### Graceful Shutdown

Ensure a clean shutdown process:

```go
func main() {
    // Initialization...

    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)
    <-c

    if err := client.Close(); err != nil {
        log.Fatalf("Failed to close client: %v", err)
    }
    worker.Stop()
}
```

## Advanced Features

Learn more about the library's advanced features by exploring our documentation on:

- [Priority Queues](./docs/priorities.md)
- [Rate Limiting](./docs/rate_limiting.md)
- [Job Retention](./docs/retention.md)
- [Job Retries](./docs/retries.md)
- [Timeouts and Deadlines](./docs/timeouts_deadline.md)
- [Using Middleware](./docs/middleware.md)
- [Error Handling](./docs/error_handling.md)
- [Manager for Web UI Development](./docs/manager.md)

## Contributing

We welcome contributions! Please submit issues or pull requests on GitHub.

## License

This library is licensed under the [MIT License](https://opensource.org/licenses/MIT).

## Credits

Special thanks to the creators of [neoq](https://github.com/acaloiaro/neoq) and [Asynq](https://github.com/hibiken/asynq) for inspiring this library.