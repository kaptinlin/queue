package tests

import (
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// --- RegisterPeriodicJob ---

func TestSchedulerRegisterPeriodicJob(t *testing.T) {
	scheduler, err := queue.NewScheduler(getRedisConfig())
	require.NoError(t, err)

	job := queue.NewJob("periodic_test", nil)
	id, err := scheduler.RegisterPeriodicJob(
		2*time.Second, job,
	)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

// --- UnregisterCronJob not found ---

func TestSchedulerUnregisterCronJob_NotFound(t *testing.T) {
	scheduler, err := queue.NewScheduler(getRedisConfig())
	require.NoError(t, err)

	err = scheduler.UnregisterCronJob("nonexistent")
	assert.ErrorIs(t, err, queue.ErrConfigJobNotFound)
}
