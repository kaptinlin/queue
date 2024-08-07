package queue

import (
	"errors"
	"sync"

	"github.com/hibiken/asynq"
)

var (
	ErrJobAlreadyExists = errors.New("job already exists")
	ErrorJobNotFound    = errors.New("job not found")
)

type ConfigProvider interface {
	asynq.PeriodicTaskConfigProvider
	RegisterCronJob(spec string, job *Job) (string, error)
	UnregisterJob(identifier string) error
}

type JobConfig struct {
	Job      *Job   // The job to be scheduled.
	Schedule string // Holds either a cron spec or an interval in string format.
}

// MemoryConfigProvider stores and provides job configurations for periodic execution.
type MemoryConfigProvider struct {
	mu   sync.Mutex
	jobs map[string]JobConfig // Maps job identifiers to their configurations.
}

// NewMemoryConfigProvider initializes a new instance of MemoryConfigProvider.
func NewMemoryConfigProvider() *MemoryConfigProvider {
	return &MemoryConfigProvider{
		jobs: make(map[string]JobConfig),
	}
}

// RegisterCronJob schedules a new job using a cron specification.
func (m *MemoryConfigProvider) RegisterCronJob(spec string, job *Job) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.jobs[job.Fingerprint]; exists {
		return "", ErrJobAlreadyExists
	}

	m.jobs[job.Fingerprint] = JobConfig{
		Job:      job,
		Schedule: spec,
	}

	return job.Fingerprint, nil
}

// UnregisterJob removes a job configuration based on its identifier.
func (m *MemoryConfigProvider) UnregisterJob(identifier string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.jobs[identifier]; !exists {
		return ErrorJobNotFound
	}

	delete(m.jobs, identifier)
	return nil
}

// GetConfigs returns a slice of asynq.PeriodicTaskConfig for all registered jobs.
func (m *MemoryConfigProvider) GetConfigs() ([]*asynq.PeriodicTaskConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	configs := make([]*asynq.PeriodicTaskConfig, 0, len(m.jobs))
	for _, config := range m.jobs {
		task, opts, err := config.Job.ConvertToAsynqTask()
		if err != nil {
			return nil, err
		}
		configs = append(configs, &asynq.PeriodicTaskConfig{
			Cronspec: config.Schedule,
			Task:     task,
			Opts:     opts,
		})
	}
	return configs, nil
}
