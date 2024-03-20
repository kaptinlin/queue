package tests

import (
	"testing"

	"github.com/hibiken/asynq"
	"github.com/kaptinlin/queue"
	"github.com/redis/go-redis/v9"
)

// TestManager_ListWorkers tests the ListWorkers method of the Manager.
func TestManager_ListWorkers(t *testing.T) {
	manager, err := setupTestManager()
	if err != nil {
		t.Fatalf("Failed to setup test manager: %v", err)
	}

	workers, err := manager.ListWorkers()
	if err != nil {
		t.Errorf("Error listing workers: %v", err)
		return
	}
	t.Logf("Found %d workers", len(workers))
}

// TestManager_ListQueues tests the ListQueues method of the Manager.
func TestManager_ListQueues(t *testing.T) {
	manager, err := setupTestManager()
	if err != nil {
		t.Fatalf("Failed to setup test manager: %v", err)
	}

	queues, err := manager.ListQueues()
	if err != nil {
		t.Errorf("Error listing queues: %v", err)
		return
	}
	t.Logf("Found %d queues", len(queues))
}

// TestManager_GetQueueInfo tests the GetQueueInfo method for a specific queue.
func TestManager_GetQueueInfo(t *testing.T) {
	manager, err := setupTestManager()
	if err != nil {
		t.Fatalf("Failed to setup test manager: %v", err)
	}

	queueName := queue.DefaultQueue
	queueInfo, dailyStats, err := manager.GetQueueInfo(queueName)
	if err != nil {
		t.Errorf("Error getting queue info for '%s': %v", queueName, err)
		return
	}
	t.Logf("Queue '%s' info: %+v", queueName, queueInfo)
	t.Logf("Daily stats: %+v", dailyStats)
}

// TestManager_ListJobsByState tests listing jobs by their state in a specific queue.
func TestManager_ListJobsByState(t *testing.T) {
	manager, err := setupTestManager()
	if err != nil {
		t.Fatalf("Failed to setup test manager: %v", err)
	}

	queueName := queue.DefaultQueue
	state := queue.StatePending // Example state
	jobs, err := manager.ListJobsByState(queueName, state, 10, 1)
	if err != nil {
		t.Errorf("Error listing jobs by state '%v' in queue '%s': %v", state, queueName, err)
		return
	}
	t.Logf("Found %d jobs in state '%v' in queue '%s'", len(jobs), state, queueName)
}

// setupTestManager is a helper function to initialize a Manager instance for testing.
func setupTestManager() (*queue.Manager, error) {
	redisConfig := getRedisConfig()
	asynqRedisOpt := redisConfig.ToAsynqRedisOpt()
	inspector := asynq.NewInspector(asynqRedisOpt)
	redisClient := asynqRedisOpt.MakeRedisClient().(redis.UniversalClient)

	manager := queue.NewManager(redisClient, inspector)
	return manager, nil
}
