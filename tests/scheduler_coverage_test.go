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

// --- WithConfigProvider ---

func TestWithConfigProvider(t *testing.T) {
	provider := queue.NewMemoryConfigProvider()
	scheduler, err := queue.NewScheduler(getRedisConfig(),
		queue.WithConfigProvider(provider),
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
	cfg := &queue.RedisConfig{Network: "bad", Addr: ""}
	_, err := queue.NewScheduler(cfg)
	assert.Error(t, err)
}

func TestSchedulerRegisterCron_AcceptsStandardSpec(t *testing.T) {
	scheduler, err := queue.NewScheduler(getRedisConfig())
	require.NoError(t, err)

	id, err := scheduler.RegisterCron("cron_standard_spec_test", "*/5 * * * *", "cron_standard_spec_test", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

// --- RegisterPeriodicJob ---

func TestSchedulerRegisterPeriodicJob(t *testing.T) {
	scheduler, err := queue.NewScheduler(getRedisConfig())
	require.NoError(t, err)

	job := newJob(t, "periodic_test", nil)
	id, err := scheduler.RegisterPeriodicJob(
		"periodic_test", 2*time.Second, job,
	)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestSchedulerRegisterPeriodicJob_InvalidInterval(t *testing.T) {
	scheduler, err := queue.NewScheduler(getRedisConfig())
	require.NoError(t, err)

	_, err = scheduler.RegisterPeriodicJob("periodic_test", 0, newJob(t, "periodic_test", nil))
	assert.ErrorIs(t, err, queue.ErrInvalidPeriodicInterval)
}

// --- UnregisterCronJob not found ---

func TestSchedulerUnregisterCronJob_NotFound(t *testing.T) {
	scheduler, err := queue.NewScheduler(getRedisConfig())
	require.NoError(t, err)

	err = scheduler.UnregisterCronJob("nonexistent")
	assert.ErrorIs(t, err, queue.ErrScheduleNotFound)
}
