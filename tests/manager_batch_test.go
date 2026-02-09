package tests

import (
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const managerTestQueue = "manager_batch_test"

// enqueueTestJobs enqueues n jobs into the specified queue and returns their IDs.
func enqueueTestJobs(t *testing.T, n int, opts ...queue.JobOption) (*queue.Client, []string) {
	t.Helper()

	client, err := queue.NewClient(getRedisConfig())
	require.NoError(t, err)

	allOpts := make([]queue.JobOption, 0, 1+len(opts))
	allOpts = append(allOpts, queue.WithQueue(managerTestQueue))
	allOpts = append(allOpts, opts...)

	ids := make([]string, 0, n)
	for range n {
		id, err := client.Enqueue("manager_batch_test_job",
			map[string]string{"key": "value"}, allOpts...)
		require.NoError(t, err)
		ids = append(ids, id)
	}
	return client, ids
}

// cleanupManagerTestQueue deletes the test queue used by manager batch tests.
func cleanupManagerTestQueue(t *testing.T, manager *queue.Manager) {
	t.Helper()
	// Best-effort cleanup; ignore errors if queue doesn't exist.
	_ = manager.DeleteQueue(managerTestQueue, true)
}

// --- RunJobsByState boundary cases ---

func TestRunJobsByState_InvalidState(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	count, err := manager.RunJobsByState(managerTestQueue, queue.JobState("bogus"))
	assert.ErrorIs(t, err, queue.ErrInvalidJobState)
	assert.Equal(t, 0, count)
}

func TestRunJobsByState_UnsupportedStates(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	tests := []struct {
		name  string
		state queue.JobState
		want  error
	}{
		{"Active", queue.StateActive, queue.ErrOperationNotSupported},
		{"Pending", queue.StatePending, queue.ErrOperationNotSupported},
		{"Completed", queue.StateCompleted, queue.ErrOperationNotSupported},
		{"Aggregating", queue.StateAggregating, queue.ErrGroupRequiredForAggregation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := manager.RunJobsByState(managerTestQueue, tt.state)
			assert.ErrorIs(t, err, tt.want)
			assert.Equal(t, 0, count)
		})
	}
}

func TestRunJobsByState_EmptyQueue(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	// Ensure the queue exists but is empty by enqueuing and deleting.
	client, ids := enqueueTestJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()

	_, _, err := manager.BatchDeleteJobs(managerTestQueue, ids)
	require.NoError(t, err)

	for _, state := range []queue.JobState{
		queue.StateScheduled, queue.StateRetry, queue.StateArchived,
	} {
		t.Run(string(state), func(t *testing.T) {
			count, err := manager.RunJobsByState(managerTestQueue, state)
			assert.NoError(t, err)
			assert.Equal(t, 0, count)
		})
	}
}

func TestRunJobsByState_ScheduledJobs(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	future := time.Now().Add(24 * time.Hour)
	client, _ := enqueueTestJobs(t, 3, queue.WithScheduleAt(&future))
	defer func() { assert.NoError(t, client.Stop()) }()

	count, err := manager.RunJobsByState(managerTestQueue, queue.StateScheduled)
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestRunJobsByState_ArchivedJobs(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	// Enqueue pending jobs, then archive them, then run them.
	client, _ := enqueueTestJobs(t, 2)
	defer func() { assert.NoError(t, client.Stop()) }()

	archived, err := manager.ArchiveJobsByState(managerTestQueue, queue.StatePending)
	require.NoError(t, err)
	require.Equal(t, 2, archived)

	count, err := manager.RunJobsByState(managerTestQueue, queue.StateArchived)
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

// --- ArchiveJobsByState boundary cases ---

func TestArchiveJobsByState_InvalidState(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	count, err := manager.ArchiveJobsByState(managerTestQueue, queue.JobState("bogus"))
	assert.ErrorIs(t, err, queue.ErrInvalidJobState)
	assert.Equal(t, 0, count)
}

func TestArchiveJobsByState_UnsupportedStates(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	tests := []struct {
		name  string
		state queue.JobState
		want  error
	}{
		{"Archived", queue.StateArchived, queue.ErrOperationNotSupported},
		{"Completed", queue.StateCompleted, queue.ErrOperationNotSupported},
		{"Active", queue.StateActive, queue.ErrArchivingActiveJobs},
		{"Aggregating", queue.StateAggregating, queue.ErrGroupRequiredForAggregation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := manager.ArchiveJobsByState(managerTestQueue, tt.state)
			assert.ErrorIs(t, err, tt.want)
			assert.Equal(t, 0, count)
		})
	}
}

func TestArchiveJobsByState_EmptyQueue(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	// Ensure queue exists but is empty.
	client, ids := enqueueTestJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()

	_, _, err := manager.BatchDeleteJobs(managerTestQueue, ids)
	require.NoError(t, err)

	for _, state := range []queue.JobState{
		queue.StatePending, queue.StateScheduled, queue.StateRetry,
	} {
		t.Run(string(state), func(t *testing.T) {
			count, err := manager.ArchiveJobsByState(managerTestQueue, state)
			assert.NoError(t, err)
			assert.Equal(t, 0, count)
		})
	}
}

func TestArchiveJobsByState_PendingJobs(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	client, _ := enqueueTestJobs(t, 3)
	defer func() { assert.NoError(t, client.Stop()) }()

	count, err := manager.ArchiveJobsByState(managerTestQueue, queue.StatePending)
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestArchiveJobsByState_ScheduledJobs(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	future := time.Now().Add(24 * time.Hour)
	client, _ := enqueueTestJobs(t, 2, queue.WithScheduleAt(&future))
	defer func() { assert.NoError(t, client.Stop()) }()

	count, err := manager.ArchiveJobsByState(managerTestQueue, queue.StateScheduled)
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

// --- DeleteJobsByState boundary cases ---

func TestDeleteJobsByState_InvalidState(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	count, err := manager.DeleteJobsByState(managerTestQueue, queue.JobState("bogus"))
	assert.ErrorIs(t, err, queue.ErrInvalidJobState)
	assert.Equal(t, 0, count)
}

func TestDeleteJobsByState_UnsupportedStates(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	tests := []struct {
		name  string
		state queue.JobState
		want  error
	}{
		{"Active", queue.StateActive, queue.ErrOperationNotSupported},
		{"Aggregating", queue.StateAggregating, queue.ErrOperationNotSupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := manager.DeleteJobsByState(managerTestQueue, tt.state)
			assert.ErrorIs(t, err, tt.want)
			assert.Equal(t, 0, count)
		})
	}
}

func TestDeleteJobsByState_EmptyQueue(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	// Ensure queue exists but is empty.
	client, ids := enqueueTestJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()

	_, _, err := manager.BatchDeleteJobs(managerTestQueue, ids)
	require.NoError(t, err)

	for _, state := range []queue.JobState{
		queue.StatePending, queue.StateScheduled, queue.StateRetry,
		queue.StateArchived, queue.StateCompleted,
	} {
		t.Run(string(state), func(t *testing.T) {
			count, err := manager.DeleteJobsByState(managerTestQueue, state)
			assert.NoError(t, err)
			assert.Equal(t, 0, count)
		})
	}
}

func TestDeleteJobsByState_PendingJobs(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	client, _ := enqueueTestJobs(t, 3)
	defer func() { assert.NoError(t, client.Stop()) }()

	count, err := manager.DeleteJobsByState(managerTestQueue, queue.StatePending)
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestDeleteJobsByState_ArchivedJobs(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	client, _ := enqueueTestJobs(t, 2)
	defer func() { assert.NoError(t, client.Stop()) }()

	archived, err := manager.ArchiveJobsByState(managerTestQueue, queue.StatePending)
	require.NoError(t, err)
	require.Equal(t, 2, archived)

	count, err := manager.DeleteJobsByState(managerTestQueue, queue.StateArchived)
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestDeleteJobsByState_ScheduledJobs(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	future := time.Now().Add(24 * time.Hour)
	client, _ := enqueueTestJobs(t, 2, queue.WithScheduleAt(&future))
	defer func() { assert.NoError(t, client.Stop()) }()

	count, err := manager.DeleteJobsByState(managerTestQueue, queue.StateScheduled)
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

// --- BatchRunJobs boundary cases ---

func TestBatchRunJobs_EmptySlice(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	succeeded, failed, err := manager.BatchRunJobs(managerTestQueue, []string{})
	assert.NoError(t, err)
	assert.Empty(t, succeeded)
	assert.Empty(t, failed)
}

func TestBatchRunJobs_NonExistentIDs(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	// Ensure queue exists.
	client, ids := enqueueTestJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()
	defer func() { cleanupManagerTestQueue(t, manager) }()

	fakeIDs := []string{"nonexistent-1", "nonexistent-2"}
	succeeded, failed, err := manager.BatchRunJobs(managerTestQueue, fakeIDs)
	assert.Error(t, err)
	assert.Empty(t, succeeded)
	assert.Len(t, failed, 2)

	// Cleanup the real job.
	manager.BatchDeleteJobs(managerTestQueue, ids) //nolint:errcheck
}

func TestBatchRunJobs_PartialFailure(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	// Enqueue scheduled jobs (can be "run").
	future := time.Now().Add(24 * time.Hour)
	client, ids := enqueueTestJobs(t, 2, queue.WithScheduleAt(&future))
	defer func() { assert.NoError(t, client.Stop()) }()

	// Mix valid scheduled IDs with a fake ID.
	mixedIDs := make([]string, 0, len(ids)+1)
	mixedIDs = append(mixedIDs, ids...)
	mixedIDs = append(mixedIDs, "nonexistent-id")
	succeeded, failed, err := manager.BatchRunJobs(managerTestQueue, mixedIDs)
	assert.Error(t, err, "should have partial error")
	assert.Len(t, succeeded, 2)
	assert.Len(t, failed, 1)
	assert.Equal(t, "nonexistent-id", failed[0])
}

// --- BatchArchiveJobs boundary cases ---

func TestBatchArchiveJobs_EmptySlice(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	succeeded, failed, err := manager.BatchArchiveJobs(managerTestQueue, []string{})
	assert.NoError(t, err)
	assert.Empty(t, succeeded)
	assert.Empty(t, failed)
}

func TestBatchArchiveJobs_NonExistentIDs(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	fakeIDs := []string{"nonexistent-1", "nonexistent-2"}
	succeeded, failed, err := manager.BatchArchiveJobs(managerTestQueue, fakeIDs)
	assert.Error(t, err)
	assert.Empty(t, succeeded)
	assert.Len(t, failed, 2)
}

func TestBatchArchiveJobs_AllSucceed(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	client, ids := enqueueTestJobs(t, 3)
	defer func() { assert.NoError(t, client.Stop()) }()

	succeeded, failed, err := manager.BatchArchiveJobs(managerTestQueue, ids)
	assert.NoError(t, err)
	assert.Len(t, succeeded, 3)
	assert.Empty(t, failed)
}

func TestBatchArchiveJobs_PartialFailure(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	client, ids := enqueueTestJobs(t, 2)
	defer func() { assert.NoError(t, client.Stop()) }()

	mixedIDs := make([]string, 0, len(ids)+1)
	mixedIDs = append(mixedIDs, ids...)
	mixedIDs = append(mixedIDs, "nonexistent-id")
	succeeded, failed, err := manager.BatchArchiveJobs(managerTestQueue, mixedIDs)
	assert.Error(t, err)
	assert.Len(t, succeeded, 2)
	assert.Len(t, failed, 1)
}

// --- BatchDeleteJobs boundary cases ---

func TestBatchDeleteJobs_EmptySlice(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	succeeded, failed, err := manager.BatchDeleteJobs(managerTestQueue, []string{})
	assert.NoError(t, err)
	assert.Empty(t, succeeded)
	assert.Empty(t, failed)
}

func TestBatchDeleteJobs_NonExistentIDs(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	fakeIDs := []string{"nonexistent-1", "nonexistent-2"}
	succeeded, failed, err := manager.BatchDeleteJobs(managerTestQueue, fakeIDs)
	assert.Error(t, err)
	assert.Empty(t, succeeded)
	assert.Len(t, failed, 2)
}

func TestBatchDeleteJobs_AllSucceed(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	client, ids := enqueueTestJobs(t, 3)
	defer func() { assert.NoError(t, client.Stop()) }()

	succeeded, failed, err := manager.BatchDeleteJobs(managerTestQueue, ids)
	assert.NoError(t, err)
	assert.Len(t, succeeded, 3)
	assert.Empty(t, failed)
}

func TestBatchDeleteJobs_PartialFailure(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	client, ids := enqueueTestJobs(t, 2)
	defer func() { assert.NoError(t, client.Stop()) }()

	mixedIDs := make([]string, 0, len(ids)+1)
	mixedIDs = append(mixedIDs, ids...)
	mixedIDs = append(mixedIDs, "nonexistent-id")
	succeeded, failed, err := manager.BatchDeleteJobs(managerTestQueue, mixedIDs)
	assert.Error(t, err)
	assert.Len(t, succeeded, 2)
	assert.Len(t, failed, 1)
}

// --- Cross-operation boundary cases ---

func TestRunThenDeleteByState(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	// Enqueue scheduled jobs, run them all, then delete pending.
	future := time.Now().Add(24 * time.Hour)
	client, _ := enqueueTestJobs(t, 3, queue.WithScheduleAt(&future))
	defer func() { assert.NoError(t, client.Stop()) }()

	runCount, err := manager.RunJobsByState(managerTestQueue, queue.StateScheduled)
	require.NoError(t, err)
	require.Equal(t, 3, runCount)

	// Jobs moved to pending; delete them.
	delCount, err := manager.DeleteJobsByState(managerTestQueue, queue.StatePending)
	assert.NoError(t, err)
	assert.Equal(t, 3, delCount)
}

func TestArchiveThenRunByState(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	// Enqueue pending jobs, archive them, then run archived.
	client, _ := enqueueTestJobs(t, 2)
	defer func() { assert.NoError(t, client.Stop()) }()

	archCount, err := manager.ArchiveJobsByState(managerTestQueue, queue.StatePending)
	require.NoError(t, err)
	require.Equal(t, 2, archCount)

	runCount, err := manager.RunJobsByState(managerTestQueue, queue.StateArchived)
	assert.NoError(t, err)
	assert.Equal(t, 2, runCount)

	// Cleanup: delete the now-pending jobs.
	delCount, err := manager.DeleteJobsByState(managerTestQueue, queue.StatePending)
	assert.NoError(t, err)
	assert.Equal(t, 2, delCount)
}

func TestArchiveThenDeleteByState(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	client, _ := enqueueTestJobs(t, 3)
	defer func() { assert.NoError(t, client.Stop()) }()

	archCount, err := manager.ArchiveJobsByState(managerTestQueue, queue.StatePending)
	require.NoError(t, err)
	require.Equal(t, 3, archCount)

	delCount, err := manager.DeleteJobsByState(managerTestQueue, queue.StateArchived)
	assert.NoError(t, err)
	assert.Equal(t, 3, delCount)
}

func TestDoubleArchiveByState_ReturnsZero(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	client, _ := enqueueTestJobs(t, 2)
	defer func() { assert.NoError(t, client.Stop()) }()

	count, err := manager.ArchiveJobsByState(managerTestQueue, queue.StatePending)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// Second archive of pending should return 0 (no pending jobs left).
	count, err = manager.ArchiveJobsByState(managerTestQueue, queue.StatePending)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestDoubleDeleteByState_ReturnsZero(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerTestQueue(t, manager)

	client, _ := enqueueTestJobs(t, 2)
	defer func() { assert.NoError(t, client.Stop()) }()

	count, err := manager.DeleteJobsByState(managerTestQueue, queue.StatePending)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// Second delete should return 0.
	count, err = manager.DeleteJobsByState(managerTestQueue, queue.StatePending)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}
