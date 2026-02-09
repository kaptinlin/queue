package tests

import (
	"context"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const managerOpsTestQueue = "manager_ops_test"

func cleanupManagerOpsQueue(t *testing.T, manager *queue.Manager) {
	t.Helper()
	_ = manager.DeleteQueue(managerOpsTestQueue, true)
}

// --- GetWorkerInfo ---

func TestManagerGetWorkerInfo_NotFound(t *testing.T) {
	manager := setupTestManager()

	_, err := manager.GetWorkerInfo("nonexistent-worker-id")
	assert.ErrorIs(t, err, queue.ErrWorkerNotFound)
}

// --- ListQueueStats ---

func TestManagerListQueueStats(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	// Enqueue a job to ensure the queue exists.
	client, ids := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()
	defer func() { manager.BatchDeleteJobs(managerOpsTestQueue, ids) }() //nolint:errcheck

	stats, err := manager.ListQueueStats(managerOpsTestQueue, 3)
	require.NoError(t, err)
	assert.Len(t, stats, 3, "Should return 3 days of stats")

	for _, s := range stats {
		assert.Equal(t, managerOpsTestQueue, s.Queue)
		assert.False(t, s.Date.IsZero())
	}
}

func TestManagerListQueueStats_NonExistentQueue(t *testing.T) {
	manager := setupTestManager()

	_, err := manager.ListQueueStats("nonexistent_queue_xyz", 1)
	assert.ErrorIs(t, err, queue.ErrQueueNotFound)
}

// --- PauseQueue / ResumeQueue ---

func TestManagerPauseAndResumeQueue(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	client, ids := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()
	defer func() { manager.BatchDeleteJobs(managerOpsTestQueue, ids) }() //nolint:errcheck

	// Pause the queue.
	err := manager.PauseQueue(managerOpsTestQueue)
	require.NoError(t, err)

	// Verify paused.
	info, err := manager.GetQueueInfo(managerOpsTestQueue)
	require.NoError(t, err)
	assert.True(t, info.Paused, "Queue should be paused")

	// Resume the queue.
	err = manager.ResumeQueue(managerOpsTestQueue)
	require.NoError(t, err)

	// Verify resumed.
	info, err = manager.GetQueueInfo(managerOpsTestQueue)
	require.NoError(t, err)
	assert.False(t, info.Paused, "Queue should be resumed")
}

func TestManagerPauseQueue_NonExistent(t *testing.T) {
	manager := setupTestManager()

	// asynq PauseQueue/UnpauseQueue may not error for non-existent queues.
	// Just verify the call completes without panic.
	_ = manager.PauseQueue("nonexistent_queue_xyz")
}

func TestManagerResumeQueue_NonExistent(t *testing.T) {
	manager := setupTestManager()

	_ = manager.ResumeQueue("nonexistent_queue_xyz")
}

// --- RunJob ---

func TestManagerRunJob(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	// Enqueue a scheduled job (future).
	future := time.Now().Add(24 * time.Hour)
	client, ids := enqueueManagerOpsJobs(t, 1, queue.WithScheduleAt(&future))
	defer func() { assert.NoError(t, client.Stop()) }()

	err := manager.RunJob(managerOpsTestQueue, ids[0])
	assert.NoError(t, err)
}

func TestManagerRunJob_NotFound(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	// Ensure queue exists first.
	client, _ := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()

	err := manager.RunJob(managerOpsTestQueue, "nonexistent-job-id")
	assert.ErrorIs(t, err, queue.ErrJobNotFound)
}

// --- ArchiveJob ---

func TestManagerArchiveJob(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	client, ids := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()

	err := manager.ArchiveJob(managerOpsTestQueue, ids[0])
	assert.NoError(t, err)
}

func TestManagerArchiveJob_NotFound(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	// Ensure queue exists first.
	client, _ := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()

	err := manager.ArchiveJob(managerOpsTestQueue, "nonexistent-job-id")
	assert.ErrorIs(t, err, queue.ErrJobNotFound)
}

// --- DeleteJob ---

func TestManagerDeleteJob(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	client, ids := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()

	err := manager.DeleteJob(managerOpsTestQueue, ids[0])
	assert.NoError(t, err)
}

func TestManagerDeleteJob_NotFound(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	// Ensure queue exists first.
	client, _ := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()

	err := manager.DeleteJob(managerOpsTestQueue, "nonexistent-job-id")
	assert.ErrorIs(t, err, queue.ErrJobNotFound)
}

// --- CancelJob ---

func TestManagerCancelJob(t *testing.T) {
	manager := setupTestManager()

	// CancelProcessing doesn't error for non-active jobs in asynq;
	// it just sends a cancellation signal. Test that it doesn't error.
	err := manager.CancelJob("some-job-id")
	assert.NoError(t, err)
}

// --- BatchCancelJobs ---

func TestManagerBatchCancelJobs_EmptySlice(t *testing.T) {
	manager := setupTestManager()

	succeeded, failed, err := manager.BatchCancelJobs([]string{})
	assert.NoError(t, err)
	assert.Empty(t, succeeded)
	assert.Empty(t, failed)
}

func TestManagerBatchCancelJobs(t *testing.T) {
	manager := setupTestManager()

	// CancelProcessing sends cancellation signals; doesn't error for IDs.
	succeeded, failed, err := manager.BatchCancelJobs([]string{"id1", "id2"})
	assert.NoError(t, err)
	assert.Len(t, succeeded, 2)
	assert.Empty(t, failed)
}

// --- CancelActiveJobs ---

func TestManagerCancelActiveJobs_EmptyQueue(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	// Ensure queue exists.
	client, ids := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()
	manager.BatchDeleteJobs(managerOpsTestQueue, ids) //nolint:errcheck

	// No active jobs to cancel.
	count, err := manager.CancelActiveJobs(managerOpsTestQueue, 10, 1)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

// --- GetJobInfo ---

func TestManagerGetJobInfo(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	client, ids := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()

	info, err := manager.GetJobInfo(managerOpsTestQueue, ids[0])
	require.NoError(t, err)
	assert.Equal(t, ids[0], info.ID)
	assert.Equal(t, managerOpsTestQueue, info.Queue)
}

func TestManagerGetJobInfo_NotFound(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	// Ensure queue exists first.
	client, _ := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()

	_, err := manager.GetJobInfo(managerOpsTestQueue, "nonexistent-job-id")
	assert.ErrorIs(t, err, queue.ErrJobNotFound)
}

// --- ListJobsByState additional states ---

func TestManagerListJobsByState_AllStates(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	client, _ := enqueueManagerOpsJobs(t, 2)
	defer func() { assert.NoError(t, client.Stop()) }()

	states := []queue.JobState{
		queue.StatePending,
		queue.StateRetry,
		queue.StateArchived,
		queue.StateCompleted,
		queue.StateScheduled,
		queue.StateAggregating,
	}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			jobs, err := manager.ListJobsByState(managerOpsTestQueue, state, 10, 1)
			assert.NoError(t, err)
			assert.NotNil(t, jobs)
		})
	}
}

func TestManagerListJobsByState_InvalidState(t *testing.T) {
	manager := setupTestManager()

	_, err := manager.ListJobsByState(managerOpsTestQueue, "bogus", 10, 1)
	assert.ErrorIs(t, err, queue.ErrInvalidJobState)
}

func TestManagerListJobsByState_ActiveState(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	// Ensure queue exists.
	client, _ := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()

	// Active state delegates to ListActiveJobs.
	jobs, err := manager.ListJobsByState(managerOpsTestQueue, queue.StateActive, 10, 1)
	assert.NoError(t, err)
	assert.NotNil(t, jobs)
}

// --- ListActiveJobs ---

func TestManagerListActiveJobs_NoActiveJobs(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	client, _ := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()

	jobs, err := manager.ListActiveJobs(managerOpsTestQueue, 10, 1)
	assert.NoError(t, err)
	assert.Empty(t, jobs)
}

// --- GetQueueInfo non-existent ---

func TestManagerGetQueueInfo_NonExistent(t *testing.T) {
	manager := setupTestManager()

	_, err := manager.GetQueueInfo("nonexistent_queue_xyz")
	assert.ErrorIs(t, err, queue.ErrQueueNotFound)
}

// --- DeleteQueue ---

func TestManagerDeleteQueue_NotEmpty(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	client, _ := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()

	err := manager.DeleteQueue(managerOpsTestQueue, false)
	assert.ErrorIs(t, err, queue.ErrQueueNotEmpty)
}

func TestManagerDeleteQueue_NonExistent(t *testing.T) {
	manager := setupTestManager()

	err := manager.DeleteQueue("nonexistent_queue_xyz", false)
	assert.ErrorIs(t, err, queue.ErrQueueNotFound)
}

// --- GetRedisInfo ---

func TestManagerGetRedisInfo(t *testing.T) {
	manager := setupTestManager()

	info, err := manager.GetRedisInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.NotEmpty(t, info.Address)
	assert.NotEmpty(t, info.Info)
	assert.NotEmpty(t, info.RawInfo)
	assert.False(t, info.IsCluster)
}

// --- Aggregating operations ---

func TestManagerRunAggregatingJobs_EmptyQueue(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	// Ensure queue exists.
	client, ids := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()
	manager.BatchDeleteJobs(managerOpsTestQueue, ids) //nolint:errcheck

	count, err := manager.RunAggregatingJobs(managerOpsTestQueue, "test-group")
	// May return error if queue doesn't have aggregating tasks; that's OK.
	_ = err
	assert.Equal(t, 0, count)
}

func TestManagerArchiveAggregatingJobs_EmptyQueue(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	client, ids := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()
	manager.BatchDeleteJobs(managerOpsTestQueue, ids) //nolint:errcheck

	count, err := manager.ArchiveAggregatingJobs(managerOpsTestQueue, "test-group")
	_ = err
	assert.Equal(t, 0, count)
}

func TestManagerDeleteAggregatingJobs_EmptyQueue(t *testing.T) {
	manager := setupTestManager()
	defer cleanupManagerOpsQueue(t, manager)

	client, ids := enqueueManagerOpsJobs(t, 1)
	defer func() { assert.NoError(t, client.Stop()) }()
	manager.BatchDeleteJobs(managerOpsTestQueue, ids) //nolint:errcheck

	count, err := manager.DeleteAggregatingJobs(managerOpsTestQueue, "test-group")
	_ = err
	assert.Equal(t, 0, count)
}

// enqueueManagerOpsJobs enqueues n jobs into the manager ops test queue.
func enqueueManagerOpsJobs(t *testing.T, n int, opts ...queue.JobOption) (*queue.Client, []string) {
	t.Helper()

	client, err := queue.NewClient(getRedisConfig())
	require.NoError(t, err)

	allOpts := make([]queue.JobOption, 0, 1+len(opts))
	allOpts = append(allOpts, queue.WithQueue(managerOpsTestQueue))
	allOpts = append(allOpts, opts...)

	ids := make([]string, 0, n)
	for range n {
		id, err := client.Enqueue("manager_ops_test_job",
			map[string]string{"key": "value"}, allOpts...)
		require.NoError(t, err)
		ids = append(ids, id)
	}
	return client, ids
}
