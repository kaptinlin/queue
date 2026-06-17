package queue

import "github.com/hibiken/asynq"

func toJobInfoList(tasks []*asynq.TaskInfo) []*JobInfo {
	jobInfos := make([]*JobInfo, len(tasks))
	for i, task := range tasks {
		jobInfos[i] = toJobInfo(task, nil)
	}
	return jobInfos
}

func requireGroup(group string) error {
	if group == "" {
		return ErrGroupRequiredForAggregation
	}
	return nil
}

// ListJobs lists jobs selected by a query.
func (m *Manager) ListJobs(query JobQuery) ([]*JobInfo, error) {
	page, err := query.Page.normalize()
	if err != nil {
		return nil, err
	}

	if query.State == StateActive {
		return m.listActiveJobs(query.Queue, page)
	}
	if query.State == StateAggregating {
		return m.ListAggregatingJobs(query.Queue, query.Group, page)
	}

	rule, err := managerRuleForState(query.State)
	if err != nil {
		return nil, err
	}
	if rule.list == nil {
		return nil, managerUnsupportedAction(rule.listErr)
	}

	tasks, err := rule.list(m.inspector, query.Queue, page)
	if err != nil {
		return nil, mapManagerError(err)
	}
	return toJobInfoList(tasks), nil
}

// ListAggregatingJobs lists aggregating jobs in a specified queue and group.
func (m *Manager) ListAggregatingJobs(queue, group string, page Page) ([]*JobInfo, error) {
	if err := requireGroup(group); err != nil {
		return nil, err
	}
	normalized, err := page.normalize()
	if err != nil {
		return nil, err
	}

	tasks, err := m.inspector.ListAggregatingTasks(queue, group, managerPageOptions(normalized)...)
	if err != nil {
		return nil, mapManagerError(err)
	}
	return toJobInfoList(tasks), nil
}

// ListActiveJobs lists active (currently processing) jobs for a given queue.
func (m *Manager) ListActiveJobs(queue string, page Page) ([]*JobInfo, error) {
	normalized, err := page.normalize()
	if err != nil {
		return nil, err
	}
	return m.listActiveJobs(queue, normalized)
}

func (m *Manager) listActiveJobs(queue string, page Page) ([]*JobInfo, error) {
	tasks, err := m.inspector.ListActiveTasks(queue, managerPageOptions(page)...)
	if err != nil {
		return nil, mapManagerError(err)
	}

	servers, err := m.inspector.Servers()
	if err != nil {
		return nil, mapManagerError(err)
	}

	workerInfoMap := make(map[string]*asynq.WorkerInfo)
	for _, server := range servers {
		for _, worker := range server.ActiveWorkers {
			if worker.Queue == queue {
				workerInfoMap[worker.TaskID] = worker
			}
		}
	}

	jobInfos := make([]*JobInfo, len(tasks))
	for i, task := range tasks {
		wi := workerInfoMap[task.ID]
		jobInfos[i] = toJobInfo(task, wi)
	}
	return jobInfos, nil
}

// JobInfo retrieves information for a single job using its ID and queue name.
func (m *Manager) JobInfo(queue, jobID string) (*JobInfo, error) {
	taskInfo, err := m.taskInfo(queue, jobID)
	if err != nil {
		return nil, err
	}
	return toJobInfo(taskInfo, nil), nil
}

// JobPayload retrieves the raw encoded payload for a job.
func (m *Manager) JobPayload(queue, jobID string) ([]byte, error) {
	taskInfo, err := m.taskInfo(queue, jobID)
	if err != nil {
		return nil, err
	}
	return append([]byte{}, taskInfo.Payload...), nil
}

// JobResult retrieves the raw encoded result for a completed retained job.
func (m *Manager) JobResult(queue, jobID string) ([]byte, error) {
	taskInfo, err := m.taskInfo(queue, jobID)
	if err != nil {
		return nil, err
	}
	if taskInfo.Result == nil {
		return nil, ErrJobResultNotFound
	}
	return append([]byte{}, taskInfo.Result...), nil
}

// RunJob triggers immediate execution of a job with the specified ID.
func (m *Manager) RunJob(queue, jobID string) error {
	return mapManagerError(m.inspector.RunTask(queue, jobID))
}

// RunJobsByState triggers all jobs in a specified queue and state to run immediately.
func (m *Manager) RunJobsByState(queue string, state JobState) (int, error) {
	rule, err := managerRuleForState(state)
	if err != nil {
		return 0, err
	}
	if rule.run == nil {
		return 0, managerUnsupportedAction(rule.runErr)
	}

	count, err := rule.run(m.inspector, queue)
	return count, mapManagerError(err)
}

// BatchRunJobs triggers immediate execution of multiple jobs identified by their IDs.
func (m *Manager) BatchRunJobs(queue string, jobIDs []string) (BatchJobResult, error) {
	return batchJobOperation(jobIDs, func(jobID string) error {
		return m.RunJob(queue, jobID)
	})
}

// ArchiveJob moves a job with the specified ID to the archive.
func (m *Manager) ArchiveJob(queue, jobID string) error {
	return mapManagerError(m.inspector.ArchiveTask(queue, jobID))
}

// ArchiveJobsByState archives all jobs in a specified queue based on their state.
func (m *Manager) ArchiveJobsByState(queue string, state JobState) (int, error) {
	rule, err := managerRuleForState(state)
	if err != nil {
		return 0, err
	}
	if rule.archive == nil {
		return 0, managerUnsupportedAction(rule.archiveErr)
	}

	count, err := rule.archive(m.inspector, queue)
	return count, mapManagerError(err)
}

// BatchArchiveJobs archives multiple jobs identified by their IDs.
func (m *Manager) BatchArchiveJobs(queue string, jobIDs []string) (BatchJobResult, error) {
	return batchJobOperation(jobIDs, func(jobID string) error {
		return m.ArchiveJob(queue, jobID)
	})
}

// CancelJob cancels a job with the specified ID.
func (m *Manager) CancelJob(jobID string) error {
	return mapManagerError(m.inspector.CancelProcessing(jobID))
}

type activeTaskListFunc func(page int) ([]*asynq.TaskInfo, error)
type activeTaskCancelFunc func(string) error

func cancelActiveJobs(size int, list activeTaskListFunc, cancel activeTaskCancelFunc) (int, error) {
	var totalCount int

	for {
		tasks, err := list(1)
		if err != nil {
			return totalCount, err
		}

		if len(tasks) == 0 {
			return totalCount, nil
		}

		for _, task := range tasks {
			if err := cancel(task.ID); err != nil {
				return totalCount, err
			}
			totalCount++
		}

		if len(tasks) < size {
			return totalCount, nil
		}
	}
}

// CancelActiveJobs cancels active jobs in batches, always rereading from the first page.
func (m *Manager) CancelActiveJobs(queue string, batchSize int) (int, error) {
	size, err := normalizeBatchSize(batchSize)
	if err != nil {
		return 0, err
	}

	return cancelActiveJobs(size,
		func(page int) ([]*asynq.TaskInfo, error) {
			tasks, err := m.inspector.ListActiveTasks(queue, managerPageOptions(Page{Size: size, Number: page})...)
			if err != nil {
				return nil, mapManagerError(err)
			}
			return tasks, nil
		},
		m.CancelJob,
	)
}

// BatchCancelJobs cancels multiple jobs identified by their IDs.
func (m *Manager) BatchCancelJobs(jobIDs []string) (BatchJobResult, error) {
	return batchJobOperation(jobIDs, func(jobID string) error {
		return m.CancelJob(jobID)
	})
}

// DeleteJob deletes a job with the specified ID from its queue.
func (m *Manager) DeleteJob(queue, jobID string) error {
	return mapManagerError(m.inspector.DeleteTask(queue, jobID))
}

// DeleteJobsByState deletes all jobs in a specified queue based on their state.
func (m *Manager) DeleteJobsByState(queue string, state JobState) (int, error) {
	rule, err := managerRuleForState(state)
	if err != nil {
		return 0, err
	}
	if rule.delete == nil {
		return 0, managerUnsupportedAction(rule.deleteErr)
	}

	count, err := rule.delete(m.inspector, queue)
	return count, mapManagerError(err)
}

// BatchDeleteJobs deletes multiple jobs identified by their IDs.
func (m *Manager) BatchDeleteJobs(queue string, jobIDs []string) (BatchJobResult, error) {
	return batchJobOperation(jobIDs, func(jobID string) error {
		return m.DeleteJob(queue, jobID)
	})
}

// RunAggregatingJobs triggers all aggregating jobs to run immediately in a specified queue and group.
func (m *Manager) RunAggregatingJobs(queue, group string) (int, error) {
	if err := requireGroup(group); err != nil {
		return 0, err
	}
	count, err := m.inspector.RunAllAggregatingTasks(queue, group)
	return count, mapManagerError(err)
}

// ArchiveAggregatingJobs archives all aggregating jobs in a specified queue and group.
func (m *Manager) ArchiveAggregatingJobs(queue, group string) (int, error) {
	if err := requireGroup(group); err != nil {
		return 0, err
	}
	count, err := m.inspector.ArchiveAllAggregatingTasks(queue, group)
	return count, mapManagerError(err)
}

// DeleteAggregatingJobs deletes all aggregating tasks in a specified queue and group.
func (m *Manager) DeleteAggregatingJobs(queue, group string) (int, error) {
	if err := requireGroup(group); err != nil {
		return 0, err
	}
	count, err := m.inspector.DeleteAllAggregatingTasks(queue, group)
	return count, mapManagerError(err)
}

func (m *Manager) taskInfo(queue, jobID string) (*asynq.TaskInfo, error) {
	taskInfo, err := m.inspector.GetTaskInfo(queue, jobID)
	if err != nil {
		return nil, mapManagerError(err)
	}
	return taskInfo, nil
}
