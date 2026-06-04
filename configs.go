package queue

import (
	"errors"
	"sync"

	"github.com/hibiken/asynq"
)

// Scheduler configuration errors.
var (
	// ErrNoScheduleIDSpecified is returned when a schedule registration has no identifier.
	ErrNoScheduleIDSpecified = errors.New("schedule requires a specified identifier")
	// ErrScheduleAlreadyExists is returned when attempting to register a schedule
	// that already exists in the config provider.
	ErrScheduleAlreadyExists = errors.New("schedule already exists")
	// ErrScheduleNotFound is returned when attempting to unregister a schedule
	// that does not exist in the config provider.
	ErrScheduleNotFound = errors.New("schedule not found")
)

// ConfigProvider defines the interface for managing periodic job configurations.
// It extends [asynq.PeriodicTaskConfigProvider] with registration and
// unregistration capabilities.
type ConfigProvider interface {
	asynq.PeriodicTaskConfigProvider
	RegisterCronJob(identifier, spec string, job *Job) (string, error)
	UnregisterJob(identifier string) error
}

type jobConfig struct {
	job      *Job
	schedule string
}

// MemoryConfigProvider stores and provides job configurations for periodic execution.
type MemoryConfigProvider struct {
	mu   sync.Mutex
	jobs map[string]jobConfig
}

// NewMemoryConfigProvider initializes a new instance of MemoryConfigProvider.
func NewMemoryConfigProvider() *MemoryConfigProvider {
	return &MemoryConfigProvider{
		jobs: make(map[string]jobConfig),
	}
}

// RegisterCronJob schedules a new job using a cron specification.
func (m *MemoryConfigProvider) RegisterCronJob(identifier, spec string, job *Job) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if identifier == "" {
		return "", ErrNoScheduleIDSpecified
	}
	if job == nil {
		return "", ErrInvalidJob
	}

	if _, exists := m.jobs[identifier]; exists {
		return "", ErrScheduleAlreadyExists
	}

	m.jobs[identifier] = jobConfig{
		job:      job,
		schedule: spec,
	}

	return identifier, nil
}

// UnregisterJob removes a job configuration based on its identifier.
func (m *MemoryConfigProvider) UnregisterJob(identifier string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.jobs[identifier]; !exists {
		return ErrScheduleNotFound
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
		task, opts, err := config.job.ConvertToAsynqTask()
		if err != nil {
			return nil, err
		}
		configs = append(configs, &asynq.PeriodicTaskConfig{
			Cronspec: config.schedule,
			Task:     task,
			Opts:     opts,
		})
	}
	return configs, nil
}
