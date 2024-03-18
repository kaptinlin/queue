package tests

import (
	"context"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"golang.org/x/time/rate"
)

func TestNewHandler(t *testing.T) {
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error { return nil })
	if handler.JobType != "test" {
		t.Errorf("NewHandler() JobType = %v, want %v", handler.JobType, "test")
	}
}

func TestHandler_WithRateLimiter(t *testing.T) {
	limiter := rate.NewLimiter(1, 1)
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error { return nil }, queue.WithRateLimiter(limiter))
	if handler.Limiter != limiter {
		t.Errorf("WithRateLimiter() did not set the expected limiter")
	}
}

func TestHandler_WithJobTimeout(t *testing.T) {
	timeout := 5 * time.Second
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error { return nil }, queue.WithJobTimeout(timeout))
	if handler.JobTimeout != timeout {
		t.Errorf("WithJobTimeout() = %v, want %v", handler.JobTimeout, timeout)
	}
}

func TestHandler_WithJobQueue(t *testing.T) {
	queueName := "customQueue"
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error { return nil }, queue.WithJobQueue(queueName))
	if handler.JobQueue != queueName {
		t.Errorf("WithJobQueue() = %v, want %v", handler.JobQueue, queueName)
	}
}

func TestHandler_WithRetryDelayFunc(t *testing.T) {
	customFuncCalled := false
	customFunc := func(attempt int, err error) time.Duration {
		customFuncCalled = true
		return 0
	}
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error { return nil }, queue.WithRetryDelayFunc(customFunc))
	handler.RetryDelayFunc(0, nil)
	if !customFuncCalled {
		t.Errorf("WithRetryDelayFunc() did not set the expected function")
	}
}

func TestHandler_Process_HandleExecuted(t *testing.T) {
	handleExecuted := false
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error {
		handleExecuted = true
		return nil
	})

	job := &queue.Job{Type: "test", Payload: "data"}
	if err := handler.Process(context.Background(), job); err != nil {
		t.Errorf("Process() unexpected error: %v", err)
	}
	if !handleExecuted {
		t.Errorf("Process() did not execute handle function")
	}
}

func TestHandler_Process_WithTimeout(t *testing.T) {
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error {
		time.Sleep(2 * time.Second) // Simulate work that exceeds the timeout
		return nil
	}, queue.WithJobTimeout(1*time.Second))

	job := &queue.Job{Type: "test", Payload: "data"}
	err := handler.Process(context.Background(), job)
	if err == nil || err.Error() != "job processing exceeded timeout: context deadline exceeded" {
		t.Errorf("Process() expected timeout error, got: %v", err)
	}
}

func TestHandler_Process_WithRateLimiter(t *testing.T) {
	limiter := rate.NewLimiter(rate.Every(10*time.Second), 1) // Allow only 1 operation per 10 seconds
	handler := queue.NewHandler("test", func(ctx context.Context, job *queue.Job) error {
		return nil
	}, queue.WithRateLimiter(limiter))

	// The first call should pass due to the rate limiter allowance
	job := &queue.Job{Type: "test", Payload: "data"}
	if err := handler.Process(context.Background(), job); err != nil {
		t.Errorf("Process() unexpected error: %v", err)
	}

	// The second call should be limited
	err := handler.Process(context.Background(), job)
	if _, ok := err.(*queue.ErrRateLimit); !ok {
		t.Errorf("Process() expected ErrRateLimit error, got: %v", err)
	}
}
