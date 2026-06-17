package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
)

// --- MemoryScheduleStore ---

func TestMemoryScheduleStore_PutDuplicate(t *testing.T) {
	t.Parallel()

	store := queue.NewMemoryScheduleStore()
	schedule := queue.Schedule{
		ID:       "test-schedule",
		Kind:     queue.ScheduleCron,
		CronSpec: "* * * * *",
		Job:      newJob(t, "test", nil),
		Enabled:  true,
	}

	require.NoError(t, store.Put(t.Context(), schedule))

	err := store.Put(t.Context(), schedule)
	assert.ErrorIs(t, err, queue.ErrScheduleAlreadyExists)
}

func TestMemoryScheduleStore_DeleteNotFound(t *testing.T) {
	t.Parallel()

	store := queue.NewMemoryScheduleStore()
	err := store.Delete(t.Context(), "nonexistent")
	assert.ErrorIs(t, err, queue.ErrScheduleNotFound)
}

func TestMemoryScheduleStore_DeleteRemovesSchedule(t *testing.T) {
	t.Parallel()

	store := queue.NewMemoryScheduleStore()
	schedule := queue.Schedule{
		ID:       "test-schedule",
		Kind:     queue.ScheduleCron,
		CronSpec: "* * * * *",
		Job:      newJob(t, "test", nil),
		Enabled:  true,
	}

	require.NoError(t, store.Put(t.Context(), schedule))
	require.NoError(t, store.Delete(t.Context(), schedule.ID))

	schedules, err := store.List(t.Context())
	require.NoError(t, err)
	assert.Empty(t, schedules)
	assert.ErrorIs(t, store.Delete(t.Context(), schedule.ID), queue.ErrScheduleNotFound)
}

func TestMemoryScheduleStore_List(t *testing.T) {
	t.Parallel()

	store := queue.NewMemoryScheduleStore()
	j1 := newJob(t, "job1", map[string]string{"k": "v1"})
	j2 := newJob(t, "job2", map[string]string{"k": "v2"})

	require.NoError(t, store.Put(t.Context(), queue.Schedule{
		ID:       "job2-schedule",
		Kind:     queue.ScheduleInterval,
		Interval: 5 * time.Second,
		Job:      j2,
		Enabled:  true,
	}))
	require.NoError(t, store.Put(t.Context(), queue.Schedule{
		ID:       "job1-schedule",
		Kind:     queue.ScheduleCron,
		CronSpec: "* * * * *",
		Job:      j1,
		Enabled:  true,
	}))

	schedules, err := store.List(t.Context())
	require.NoError(t, err)
	require.Len(t, schedules, 2)
	assert.Equal(t, "job1-schedule", schedules[0].ID)
	assert.Equal(t, "job2-schedule", schedules[1].ID)
}

func TestMemoryScheduleStore_ListEmpty(t *testing.T) {
	t.Parallel()

	store := queue.NewMemoryScheduleStore()
	schedules, err := store.List(t.Context())
	require.NoError(t, err)
	assert.Empty(t, schedules)
}

func TestMemoryScheduleStore_AllowsDuplicateContentWithDifferentScheduleIDs(t *testing.T) {
	t.Parallel()

	store := queue.NewMemoryScheduleStore()
	j1 := newJob(t, "test", map[string]string{"k": "v"})
	j2 := newJob(t, "test", map[string]string{"k": "v"})

	require.NoError(t, store.Put(t.Context(), queue.Schedule{
		ID:       "first",
		Kind:     queue.ScheduleCron,
		CronSpec: "* * * * *",
		Job:      j1,
		Enabled:  true,
	}))

	err := store.Put(t.Context(), queue.Schedule{
		ID:       "second",
		Kind:     queue.ScheduleCron,
		CronSpec: "*/5 * * * *",
		Job:      j2,
		Enabled:  true,
	})
	assert.NoError(t, err)
}

func TestMemoryScheduleStore_PutValidation(t *testing.T) {
	t.Parallel()

	validJob := newJob(t, "test", nil)
	tests := []struct {
		name     string
		schedule queue.Schedule
		want     error
	}{
		{
			name: "empty identifier",
			schedule: queue.Schedule{
				Kind:     queue.ScheduleCron,
				CronSpec: "* * * * *",
				Job:      validJob,
			},
			want: queue.ErrNoScheduleIDSpecified,
		},
		{
			name: "nil job",
			schedule: queue.Schedule{
				ID:       "test",
				Kind:     queue.ScheduleCron,
				CronSpec: "* * * * *",
			},
			want: queue.ErrInvalidJob,
		},
		{
			name: "invalid kind",
			schedule: queue.Schedule{
				ID:  "test",
				Job: validJob,
			},
			want: queue.ErrInvalidScheduleKind,
		},
		{
			name: "invalid interval",
			schedule: queue.Schedule{
				ID:   "test",
				Kind: queue.ScheduleInterval,
				Job:  validJob,
			},
			want: queue.ErrInvalidPeriodicInterval,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := queue.NewMemoryScheduleStore()
			err := store.Put(t.Context(), tc.schedule)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}
