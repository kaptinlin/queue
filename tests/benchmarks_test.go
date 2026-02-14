package tests

import (
	"context"
	"testing"

	"github.com/kaptinlin/queue"
)

func BenchmarkJobCreation(b *testing.B) {
	payload := map[string]any{"key": "value", "count": 123}

	for b.Loop() {
		_ = queue.NewJob("test_job", payload)
	}
}

func BenchmarkJobCreationWithOptions(b *testing.B) {
	payload := map[string]any{"key": "value", "count": 123}

	for b.Loop() {
		_ = queue.NewJob("test_job", payload,
			queue.WithQueue("critical"),
			queue.WithMaxRetries(3),
		)
	}
}

func BenchmarkJobConvertToAsynqTask(b *testing.B) {
	payload := map[string]any{"key": "value", "count": 123}
	job := queue.NewJob("test_job", payload, queue.WithQueue("default"))

	for b.Loop() {
		_, _, err := job.ConvertToAsynqTask()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHandlerCreation(b *testing.B) {
	handlerFunc := func(_ context.Context, _ *queue.Job) error {
		return nil
	}

	for b.Loop() {
		_ = queue.NewHandler("test_job", handlerFunc)
	}
}

func BenchmarkHandlerProcess(b *testing.B) {
	handler := queue.NewHandler("test_job", func(_ context.Context, _ *queue.Job) error {
		return nil
	})

	job := queue.NewJob("test_job", map[string]any{"key": "value"})
	ctx := context.Background()

	for b.Loop() {
		err := handler.Process(ctx, job)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJobDecodePayload(b *testing.B) {
	type TestPayload struct {
		Key   string `json:"key"`
		Count int    `json:"count"`
	}

	payload := map[string]any{"key": "value", "count": 123}
	job := queue.NewJob("test_job", payload)

	for b.Loop() {
		var decoded TestPayload
		err := job.DecodePayload(&decoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkClientEnqueue(b *testing.B) {
	redisConfig := getRedisConfig()
	client, err := queue.NewClient(redisConfig)
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	defer func() { _ = client.Stop() }()

	payload := map[string]any{"key": "value"}

	for b.Loop() {
		_, err := client.Enqueue("benchmark_job", payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkClientEnqueueJob(b *testing.B) {
	redisConfig := getRedisConfig()
	client, err := queue.NewClient(redisConfig)
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	defer func() { _ = client.Stop() }()

	for b.Loop() {
		job := queue.NewJob("benchmark_job", map[string]any{"key": "value"})
		_, err := client.EnqueueJob(job)
		if err != nil {
			b.Fatal(err)
		}
	}
}
