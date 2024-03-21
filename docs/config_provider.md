# Config Provider For Scheduler

When using the Scheduler without specifying a custom config provider, it defaults to using `MemoryConfigProvider`. This in-memory provider stores job configurations only for the duration of the application's runtime. Consequently, jobs need to be re-registered upon each restart of the application, as the configurations do not persist between sessions.

### Implementing a Custom Config Provider with Database Persistence

To ensure job configurations persist across application restarts, implement a custom `ConfigProvider` that interacts with a database. This approach allows job details to be saved externally, providing durability and resilience for your scheduled tasks.

```go
type CustomConfigProvider struct {
    db *sql.DB // Database connection
}

// Initialize a new CustomConfigProvider
func NewCustomConfigProvider(db *sql.DB) *CustomConfigProvider {
    return &CustomConfigProvider{db: db}
}

// RegisterCronJob stores job configurations in the database
func (c *CustomConfigProvider) RegisterCronJob(spec string, job *Job) (string, error) {
    _, err := c.db.Exec("INSERT INTO jobs (spec, type, payload) VALUES (?, ?, ?)", 
        spec, job.Type, job.Payload)
    if err != nil {
        return "", err
    }
    return job.Fingerprint, nil
}

// GetConfigs retrieves job configurations from the database and converts them to asynq.PeriodicTaskConfig
func (c *CustomConfigProvider) GetConfigs() ([]*asynq.PeriodicTaskConfig, error) {
    rows, err := c.db.Query("SELECT spec, type, payload FROM jobs")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var configs []*asynq.PeriodicTaskConfig
    for rows.Next() {
        var spec, jobType, payload string
        if err := rows.Scan(&spec, &jobType, &payload); err != nil {
            return nil, err
        }
        task, opts, _ := NewJob(jobType, payload).ConvertToAsynqTask() // Example conversion
        configs = append(configs, &asynq.PeriodicTaskConfig{
            Cronspec: spec,
            Task:     task,
            Opts:     opts,
        })
    }
    return configs, nil
}
```

### Usage with Scheduler

Integrate your `CustomConfigProvider` during Scheduler initialization to replace the default in-memory storage:

```go
db, _ := sql.Open("driver-name", "datasource-name")
customProvider := NewCustomConfigProvider(db)

scheduler, err := queue.NewScheduler(redisConfig,
    queue.WithConfigProvider(customProvider),
)
if err != nil {
    log.Fatal("Scheduler initialization with custom config provider failed:", err)
}
```

This setup ensures your job configurations are persisted in a database, allowing them to be automatically reloaded upon application restarts, thus eliminating the need for manual re-registration of jobs.