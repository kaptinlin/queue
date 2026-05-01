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

// toJobState converts an asynq.TaskState to a JobState.
// It returns the mapped state and true if the mapping exists,
// or an empty JobState and false for unknown task states.
func toJobState(taskState asynq.TaskState) (JobState, bool) {
	switch taskState {
	case asynq.TaskStateActive:
		return StateActive, true
	case asynq.TaskStatePending:
		return StatePending, true
	case asynq.TaskStateScheduled:
		return StateScheduled, true
	case asynq.TaskStateRetry:
		return StateRetry, true
	case asynq.TaskStateArchived:
		return StateArchived, true
	case asynq.TaskStateCompleted:
		return StateCompleted, true
	case asynq.TaskStateAggregating:
		return StateAggregating, true
	default:
		return "", false
	}
}
