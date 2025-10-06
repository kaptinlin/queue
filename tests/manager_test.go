package tests

import (
	"testing"

	"github.com/hibiken/asynq"
	"github.com/kaptinlin/queue"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManagerListWorkers tests the ListWorkers method of the Manager.
func TestManagerListWorkers(t *testing.T) {
	manager := setupTestManager()

	workers, err := manager.ListWorkers()
	require.NoError(t, err, "Error listing workers")
	t.Logf("Found %d workers", len(workers))
}

// TestManagerListQueues tests the ListQueues method of the Manager.
func TestManagerListQueues(t *testing.T) {
	manager := setupTestManager()

	queues, err := manager.ListQueues()
	require.NoError(t, err, "Error listing queues")
	t.Logf("Found %d queues", len(queues))
}

// TestManagerGetQueueInfo tests the GetQueueInfo method for a specific queue.
func TestManagerGetQueueInfo(t *testing.T) {
	manager := setupTestManager()

	queueName := queue.DefaultQueue
	queueInfo, err := manager.GetQueueInfo(queueName)
	require.NoError(t, err, "Error getting queue info")
	assert.NotNil(t, queueInfo, "Queue info should not be nil")
	t.Logf("Queue '%s' info: %+v", queueName, queueInfo)
}

// TestManagerListJobsByState tests listing jobs by their state in a specific queue.
func TestManagerListJobsByState(t *testing.T) {
	manager := setupTestManager()

	queueName := queue.DefaultQueue
	state := queue.StatePending // Example state
	jobs, err := manager.ListJobsByState(queueName, state, 10, 1)
	require.NoError(t, err, "Error listing jobs by state")
	t.Logf("Found %d jobs in state '%v' in queue '%s'", len(jobs), state, queueName)
}

// setupTestManager is a helper function to initialize a Manager instance for testing.
func setupTestManager() *queue.Manager {
	redisConfig := getRedisConfig()
	asynqRedisOpt := redisConfig.ToAsynqRedisOpt()
	inspector := asynq.NewInspector(asynqRedisOpt)
	redisClient := asynqRedisOpt.MakeRedisClient().(redis.UniversalClient)

	manager := queue.NewManager(redisClient, inspector)
	return manager
}
