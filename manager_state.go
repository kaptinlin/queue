package queue

import "github.com/hibiken/asynq"

type managerTaskListFunc func(*asynq.Inspector, string, Page) ([]*asynq.TaskInfo, error)
type managerTaskCountFunc func(*asynq.Inspector, string) (int, error)

type managerStateRule struct {
	list       managerTaskListFunc
	run        managerTaskCountFunc
	archive    managerTaskCountFunc
	delete     managerTaskCountFunc
	listErr    error
	runErr     error
	archiveErr error
	deleteErr  error
}

var managerStateRules = map[JobState]managerStateRule{
	StatePending: {
		list: func(inspector *asynq.Inspector, queue string, page Page) ([]*asynq.TaskInfo, error) {
			return inspector.ListPendingTasks(queue, managerPageOptions(page)...)
		},
		runErr: ErrOperationNotSupported,
		archive: func(inspector *asynq.Inspector, queue string) (int, error) {
			return inspector.ArchiveAllPendingTasks(queue)
		},
		delete: func(inspector *asynq.Inspector, queue string) (int, error) {
			return inspector.DeleteAllPendingTasks(queue)
		},
	},
	StateScheduled: {
		list: func(inspector *asynq.Inspector, queue string, page Page) ([]*asynq.TaskInfo, error) {
			return inspector.ListScheduledTasks(queue, managerPageOptions(page)...)
		},
		run: func(inspector *asynq.Inspector, queue string) (int, error) {
			return inspector.RunAllScheduledTasks(queue)
		},
		archive: func(inspector *asynq.Inspector, queue string) (int, error) {
			return inspector.ArchiveAllScheduledTasks(queue)
		},
		delete: func(inspector *asynq.Inspector, queue string) (int, error) {
			return inspector.DeleteAllScheduledTasks(queue)
		},
	},
	StateRetry: {
		list: func(inspector *asynq.Inspector, queue string, page Page) ([]*asynq.TaskInfo, error) {
			return inspector.ListRetryTasks(queue, managerPageOptions(page)...)
		},
		run: func(inspector *asynq.Inspector, queue string) (int, error) {
			return inspector.RunAllRetryTasks(queue)
		},
		archive: func(inspector *asynq.Inspector, queue string) (int, error) {
			return inspector.ArchiveAllRetryTasks(queue)
		},
		delete: func(inspector *asynq.Inspector, queue string) (int, error) {
			return inspector.DeleteAllRetryTasks(queue)
		},
	},
	StateArchived: {
		list: func(inspector *asynq.Inspector, queue string, page Page) ([]*asynq.TaskInfo, error) {
			return inspector.ListArchivedTasks(queue, managerPageOptions(page)...)
		},
		run: func(inspector *asynq.Inspector, queue string) (int, error) {
			return inspector.RunAllArchivedTasks(queue)
		},
		archiveErr: ErrOperationNotSupported,
		delete: func(inspector *asynq.Inspector, queue string) (int, error) {
			return inspector.DeleteAllArchivedTasks(queue)
		},
	},
	StateCompleted: {
		list: func(inspector *asynq.Inspector, queue string, page Page) ([]*asynq.TaskInfo, error) {
			return inspector.ListCompletedTasks(queue, managerPageOptions(page)...)
		},
		runErr:     ErrOperationNotSupported,
		archiveErr: ErrOperationNotSupported,
		delete: func(inspector *asynq.Inspector, queue string) (int, error) {
			return inspector.DeleteAllCompletedTasks(queue)
		},
	},
	StateActive: {
		runErr:     ErrOperationNotSupported,
		archiveErr: ErrArchivingActiveJobs,
		deleteErr:  ErrOperationNotSupported,
	},
	StateAggregating: {
		listErr:    ErrGroupRequiredForAggregation,
		runErr:     ErrGroupRequiredForAggregation,
		archiveErr: ErrGroupRequiredForAggregation,
		deleteErr:  ErrGroupRequiredForAggregation,
	},
}

func managerPageOptions(page Page) []asynq.ListOption {
	return []asynq.ListOption{asynq.PageSize(page.Size), asynq.Page(page.Number)}
}

func managerRuleForState(state JobState) (managerStateRule, error) {
	if !IsValidJobState(state) {
		return managerStateRule{}, ErrInvalidJobState
	}

	rule, ok := managerStateRules[state]
	if !ok {
		return managerStateRule{}, ErrUnsupportedJobStateForAction
	}
	return rule, nil
}

func managerUnsupportedAction(err error) error {
	if err != nil {
		return err
	}
	return ErrOperationNotSupported
}
