# Using middleware

Middleware provides a powerful mechanism to enhance and customize the behavior of task processing within the `queue` library. This functionality allows for executing code before and after your task handlers, enabling common features like logging, tracing, metrics collection, and more, without cluttering your business logic.

## How Middleware Works

Middleware in the `queue` library is functions that wrap around your task processing logic. When a task is executed, the middleware functions are called in the order they were added, allowing each middleware to perform operations both before and after the task handler.

## Implementing Middleware

A middleware is implemented as a function that takes a `HandlerFunc` (the original task processing function) and returns a new `HandlerFunc` that includes the middleware's logic.

### Middleware Signature

```go
type MiddlewareFunc func(HandlerFunc) HandlerFunc
```

### Example: A Simple Logging Middleware

Below is an example of a middleware function that logs the start and completion of tasks.

```go
func LoggingMiddleware(logger *log.Logger) MiddlewareFunc {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, job *Job) error {
            logger.Printf("Starting job: %s", job.Type)
            err := next(ctx, job)
            if err != nil {
                logger.Printf("Job %s failed: %v", job.Type, err)
            } else {
                logger.Printf("Job %s completed successfully", job.Type)
            }
            return err
        }
    }
}
```

## Applying Middleware

Middleware can be applied at three different levels within the `queue` system: globally across all workers, to specific groups of tasks, or directly to individual task handlers.

### Global Middleware

Global middleware affects all tasks processed by a worker.

```go
worker.Use(LoggingMiddleware(log.New(os.Stdout, "", log.LstdFlags)))
```

### Group Middleware

Group middleware is applied to all tasks within a defined group, allowing for specialized processing logic based on task categorization.

```go
emailTasks := worker.Group("email")
emailTasks.Use(TracingMiddleware(tracer))
```

### Handler Middleware

Handler middleware is applied directly to a specific task handler, providing the most granular level of control.

```go
worker.Register("send_email", SendEmailHandler, WithMiddleware(LoggingMiddleware(log.New(os.Stdout, "", log.LstdFlags))))
```

## Best Practices

- **Keep It Simple:** Middleware should be kept simple and focused on a single responsibility to ensure maintainability and readability.
- **Error Handling:** Consider how your middleware will handle and propagate errors. Middleware is an excellent place to implement consistent error logging and handling strategies.
- **Performance Considerations:** Be mindful of the performance impact of your middleware, especially if it involves IO operations such as logging to a file or sending telemetry data over the network.

By judiciously using middleware, you can significantly enhance the functionality and maintainability of your background task processing logic within the `queue` library.