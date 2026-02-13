package tests

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/go-json-experiment/json"
	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJob(t *testing.T) {
	jobType := "testJob"
	payload := map[string]interface{}{"key": "value"}
	job := queue.NewJob(jobType, payload)

	assert.Equal(t, jobType, job.Type, "Job type should match")
	assert.True(t, reflect.DeepEqual(job.Payload, payload), "Job payload should match")
}

func TestJob_ConvertToAsynqTask(t *testing.T) {
	jobType := "testConversion"
	payload := map[string]interface{}{"key": "value"}
	job := queue.NewJob(jobType, payload)

	task, _, err := job.ConvertToAsynqTask()
	require.NoError(t, err, "ConvertToAsynqTask should not fail")

	assert.Equal(t, jobType, task.Type(), "Task type should match job type")

	var taskPayload map[string]interface{}
	err = json.Unmarshal(task.Payload(), &taskPayload)
	require.NoError(t, err, "json.Unmarshal should not fail")

	assert.True(t, reflect.DeepEqual(taskPayload, payload), "Task payload should match job payload")
}

func TestJob_DecodePayload(t *testing.T) {
	jobType := "testDecode"
	payload := struct{ Key string }{"value"}
	job := queue.NewJob(jobType, payload)

	var decodedPayload struct{ Key string }
	err := job.DecodePayload(&decodedPayload)
	require.NoError(t, err, "DecodePayload should not fail")

	assert.Equal(t, payload.Key, decodedPayload.Key, "Decoded payload key should match")
}

func TestJobOptions(t *testing.T) {
	now := time.Now()
	job := queue.NewJob("testOptions", nil,
		queue.WithDelay(10*time.Second),
		queue.WithMaxRetries(5),
		queue.WithQueue("customQueue"),
		queue.WithScheduleAt(&now),
		queue.WithRetention(24*time.Hour),
		queue.WithDeadline(&now),
	)

	assert.Equal(t, 10*time.Second, job.Options.Delay, "Delay should be 10s")
	assert.Equal(t, 5, job.Options.MaxRetries, "Max retries should be 5")
	assert.Equal(t, "customQueue", job.Options.Queue, "Queue should be 'customQueue'")
	assert.True(t, job.Options.ScheduleAt.Equal(now), "ScheduleAt should match")
	assert.Equal(t, 24*time.Hour, job.Options.Retention, "Retention should be 24h")
	assert.True(t, job.Options.Deadline.Equal(now), "Deadline should match")
}

type TestPayload struct {
	Name    string
	Age     int
	Hobbies []string
}

type NestedPayload struct {
	Data TestPayload
}

func TestJobPayloadBasicType(t *testing.T) {
	jobType := "testBasicPayload"
	payload := "This is a test string."

	job := queue.NewJob(jobType, payload)
	_, _, err := job.ConvertToAsynqTask()
	require.NoError(t, err, "Failed to convert job to task")

	var decodedPayload string
	err = job.DecodePayload(&decodedPayload)
	require.NoError(t, err, "Failed to decode payload")

	assert.Equal(t, payload, decodedPayload, "Decoded payload should match original")
}

// TestJobPayloadStruct with corrections.
func TestJobPayloadStruct(t *testing.T) {
	jobType := "testStructPayload"
	payload := TestPayload{Name: "John Doe", Age: 30, Hobbies: []string{"Reading", "Cycling"}}

	job := queue.NewJob(jobType, payload)
	task, _, err := job.ConvertToAsynqTask()
	require.NoError(t, err, "Failed to convert job to task")

	// Assuming task is used later in this function.
	_ = task

	var decodedPayload TestPayload
	err = job.DecodePayload(&decodedPayload)
	require.NoError(t, err, "Failed to decode payload")

	assert.True(t, reflect.DeepEqual(decodedPayload, payload), "Decoded payload should match original")
}

// TestJobPayloadNestedStruct with corrections.
func TestJobPayloadNestedStruct(t *testing.T) {
	jobType := "testNestedStructPayload"
	payload := NestedPayload{Data: TestPayload{Name: "Jane Doe", Age: 28, Hobbies: []string{"Skiing", "Photography"}}}

	job := queue.NewJob(jobType, payload)
	task, _, err := job.ConvertToAsynqTask()
	require.NoError(t, err, "Failed to convert job to task")

	// Assuming task is used later in this function.
	_ = task

	var decodedPayload NestedPayload
	err = job.DecodePayload(&decodedPayload)
	require.NoError(t, err, "Failed to decode payload")

	assert.True(t, reflect.DeepEqual(decodedPayload, payload), "Decoded payload should match original")
}
func TestWriteResultAndRetrieve(t *testing.T) {
	redisConfig := getRedisConfig()

	// Initialize the queue client
	client, err := queue.NewClient(redisConfig)
	require.NoError(t, err, "Failed to create client")
	defer func() {
		assert.NoError(t, client.Stop(), "Failed to stop client")
	}()

	// Initialize the worker to process jobs
	worker, err := queue.NewWorker(redisConfig)
	require.NoError(t, err, "Failed to create worker")

	// Prepare a WaitGroup for job completion synchronization
	var wg sync.WaitGroup

	// Define expected result
	expectedResult := map[string]interface{}{
		"status": "completed",
		"detail": "Job processed successfully",
	}

	// Define the job handler
	testJobHandler := func(ctx context.Context, job *queue.Job) error {
		defer wg.Done() // Signal job processing completion

		// Simulate job processing...

		// Write result to job
		if err := job.WriteResult(expectedResult); err != nil {
			return err
		}

		return nil
	}

	// Register the job type and handler with the worker
	err = worker.Register(testJobType, testJobHandler)
	require.NoError(t, err, "Failed to register job handler")

	// Start the worker in a separate goroutine
	go func() {
		if err := worker.Start(); err != nil {
			t.Errorf("Worker failed to start: %v", err)
		}
	}()
	defer func() {
		assert.NoError(t, worker.Stop(), "Failed to stop worker")
	}()

	// Enqueue the job
	payload := TestJobPayload{Message: "Test WriteResult"}
	job := queue.NewJob(testJobType, payload, queue.WithRetention(24*time.Hour))
	wg.Add(1)
	jobID, err := client.EnqueueJob(job)
	require.NoError(t, err, "Failed to enqueue job")

	// Wait for job processing to complete
	wg.Wait()

	// Allow some time for job result to be processed and stored
	time.Sleep(5 * time.Second)

	// Initialize manager to retrieve job information
	manager := setupTestManager()
	jobInfo, err := manager.GetJobInfo(queue.DefaultQueue, jobID)
	require.NoError(t, err, "Failed to get job info")

	// Deserialize the job result to verify it
	var result map[string]interface{}
	err = json.Unmarshal([]byte(*jobInfo.Result), &result)
	require.NoError(t, err, "Failed to unmarshal job result")

	// Assert that the result matches the expected result
	assert.Equal(t, expectedResult["status"], result["status"], "Job result status should match")
	assert.Equal(t, expectedResult["detail"], result["detail"], "Job result detail should match")
}
