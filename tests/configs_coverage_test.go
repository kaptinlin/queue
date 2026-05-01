package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
)

// --- MemoryConfigProvider ---

func TestMemoryConfigProvider_RegisterDuplicate(t *testing.T) {
	t.Parallel()

	p := queue.NewMemoryConfigProvider()
	job := queue.NewJob("test", nil)

	_, err := p.RegisterCronJob("* * * * *", job)
	require.NoError(t, err)

	_, err = p.RegisterCronJob("* * * * *", job)
	assert.ErrorIs(t, err, queue.ErrJobAlreadyExists)
}

func TestMemoryConfigProvider_UnregisterNotFound(t *testing.T) {
	t.Parallel()

	p := queue.NewMemoryConfigProvider()
	err := p.UnregisterJob("nonexistent")
	assert.ErrorIs(t, err, queue.ErrConfigJobNotFound)
}

func TestMemoryConfigProvider_UnregisterRemovesJob(t *testing.T) {
	t.Parallel()

	p := queue.NewMemoryConfigProvider()
	job := queue.NewJob("test", nil)

	id, err := p.RegisterCronJob("* * * * *", job)
	require.NoError(t, err)

	require.NoError(t, p.UnregisterJob(id))

	configs, err := p.GetConfigs()
	require.NoError(t, err)
	assert.Empty(t, configs)
	assert.ErrorIs(t, p.UnregisterJob(id), queue.ErrConfigJobNotFound)
}

func TestMemoryConfigProvider_GetConfigs(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	p := queue.NewMemoryConfigProvider()
	configs, err := p.GetConfigs()
	require.NoError(t, err)
	assert.Empty(t, configs)
}

func TestMemoryConfigProvider_RegisterDuplicateFingerprint(t *testing.T) {
	t.Parallel()

	p := queue.NewMemoryConfigProvider()
	j1 := queue.NewJob("test", map[string]string{"k": "v"})
	j2 := queue.NewJob("test", map[string]string{"k": "v"})

	_, err := p.RegisterCronJob("* * * * *", j1)
	require.NoError(t, err)

	_, err = p.RegisterCronJob("*/5 * * * *", j2)
	assert.ErrorIs(t, err, queue.ErrJobAlreadyExists)
}

func TestMemoryConfigProvider_GetConfigsErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		job  *queue.Job
		want error
	}{
		{name: "empty type", job: queue.NewJob("", nil), want: queue.ErrNoJobTypeSpecified},
		{name: "empty queue", job: queue.NewJob("test", nil, queue.WithQueue("")), want: queue.ErrNoJobQueueSpecified},
		{name: "serialization failure", job: queue.NewJob("test", func() {}), want: queue.ErrSerializationFailure},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := queue.NewMemoryConfigProvider()
			_, err := p.RegisterCronJob("* * * * *", tc.job)
			require.NoError(t, err)

			configs, err := p.GetConfigs()
			assert.Nil(t, configs)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}
