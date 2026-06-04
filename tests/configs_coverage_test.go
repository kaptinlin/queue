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
	job := newJob(t, "test", nil)

	_, err := p.RegisterCronJob("test-schedule", "* * * * *", job)
	require.NoError(t, err)

	_, err = p.RegisterCronJob("test-schedule", "* * * * *", job)
	assert.ErrorIs(t, err, queue.ErrScheduleAlreadyExists)
}

func TestMemoryConfigProvider_UnregisterNotFound(t *testing.T) {
	t.Parallel()

	p := queue.NewMemoryConfigProvider()
	err := p.UnregisterJob("nonexistent")
	assert.ErrorIs(t, err, queue.ErrScheduleNotFound)
}

func TestMemoryConfigProvider_UnregisterRemovesJob(t *testing.T) {
	t.Parallel()

	p := queue.NewMemoryConfigProvider()
	job := newJob(t, "test", nil)

	id, err := p.RegisterCronJob("test-schedule", "* * * * *", job)
	require.NoError(t, err)

	require.NoError(t, p.UnregisterJob(id))

	configs, err := p.GetConfigs()
	require.NoError(t, err)
	assert.Empty(t, configs)
	assert.ErrorIs(t, p.UnregisterJob(id), queue.ErrScheduleNotFound)
}

func TestMemoryConfigProvider_GetConfigs(t *testing.T) {
	t.Parallel()

	p := queue.NewMemoryConfigProvider()

	j1 := newJob(t, "job1", map[string]string{"k": "v1"})
	j2 := newJob(t, "job2", map[string]string{"k": "v2"})

	_, err := p.RegisterCronJob("job1-schedule", "* * * * *", j1)
	require.NoError(t, err)
	_, err = p.RegisterCronJob("job2-schedule", "*/5 * * * *", j2)
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

func TestMemoryConfigProvider_AllowsDuplicateContentWithDifferentScheduleIDs(t *testing.T) {
	t.Parallel()

	p := queue.NewMemoryConfigProvider()
	j1 := newJob(t, "test", map[string]string{"k": "v"})
	j2 := newJob(t, "test", map[string]string{"k": "v"})

	_, err := p.RegisterCronJob("first", "* * * * *", j1)
	require.NoError(t, err)

	_, err = p.RegisterCronJob("second", "*/5 * * * *", j2)
	assert.NoError(t, err)
}

func TestMemoryConfigProvider_RegisterValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		job        *queue.Job
		want       error
	}{
		{name: "empty identifier", identifier: "", job: newJob(t, "test", nil), want: queue.ErrNoScheduleIDSpecified},
		{name: "nil job", identifier: "test", job: nil, want: queue.ErrInvalidJob},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := queue.NewMemoryConfigProvider()
			id, err := p.RegisterCronJob(tc.identifier, "* * * * *", tc.job)
			assert.Empty(t, id)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}
