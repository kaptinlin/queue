package queue

import "github.com/hibiken/asynq"

// JobState represents the state of a job in the queue.
type JobState string

const (
	// StateActive represents jobs that are currently being processed.
	StateActive JobState = "active"
	// StatePending represents jobs that are waiting to be processed.
	StatePending JobState = "pending"
	// StateRetry represents jobs that will be retried after a failure.
	StateRetry JobState = "retry"
	// StateArchived represents jobs that have been moved to the archive.
	StateArchived JobState = "archived"
	// StateCompleted represents jobs that have been completed successfully.
	StateCompleted JobState = "completed"
	// StateScheduled represents jobs that are scheduled to be run in the future.
	StateScheduled JobState = "scheduled"
	// StateAggregating represents jobs that are part of a batch or group waiting to be processed together.
	StateAggregating JobState = "aggregating"
)

// IsValidJobState checks if the provided job state is valid and supported.
func IsValidJobState(state JobState) bool {
	switch state {
	case StateActive, StatePending, StateRetry, StateArchived, StateCompleted, StateScheduled, StateAggregating:
		return true
	default:
		return false
	}
}

var taskStateToJobStateMap = map[asynq.TaskState]JobState{
	asynq.TaskStateActive:      StateActive,
	asynq.TaskStatePending:     StatePending,
	asynq.TaskStateScheduled:   StateScheduled,
	asynq.TaskStateRetry:       StateRetry,
	asynq.TaskStateArchived:    StateArchived,
	asynq.TaskStateCompleted:   StateCompleted,
	asynq.TaskStateAggregating: StateAggregating,
}

func toJobState(taskState asynq.TaskState) JobState {
	if jobState, exists := taskStateToJobStateMap[taskState]; exists {
		return jobState
	}
	return JobState("unknown")
}
