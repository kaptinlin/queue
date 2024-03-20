# Middleware in `queue` Library

Middleware in the `queue` library enables pre and post-processing around your job handlers, facilitating functionalities like logging, tracing, and metrics collection without cluttering your core logic.

## How It Works

Middleware functions encapsulate your job processing logic. They're executed in sequence, allowing for operations to be inserted before and after the main job handler executes.

## Implementing Middleware

Middleware is crafted as a function that accepts a `HandlerFunc` and returns a modified `HandlerFunc` incorporating the middleware's operations.

### Middleware Signature

```go
type MiddlewareFunc func(HandlerFunc) HandlerFunc
```

### Example: Logging Middleware

Below is an example of logging middleware:

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

Middleware can be applied at various levels for different scopes of control: globally, to job groups, or to individual job handlers.

### Global Middleware

Impacts all jobs processed by a worker.

```go
worker.Use(LoggingMiddleware(log.New(os.Stdout, "", log.LstdFlags)))
```

### Group Middleware

Targets all jobs within a specified group.

```go
emailJobs := worker.Group("email")
emailJobs.Use(TracingMiddleware(tracer))
```

### Handler Middleware

Applies directly to a specified job handler.

```go
worker.Register("send_email", SendEmailHandler, WithMiddleware(LoggingMiddleware(log.New(os.Stdout, "", log.LstdFlags))))
```

## Best Practices

- **Simplicity:** Middleware should focus on a single responsibility for clarity and maintainability.
- **Error Handling:** Middleware offers a strategic point for consistent error management.
- **Performance:** Consider the impact on performance, especially for operations involving I/O.

Thoughtful middleware use can significantly streamline and enhance the handling of background jobs within the `queue` library.