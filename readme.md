# Golang Queue Processing Library

This library presents an efficient and flexible solution for queue processing in Go. Designed to schedule and execute background tasks across various scenarios, it leverages the [Asynq](https://github.com/hibiken/asynq) library and supports Redis as a backend store. It features custom error handling and task processing logic to suit different needs.

## Getting Started

### Installation

To get started, ensure your Go environment is set up (version 1.21.4 or higher), and install the library using:

```bash
go get -u github.com/kaptinlin/queue
```

### Configuring Redis

Configure Redis easily with `NewRedisConfig` and various `WithRedis*` functions:

```go
import "github.com/kaptinlin/queue"

redisConfig := queue.NewRedisConfig(
    queue.WithRedisAddress("localhost:6379"),
    queue.WithRedisDB(0),
    queue.WithRedisPassword("your_password"),
)
```

### Initializing the Client

Initialize a client with the Redis configuration:

```go
client, err := queue.NewClient(redisConfig)
if err != nil {
    log.Fatalf("Error initializing client: %v", err)
}
```

### Enqueuing Jobs

To add a job to the queue, specify its type, payload, and execution options. Use the `Enqueue` method for a quick setup:

```go
jobType := "email:send"
payload := map[string]interface{}{"email": "user@example.com", "content": "Welcome!"}

_, err = client.Enqueue(jobType, payload, queue.WithDelay(5*time.Second))
if err != nil {
    log.Printf("Failed to enqueue job: %v", err)
}
```

Alternatively, for more detailed job configuration, create a `Job` instance and then enqueue it:

```go
job := queue.NewJob(jobType, payload, queue.WithDelay(5*time.Second))
if _, err := client.EnqueueJob(job); err != nil {
    log.Printf("Failed to enqueue job: %v", err)
}
```

### Registering Handlers

To process a specific type of job, first define a function to handle the job:

```go
func handleEmailSendJob(ctx context.Context, job *queue.Job) error {
    log.Printf("Sending email to: %s", job.Payload["email"])
    return nil
}
```

For simplified handler setup, register this function directly with the job type using the `Register` method:

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

For a more detailed setup or to utilize additional handler options, create and register a `Handler` instance:

```go
handler := queue.NewHandler("email:send", handleEmailSendJob)
if err := worker.RegisterHandler(handler); err != nil {
    log.Fatalf("Failed to register handler: %v", err)
}
```

This approach allows for either a straightforward or a more customized configuration of job processing, depending on your needs.

### Graceful Shutdown

Ensure your application shuts down gracefully:

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

## Contributing

Contributions are welcome via GitHub issues or pull requests.

## License

Licensed under the [MIT License](https://opensource.org/licenses/MIT).