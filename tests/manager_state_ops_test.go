package tests

import (
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const stateOpsTestQueue = "state_ops_test"

func cleanupStateOpsQueue(t *testing.T, manager *queue.Manager) {
	t.Helper()
	_ = manager.DeleteQueue(stateOpsTestQueue, true)
}

func enqueueStateOpsJobs(t *testing.T, n int, opts ...queue.JobOption) (*queue.Client, []string) {
	t.Helper()
	client, err := queue.NewClient(getRedisConfig())
	require.NoError(t, err)

	allOpts := make([]queue.JobOption, 0, 1+len(opts))
	allOpts = append(allOpts, queue.WithQueue(stateOpsTestQueue))
	allOpts = append(allOpts, opts...)

	ids := make([]string, 0, n)
	for range n {
		id, err := client.Enqueue("state_ops_job",
			map[string]string{"k": "v"}, allOpts...)
		require.NoError(t, err)
		ids = append(ids, id)
	}
	return client, ids
}

// --- RunJobsByState ---

func TestManagerRunJobsByState_Scheduled(t *testing.T) {
	manager := setupTestManager()
	defer cleanupStateOpsQueue(t, manager)

	future := time.Now().Add(24 * time.Hour)
	client, _ := enqueueStateOpsJobs(t, 2,
		queue.WithScheduleAt(&future))
	defer func() { _ = client.Stop() }()

	count, err := manager.RunJobsByState(stateOpsTestQueue,
		queue.StateScheduled)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestManagerRunJobsByState_InvalidState(t *testing.T) {
	manager := setupTestManager()

	_, err := manager.RunJobsByState(stateOpsTestQueue, "bogus")
	assert.ErrorIs(t, err, queue.ErrInvalidJobState)
}

func TestManagerRunJobsByState_Active(t *testing.T) {
	manager := setupTestManager()

	_, err := manager.RunJobsByState(stateOpsTestQueue,
		queue.StateActive)
	assert.ErrorIs(t, err, queue.ErrOperationNotSupported)
}

func TestManagerRunJobsByState_Aggregating(t *testing.T) {
	manager := setupTestManager()

	_, err := manager.RunJobsByState(stateOpsTestQueue,
		queue.StateAggregating)
	assert.ErrorIs(t, err,
		queue.ErrGroupRequiredForAggregation)
}

// --- ArchiveJobsByState ---

func TestManagerArchiveJobsByState_Pending(t *testing.T) {
	manager := setupTestManager()
	defer cleanupStateOpsQueue(t, manager)

	client, _ := enqueueStateOpsJobs(t, 2)
	defer func() { _ = client.Stop() }()

	count, err := manager.ArchiveJobsByState(stateOpsTestQueue,
		queue.StatePending)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 2)
}

func TestManagerArchiveJobsByState_InvalidState(t *testing.T) {
	manager := setupTestManager()

	_, err := manager.ArchiveJobsByState(stateOpsTestQueue,
		"bogus")
	assert.ErrorIs(t, err, queue.ErrInvalidJobState)
}

func TestManagerArchiveJobsByState_Active(t *testing.T) {
	manager := setupTestManager()

	_, err := manager.ArchiveJobsByState(stateOpsTestQueue,
		queue.StateActive)
	assert.ErrorIs(t, err, queue.ErrArchivingActiveJobs)
}

func TestManagerArchiveJobsByState_Aggregating(t *testing.T) {
	manager := setupTestManager()

	_, err := manager.ArchiveJobsByState(stateOpsTestQueue,
		queue.StateAggregating)
	assert.ErrorIs(t, err,
		queue.ErrGroupRequiredForAggregation)
}

// --- DeleteJobsByState ---

func TestManagerDeleteJobsByState_Pending(t *testing.T) {
	manager := setupTestManager()
	defer cleanupStateOpsQueue(t, manager)

	client, _ := enqueueStateOpsJobs(t, 2)
	defer func() { _ = client.Stop() }()

	count, err := manager.DeleteJobsByState(stateOpsTestQueue,
		queue.StatePending)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 2)
}

func TestManagerDeleteJobsByState_InvalidState(t *testing.T) {
	manager := setupTestManager()

	_, err := manager.DeleteJobsByState(stateOpsTestQueue,
		"bogus")
	assert.ErrorIs(t, err, queue.ErrInvalidJobState)
}

func TestManagerDeleteJobsByState_Active(t *testing.T) {
	manager := setupTestManager()

	_, err := manager.DeleteJobsByState(stateOpsTestQueue,
		queue.StateActive)
	assert.ErrorIs(t, err, queue.ErrOperationNotSupported)
}

// --- ListQueues with verification ---

func TestManagerListQueues_ContainsTestQueue(t *testing.T) {
	manager := setupTestManager()
	defer cleanupStateOpsQueue(t, manager)

	client, _ := enqueueStateOpsJobs(t, 1)
	defer func() { _ = client.Stop() }()

	queues, err := manager.ListQueues()
	require.NoError(t, err)
	assert.NotEmpty(t, queues)

	// Verify our test queue is in the list.
	found := false
	for _, q := range queues {
		if q.Queue == stateOpsTestQueue {
			found = true
			break
		}
	}
	assert.True(t, found,
		"Expected test queue in list")
}
