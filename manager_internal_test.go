package queue

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type managerDebugError string

func (e managerDebugError) Error() string {
	return string(e)
}

func (e managerDebugError) DebugString() string {
	return string(e)
}

func TestParseRedisInfo(t *testing.T) {
	raw := "redis_version:7.0.0\r\nused_memory:1024\r\n# Server\r\n"
	info := parseRedisInfo(raw)
	assert.Equal(t, "7.0.0", info["redis_version"])
	assert.Equal(t, "1024", info["used_memory"])
}

func TestParseRedisInfo_Empty(t *testing.T) {
	info := parseRedisInfo("")
	assert.Empty(t, info)
}

func TestMapManagerError_Nil(t *testing.T) {
	assert.NoError(t, mapManagerError(nil))
}

func TestMapManagerError_AsynqQueueNotFound(t *testing.T) {
	err := fmt.Errorf("inspector: %w", asynq.ErrQueueNotFound)
	got := mapManagerError(err)

	assert.ErrorIs(t, got, ErrQueueNotFound)
	assert.ErrorIs(t, got, asynq.ErrQueueNotFound)
}

func TestMapManagerError_AsynqTaskNotFound(t *testing.T) {
	err := fmt.Errorf("inspector: %w", asynq.ErrTaskNotFound)
	got := mapManagerError(err)

	assert.ErrorIs(t, got, ErrJobNotFound)
	assert.ErrorIs(t, got, asynq.ErrTaskNotFound)
}

func TestMapManagerError_AsynqQueueNotEmpty(t *testing.T) {
	err := fmt.Errorf("inspector: %w", asynq.ErrQueueNotEmpty)
	got := mapManagerError(err)

	assert.ErrorIs(t, got, ErrQueueNotEmpty)
	assert.ErrorIs(t, got, asynq.ErrQueueNotEmpty)
}

func TestMapManagerError_DebugTaskNotFound(t *testing.T) {
	err := managerDebugError("redis: NOT_FOUND cannot find task abc")
	got := mapManagerError(err)

	assert.ErrorIs(t, got, ErrJobNotFound)
	assert.ErrorIs(t, got, err)
}

func TestMapManagerError_DebugQueueNotFound(t *testing.T) {
	err := managerDebugError("redis: NOT_FOUND queue critical does not exist")
	got := mapManagerError(err)

	assert.ErrorIs(t, got, ErrQueueNotFound)
	assert.ErrorIs(t, got, err)
}

func TestMapManagerError_Other(t *testing.T) {
	//nolint:err113 // Test the negative case with a one-off error value.
	err := errors.New("some other error")
	assert.ErrorIs(t, mapManagerError(err), err)
}

func TestMapRedisInfoError_Nil(t *testing.T) {
	assert.NoError(t, mapRedisInfoError(nil))
}

func TestMapRedisInfoError_Context(t *testing.T) {
	err := fmt.Errorf("redis info: %w", context.Canceled)
	assert.ErrorIs(t, mapRedisInfoError(err), context.Canceled)
	assert.NotErrorIs(t, mapRedisInfoError(err), ErrRedisUnavailable)
}

func TestMapRedisInfoError_PreservesManagerSentinel(t *testing.T) {
	err := fmt.Errorf("queue location: %w", ErrQueueNotFound)
	assert.ErrorIs(t, mapRedisInfoError(err), ErrQueueNotFound)
	assert.NotErrorIs(t, mapRedisInfoError(err), ErrRedisUnavailable)
}

func TestMapRedisInfoError_WrapsUnavailable(t *testing.T) {
	//nolint:err113 // Test the boundary behavior with a representative driver error.
	err := errors.New("connection refused")
	got := mapRedisInfoError(err)

	assert.ErrorIs(t, got, ErrRedisUnavailable)
	assert.ErrorIs(t, got, err)
}

func TestRedisInfo_NilContext(t *testing.T) {
	manager, err := NewManager(DefaultRedisConfig())
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, manager.Close())
	})

	info, err := manager.RedisInfo(nil) //nolint:staticcheck // Exercise nil context validation.

	assert.Nil(t, info)
	assert.ErrorIs(t, err, ErrInvalidContext)
}

func TestPageNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		page    Page
		want    Page
		wantErr error
	}{
		{
			name: "zero value uses default first page",
			want: Page{Size: defaultManagerPageSize, Number: 1},
		},
		{
			name: "explicit page",
			page: Page{Size: 25, Number: 2},
			want: Page{Size: 25, Number: 2},
		},
		{
			name:    "missing size",
			page:    Page{Number: 1},
			wantErr: ErrInvalidPage,
		},
		{
			name:    "missing number",
			page:    Page{Size: 25},
			wantErr: ErrInvalidPage,
		},
		{
			name:    "negative size",
			page:    Page{Size: -1, Number: 1},
			wantErr: ErrInvalidPage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.page.normalize()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestManagerListJobsRejectsInvalidPage(t *testing.T) {
	t.Parallel()

	manager := &Manager{}
	jobs, err := manager.ListJobs(JobQuery{
		Queue: "default",
		State: StatePending,
		Page:  Page{Size: -1, Number: 1},
	})

	assert.Nil(t, jobs)
	assert.ErrorIs(t, err, ErrInvalidPage)
}

func TestManagerCancelActiveJobsRejectsInvalidBatchSize(t *testing.T) {
	t.Parallel()

	manager := &Manager{}
	count, err := manager.CancelActiveJobs("default", 0)

	assert.Zero(t, count)
	assert.ErrorIs(t, err, ErrInvalidBatchSize)
}

func TestCancelActiveJobsAlwaysReadsFirstPage(t *testing.T) {
	active := []string{"a", "b", "c"}
	var pages []int
	var canceled []string

	list := func(page int) ([]*asynq.TaskInfo, error) {
		pages = append(pages, page)
		count := min(2, len(active))
		tasks := make([]*asynq.TaskInfo, count)
		for i := range count {
			tasks[i] = &asynq.TaskInfo{ID: active[i]}
		}
		return tasks, nil
	}
	cancel := func(id string) error {
		canceled = append(canceled, id)
		for i, activeID := range active {
			if activeID == id {
				active = append(active[:i], active[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("task %s not active", id)
	}

	count, err := cancelActiveJobs(2, list, cancel)

	require.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.Equal(t, []int{1, 1}, pages)
	assert.Equal(t, []string{"a", "b", "c"}, canceled)
	assert.Empty(t, active)
}
