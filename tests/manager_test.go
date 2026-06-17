package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
)

// TestManagerListWorkers tests the ListWorkers method of the Manager.
func TestManagerListWorkers(t *testing.T) {
	manager := setupTestManager(t)

	workers, err := manager.ListWorkers()
	require.NoError(t, err, "Error listing workers")
	t.Logf("Found %d workers", len(workers))
}

// TestManagerListQueues tests the ListQueues method of the Manager.
func TestManagerListQueues(t *testing.T) {
	manager := setupTestManager(t)

	queues, err := manager.ListQueues()
	require.NoError(t, err, "Error listing queues")
	t.Logf("Found %d queues", len(queues))
}

// TestManagerQueueInfo tests the QueueInfo method for a specific queue.
func TestManagerQueueInfo(t *testing.T) {
	manager := setupTestManager(t)

	queueName := queue.DefaultQueue
	queueInfo, err := manager.QueueInfo(queueName)
	require.NoError(t, err, "Error getting queue info")
	assert.NotNil(t, queueInfo, "Queue info should not be nil")
	t.Logf("Queue '%s' info: %+v", queueName, queueInfo)
}

// TestManagerListJobs tests listing jobs by query.
func TestManagerListJobs(t *testing.T) {
	manager := setupTestManager(t)

	queueName := queue.DefaultQueue
	state := queue.StatePending // Example state
	jobs, err := manager.ListJobs(queue.JobQuery{
		Queue: queueName,
		State: state,
		Page:  queue.Page{Size: 10, Number: 1},
	})
	require.NoError(t, err, "Error listing jobs")
	t.Logf("Found %d jobs in state '%v' in queue '%s'", len(jobs), state, queueName)
}

func setupTestManager(t *testing.T) *queue.Manager {
	t.Helper()

	manager, err := queue.NewManager(getRedisConfig())
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, manager.Close())
	})
	return manager
}
