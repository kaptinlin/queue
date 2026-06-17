package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
)

// --- WithSchedulerLocation ---

func TestWithSchedulerLocation(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	scheduler, err := queue.NewScheduler(getRedisConfig(),
		queue.WithSchedulerLocation(loc),
	)
	require.NoError(t, err)
	assert.NotNil(t, scheduler)
}

// --- WithScheduleStore ---

func TestWithScheduleStore(t *testing.T) {
	store := queue.NewMemoryScheduleStore()
	scheduler, err := queue.NewScheduler(getRedisConfig(),
		queue.WithScheduleStore(store),
	)
	require.NoError(t, err)
	assert.NotNil(t, scheduler)
}

// --- NewScheduler validation ---

func TestNewScheduler_NilRedisConfig(t *testing.T) {
	_, err := queue.NewScheduler(nil)
	assert.ErrorIs(t, err, queue.ErrInvalidRedisConfig)
}

func TestNewScheduler_InvalidRedisConfig(t *testing.T) {
	cfg, err := queue.NewRedisConfig(queue.WithRedisNetwork("bad"))
	assert.Nil(t, cfg)
	assert.ErrorIs(t, err, queue.ErrRedisUnsupportedNetwork)
}

func TestSchedulerRegisterCron_AcceptsStandardSpec(t *testing.T) {
	scheduler, err := queue.NewScheduler(getRedisConfig())
	require.NoError(t, err)

	job := newJob(t, "cron_standard_spec_test", nil)
	id, err := scheduler.RegisterCron(t.Context(), "cron_standard_spec_test", "*/5 * * * *", job)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

// --- RegisterInterval ---

func TestSchedulerRegisterInterval(t *testing.T) {
	scheduler, err := queue.NewScheduler(getRedisConfig())
	require.NoError(t, err)

	job := newJob(t, "periodic_test", nil)
	id, err := scheduler.RegisterInterval(t.Context(), "periodic_test", 2*time.Second, job)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestSchedulerRegisterInterval_InvalidInterval(t *testing.T) {
	scheduler, err := queue.NewScheduler(getRedisConfig())
	require.NoError(t, err)

	_, err = scheduler.RegisterInterval(t.Context(), "periodic_test", 0, newJob(t, "periodic_test", nil))
	assert.ErrorIs(t, err, queue.ErrInvalidPeriodicInterval)
}

// --- Unregister not found ---

func TestSchedulerUnregister_NotFound(t *testing.T) {
	scheduler, err := queue.NewScheduler(getRedisConfig())
	require.NoError(t, err)

	err = scheduler.Unregister(t.Context(), "nonexistent")
	assert.ErrorIs(t, err, queue.ErrScheduleNotFound)
}
