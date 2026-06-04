package tests

import (
	"context"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
)

type testHelper interface {
	Helper()
	Fatalf(format string, args ...any)
}

func newJob(t testHelper, jobType string, payload any, opts ...queue.JobOption) *queue.Job {
	t.Helper()

	job, err := queue.NewJob(jobType, payload, opts...)
	if err != nil {
		t.Fatalf("NewJob() failed: %v", err)
	}
	return job
}

func newHandler(t testHelper, jobType string, handle queue.HandlerFunc, opts ...queue.HandlerOption) *queue.Handler {
	t.Helper()

	handler, err := queue.NewHandler(jobType, handle, opts...)
	if err != nil {
		t.Fatalf("NewHandler() failed: %v", err)
	}
	return handler
}

func runWorker(t testing.TB, worker *queue.Worker) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- worker.Run(ctx)
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Worker.Run() failed: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Error("timed out waiting for worker shutdown")
		}
	})
}

func runScheduler(t testing.TB, scheduler *queue.Scheduler) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- scheduler.Run(ctx)
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Scheduler.Run() failed: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Error("timed out waiting for scheduler shutdown")
		}
	})
}

var _ testHelper = (*testing.T)(nil)
var _ testHelper = (*testing.B)(nil)
