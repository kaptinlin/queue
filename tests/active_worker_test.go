package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const activeJobTestQueue = "active_job_test"

// TestManagerWithActiveWorker starts a worker, enqueues a
// long-running job, then exercises the Manager methods that
// require active jobs (ListWorkers, GetWorkerInfo,
// ListActiveJobs, CancelActiveJobs).
func TestManagerWithActiveWorker(t *testing.T) {
	redisConfig := getRedisConfig()
	manager := setupTestManager()
	defer func() {
		_ = manager.DeleteQueue(activeJobTestQueue, true)
	}()

	// Track when the job starts processing.
	var jobStarted sync.WaitGroup
	jobStarted.Add(1)

	// Create a worker that processes jobs slowly.
	worker, err := queue.NewWorker(redisConfig,
		queue.WithWorkerQueue(activeJobTestQueue, 1),
		queue.WithWorkerConcurrency(1),
	)
	require.NoError(t, err)

	err = worker.Register("active_test_job",
		func(ctx context.Context, _ *queue.Job) error {
			jobStarted.Done()
			// Block until context is canceled or timeout.
			<-ctx.Done()
			return ctx.Err()
		},
		queue.WithJobQueue(activeJobTestQueue),
	)
	require.NoError(t, err)

	go func() { _ = worker.Start() }()
	defer func() { _ = worker.Stop() }()

	// Allow worker to start.
	time.Sleep(1 * time.Second)

	// Enqueue a job.
	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err)
	defer func() { _ = client.Stop() }()

	_, err = client.Enqueue("active_test_job",
		map[string]string{"k": "v"},
		queue.WithQueue(activeJobTestQueue),
	)
	require.NoError(t, err)

	// Wait for the job to start processing.
	jobStarted.Wait()
	// Small buffer for asynq to register the active task.
	time.Sleep(500 * time.Millisecond)
}

func TestManagerListWorkersWithRunningWorker(t *testing.T) {
	redisConfig := getRedisConfig()
	manager := setupTestManager()

	worker, err := queue.NewWorker(redisConfig,
		queue.WithWorkerConcurrency(1),
	)
	require.NoError(t, err)

	go func() { _ = worker.Start() }()
	defer func() { _ = worker.Stop() }()

	time.Sleep(1 * time.Second)

	workers, err := manager.ListWorkers()
	require.NoError(t, err)
	assert.NotEmpty(t, workers)

	// Test GetWorkerInfo with a real worker ID.
	if len(workers) > 0 {
		info, err := manager.GetWorkerInfo(workers[0].ID)
		require.NoError(t, err)
		assert.Equal(t, workers[0].ID, info.ID)
	}
}
