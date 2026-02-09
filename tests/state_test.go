package tests

import (
	"testing"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
)

func TestIsValidJobState(t *testing.T) {
	tests := []struct {
		name  string
		state queue.JobState
		want  bool
	}{
		{name: "active", state: queue.StateActive, want: true},
		{name: "pending", state: queue.StatePending, want: true},
		{name: "retry", state: queue.StateRetry, want: true},
		{name: "archived", state: queue.StateArchived, want: true},
		{name: "completed", state: queue.StateCompleted, want: true},
		{name: "scheduled", state: queue.StateScheduled, want: true},
		{name: "aggregating", state: queue.StateAggregating, want: true},
		{name: "empty string", state: "", want: false},
		{name: "unknown", state: "unknown", want: false},
		{name: "invalid", state: "invalid", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := queue.IsValidJobState(tt.state)
			assert.Equal(t, tt.want, got,
				"IsValidJobState(%q) = %v, want %v",
				tt.state, got, tt.want)
		})
	}
}

func TestJobStateValues(t *testing.T) {
	// Verify that each state constant has the expected string value.
	assert.Equal(t, queue.JobState("active"), queue.StateActive)
	assert.Equal(t, queue.JobState("pending"), queue.StatePending)
	assert.Equal(t, queue.JobState("retry"), queue.StateRetry)
	assert.Equal(t, queue.JobState("archived"), queue.StateArchived)
	assert.Equal(t, queue.JobState("completed"), queue.StateCompleted)
	assert.Equal(t, queue.JobState("scheduled"), queue.StateScheduled)
	assert.Equal(t, queue.JobState("aggregating"), queue.StateAggregating)
}
