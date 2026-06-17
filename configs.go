package queue

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

// Scheduler configuration errors.
var (
	// ErrNoScheduleIDSpecified is returned when a schedule registration has no identifier.
	ErrNoScheduleIDSpecified = errors.New("schedule requires a specified identifier")
	// ErrScheduleAlreadyExists is returned when attempting to register a schedule
	// that already exists in the schedule store.
	ErrScheduleAlreadyExists = errors.New("schedule already exists")
	// ErrScheduleNotFound is returned when attempting to unregister a schedule
	// that does not exist in the schedule store.
	ErrScheduleNotFound = errors.New("schedule not found")
	// ErrInvalidScheduleKind is returned when a schedule has an unknown kind.
	ErrInvalidScheduleKind = errors.New("invalid schedule kind")
)

// ScheduleKind identifies how a schedule is triggered.
type ScheduleKind string

const (
	// ScheduleCron runs a job on a cron expression.
	ScheduleCron ScheduleKind = "cron"
	// ScheduleInterval runs a job at a fixed interval.
	ScheduleInterval ScheduleKind = "interval"
)

// Schedule describes one persistent scheduler entry.
type Schedule struct {
	ID       string
	Kind     ScheduleKind
	CronSpec string
	Interval time.Duration
	Job      *Job
	Enabled  bool
}

// ScheduleStore stores scheduler entries without exposing the queue backend.
type ScheduleStore interface {
	Put(ctx context.Context, schedule Schedule) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]Schedule, error)
}

// MemoryScheduleStore stores schedules in memory for the lifetime of the process.
type MemoryScheduleStore struct {
	mu        sync.Mutex
	schedules map[string]Schedule
}

// NewMemoryScheduleStore creates an empty in-memory schedule store.
func NewMemoryScheduleStore() *MemoryScheduleStore {
	return &MemoryScheduleStore{
		schedules: make(map[string]Schedule),
	}
}

// Put stores a new schedule.
func (m *MemoryScheduleStore) Put(ctx context.Context, schedule Schedule) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if err := validateSchedule(schedule); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.schedules[schedule.ID]; exists {
		return ErrScheduleAlreadyExists
	}
	m.schedules[schedule.ID] = schedule
	return nil
}

// Delete removes a schedule by ID.
func (m *MemoryScheduleStore) Delete(ctx context.Context, id string) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if id == "" {
		return ErrNoScheduleIDSpecified
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.schedules[id]; !exists {
		return ErrScheduleNotFound
	}
	delete(m.schedules, id)
	return nil
}

// List returns all schedules in deterministic ID order.
func (m *MemoryScheduleStore) List(ctx context.Context) ([]Schedule, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	schedules := make([]Schedule, 0, len(m.schedules))
	for _, schedule := range m.schedules {
		schedules = append(schedules, schedule)
	}
	sort.Slice(schedules, func(i, j int) bool {
		return schedules[i].ID < schedules[j].ID
	})
	return schedules, nil
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalidContext
	}
	return ctx.Err()
}

func validateSchedule(schedule Schedule) error {
	if schedule.ID == "" {
		return ErrNoScheduleIDSpecified
	}
	if schedule.Job == nil {
		return ErrInvalidJob
	}

	switch schedule.Kind {
	case ScheduleCron:
		if schedule.CronSpec == "" {
			return ErrInvalidCronSpec
		}
	case ScheduleInterval:
		if schedule.Interval <= 0 {
			return ErrInvalidPeriodicInterval
		}
	default:
		return ErrInvalidScheduleKind
	}
	return nil
}
