package queue

import "testing"

func mustNewJob(t testing.TB, jobType string, payload any, opts ...JobOption) *Job {
	t.Helper()

	job, err := NewJob(jobType, payload, opts...)
	if err != nil {
		t.Fatalf("NewJob() failed: %v", err)
	}
	return job
}

func mustNewHandler(t testing.TB, jobType string, handle HandlerFunc, opts ...HandlerOption) *Handler {
	t.Helper()

	handler, err := NewHandler(jobType, handle, opts...)
	if err != nil {
		t.Fatalf("NewHandler() failed: %v", err)
	}
	return handler
}
