package tests

import (
	"testing"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- MemoryConfigProvider ---

func TestMemoryConfigProvider_RegisterDuplicate(t *testing.T) {
	p := queue.NewMemoryConfigProvider()
	job := queue.NewJob("test", nil)

	_, err := p.RegisterCronJob("* * * * *", job)
	require.NoError(t, err)

	_, err = p.RegisterCronJob("* * * * *", job)
	assert.ErrorIs(t, err, queue.ErrJobAlreadyExists)
}

func TestMemoryConfigProvider_UnregisterNotFound(t *testing.T) {
	p := queue.NewMemoryConfigProvider()
	err := p.UnregisterJob("nonexistent")
	assert.ErrorIs(t, err, queue.ErrConfigJobNotFound)
}

func TestMemoryConfigProvider_GetConfigs(t *testing.T) {
	p := queue.NewMemoryConfigProvider()

	j1 := queue.NewJob("job1", map[string]string{"k": "v1"})
	j2 := queue.NewJob("job2", map[string]string{"k": "v2"})

	_, err := p.RegisterCronJob("* * * * *", j1)
	require.NoError(t, err)
	_, err = p.RegisterCronJob("*/5 * * * *", j2)
	require.NoError(t, err)

	configs, err := p.GetConfigs()
	require.NoError(t, err)
	assert.Len(t, configs, 2)
}

func TestMemoryConfigProvider_GetConfigs_Empty(t *testing.T) {
	p := queue.NewMemoryConfigProvider()
	configs, err := p.GetConfigs()
	require.NoError(t, err)
	assert.Empty(t, configs)
}
