package tests

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
)

func TestNewJob(t *testing.T) {
	jobType := "testJob"
	payload := map[string]interface{}{"key": "value"}
	job := queue.NewJob(jobType, payload)

	if job.Type != jobType {
		t.Errorf("expected job type to be %s, got %s", jobType, job.Type)
	}

	if !reflect.DeepEqual(job.Payload, payload) {
		t.Errorf("expected job payload to be %+v, got %+v", payload, job.Payload)
	}
}

func TestJob_ConvertToAsynqTask(t *testing.T) {
	jobType := "testConversion"
	payload := map[string]interface{}{"key": "value"}
	job := queue.NewJob(jobType, payload)

	task, _, err := job.ConvertToAsynqTask()
	if err != nil {
		t.Fatalf("ConvertToAsynqTask failed: %v", err)
	}

	if task.Type() != jobType {
		t.Errorf("expected task type to be %s, got %s", jobType, task.Type())
	}

	var taskPayload map[string]interface{}
	if err := json.Unmarshal(task.Payload(), &taskPayload); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(taskPayload, payload) {
		t.Errorf("expected task payload to be %+v, got %+v", payload, taskPayload)
	}
}

func TestJob_DecodePayload(t *testing.T) {
	jobType := "testDecode"
	payload := struct{ Key string }{"value"}
	job := queue.NewJob(jobType, payload)

	var decodedPayload struct{ Key string }
	if err := job.DecodePayload(&decodedPayload); err != nil {
		t.Fatalf("DecodePayload failed: %v", err)
	}

	if decodedPayload.Key != payload.Key {
		t.Errorf("expected decoded payload key to be %s, got %s", payload.Key, decodedPayload.Key)
	}
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

	if job.Options.Delay != 10*time.Second {
		t.Errorf("expected delay to be 10s, got %v", job.Options.Delay)
	}

	if job.Options.MaxRetries != 5 {
		t.Errorf("expected max retries to be 5, got %d", job.Options.MaxRetries)
	}

	if job.Options.Queue != "customQueue" {
		t.Errorf("expected queue to be 'customQueue', got '%s'", job.Options.Queue)
	}

	if !job.Options.ScheduleAt.Equal(now) {
		t.Errorf("expected schedule at to be %v, got %v", now, job.Options.ScheduleAt)
	}

	if job.Options.Retention != 24*time.Hour {
		t.Errorf("expected retention to be 24h, got %v", job.Options.Retention)
	}

	if !job.Options.Deadline.Equal(now) {
		t.Errorf("expected deadline to be %v, got %v", now, job.Options.Deadline)
	}
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
	if err != nil {
		t.Fatalf("Failed to convert job to task: %v", err)
	}

	var decodedPayload string
	if err := job.DecodePayload(&decodedPayload); err != nil {
		t.Fatalf("Failed to decode payload: %v", err)
	}

	if decodedPayload != payload {
		t.Errorf("Expected payload to be %v, got %v", payload, decodedPayload)
	}
}

// TestJobPayloadStruct with corrections.
func TestJobPayloadStruct(t *testing.T) {
	jobType := "testStructPayload"
	payload := TestPayload{Name: "John Doe", Age: 30, Hobbies: []string{"Reading", "Cycling"}}

	job := queue.NewJob(jobType, payload)
	task, _, err := job.ConvertToAsynqTask()
	if err != nil {
		t.Fatalf("Failed to convert job to task: %v", err)
	}

	// Assuming task is used later in this function.
	_ = task

	var decodedPayload TestPayload
	if err := job.DecodePayload(&decodedPayload); err != nil {
		t.Fatalf("Failed to decode payload: %v", err)
	}

	if !reflect.DeepEqual(decodedPayload, payload) {
		t.Errorf("Expected payload to be %+v, got %+v", payload, decodedPayload)
	}
}

// TestJobPayloadNestedStruct with corrections.
func TestJobPayloadNestedStruct(t *testing.T) {
	jobType := "testNestedStructPayload"
	payload := NestedPayload{Data: TestPayload{Name: "Jane Doe", Age: 28, Hobbies: []string{"Skiing", "Photography"}}}

	job := queue.NewJob(jobType, payload)
	task, _, err := job.ConvertToAsynqTask()
	if err != nil {
		t.Fatalf("Failed to convert job to task: %v", err)
	}

	// Assuming task is used later in this function.
	_ = task

	var decodedPayload NestedPayload
	if err := job.DecodePayload(&decodedPayload); err != nil {
		t.Fatalf("Failed to decode payload: %v", err)
	}

	if !reflect.DeepEqual(decodedPayload, payload) {
		t.Errorf("Expected payload to be %+v, got %+v", payload, decodedPayload)
	}
}
func TestWriteResultAndRetrieve(t *testing.T) {
	redisConfig := getRedisConfig()

	// Initialize the queue client
	client, err := queue.NewClient(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Stop()

	// Initialize the worker to process jobs
	worker, err := queue.NewWorker(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}

	// Prepare a WaitGroup for job completion synchronization
	var wg sync.WaitGroup
	wg.Add(1)

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
	if err := worker.Register(testJobType, testJobHandler); err != nil {
		t.Fatalf("Failed to register job handler: %v", err)
	}

	// Start the worker in a separate goroutine
	go func() {
		if err := worker.Start(); err != nil {
			t.Errorf("Worker failed to start: %v", err)
		}
	}()
	defer worker.Stop()

	// Enqueue the job
	payload := TestJobPayload{Message: "Test WriteResult"}
	job := queue.NewJob(testJobType, payload, queue.WithRetention(24*time.Hour))
	jobID, err := client.EnqueueJob(job)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Wait for job processing to complete
	wg.Wait()

	// Allow some time for job result to be processed and stored
	time.Sleep(5 * time.Second)

	// Initialize manager to retrieve job information
	manager, err := setupTestManager()
	if err != nil {
		t.Fatalf("Failed to setup manager: %v", err)
	}
	jobInfo, err := manager.GetJobInfo(queue.DefaultQueue, jobID)
	if err != nil {
		t.Fatalf("Failed to get job info: %v", err)
	}

	// Deserialize the job result to verify it
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(*jobInfo.Result), &result); err != nil {
		t.Fatalf("Failed to unmarshal job result: %v", err)
	}

	// Assert that the result matches the expected result
	if result["status"] != expectedResult["status"] || result["detail"] != expectedResult["detail"] {
		t.Errorf("Job result did not match expected result. Got %v, want %v", result, expectedResult)
	}
}
