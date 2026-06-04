package queue

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// Manager provides operational inspection and management APIs for queues and jobs.
type Manager struct {
	client    redis.UniversalClient
	inspector *asynq.Inspector
}

// NewManager creates a Manager for operational inspection and state changes.
func NewManager(client redis.UniversalClient, inspector *asynq.Inspector) (*Manager, error) {
	if client == nil {
		return nil, ErrInvalidManagerClient
	}
	if inspector == nil {
		return nil, ErrInvalidManagerInspector
	}
	return &Manager{
		client:    client,
		inspector: inspector,
	}, nil
}

// BatchJobError records one failed item from a batch job operation.
type BatchJobError struct {
	JobID string `json:"job_id"`
	Err   error  `json:"-"`
}

// Error implements the error interface.
func (e BatchJobError) Error() string {
	return "job " + e.JobID + ": " + e.Err.Error()
}

// Unwrap returns the underlying cause.
func (e BatchJobError) Unwrap() error {
	return e.Err
}

// BatchJobResult records the successful and failed items from a batch job operation.
type BatchJobResult struct {
	Succeeded []string        `json:"succeeded"`
	Failed    []BatchJobError `json:"failed"`
}

// Err returns all per-job failures joined into one error.
func (r BatchJobResult) Err() error {
	if len(r.Failed) == 0 {
		return nil
	}

	errs := make([]error, len(r.Failed))
	for i, failure := range r.Failed {
		errs[i] = failure
	}
	return errors.Join(errs...)
}

type managerTaskListFunc func(*asynq.Inspector, string, int, int) ([]*asynq.TaskInfo, error)
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
		list: func(inspector *asynq.Inspector, queue string, size, page int) ([]*asynq.TaskInfo, error) {
			return inspector.ListPendingTasks(queue, managerPageOptions(size, page)...)
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
		list: func(inspector *asynq.Inspector, queue string, size, page int) ([]*asynq.TaskInfo, error) {
			return inspector.ListScheduledTasks(queue, managerPageOptions(size, page)...)
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
		list: func(inspector *asynq.Inspector, queue string, size, page int) ([]*asynq.TaskInfo, error) {
			return inspector.ListRetryTasks(queue, managerPageOptions(size, page)...)
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
		list: func(inspector *asynq.Inspector, queue string, size, page int) ([]*asynq.TaskInfo, error) {
			return inspector.ListArchivedTasks(queue, managerPageOptions(size, page)...)
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
		list: func(inspector *asynq.Inspector, queue string, size, page int) ([]*asynq.TaskInfo, error) {
			return inspector.ListCompletedTasks(queue, managerPageOptions(size, page)...)
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

func managerPageOptions(size, page int) []asynq.ListOption {
	return []asynq.ListOption{asynq.PageSize(size), asynq.Page(page)}
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

// ListWorkers retrieves information about all Asynq servers (workers) and the jobs they are currently processing.
func (m *Manager) ListWorkers() ([]*WorkerInfo, error) {
	servers, err := m.inspector.Servers()
	if err != nil {
		return nil, mapManagerError(err)
	}

	workers := make([]*WorkerInfo, len(servers))
	for i, server := range servers {
		workers[i] = toWorkerInfo(server)
	}
	return workers, nil
}

// WorkerInfo retrieves detailed information about a single worker using its ID.
func (m *Manager) WorkerInfo(workerID string) (*WorkerInfo, error) {
	servers, err := m.inspector.Servers()
	if err != nil {
		return nil, mapManagerError(err)
	}

	for _, server := range servers {
		if server.ID == workerID {
			return toWorkerInfo(server), nil
		}
	}

	return nil, ErrWorkerNotFound
}

// ListQueues lists all queue names.
func (m *Manager) ListQueues() ([]*QueueInfo, error) {
	queues, err := m.inspector.Queues()
	if err != nil {
		return nil, mapManagerError(err)
	}

	snapshots := make([]*QueueInfo, len(queues))
	for i, queue := range queues {
		qinfo, err := m.inspector.GetQueueInfo(queue)
		if err != nil {
			return nil, mapManagerError(err)
		}
		snapshots[i] = toQueueInfo(qinfo)
	}
	return snapshots, nil
}

// QueueInfo retrieves detailed information about a queue.
func (m *Manager) QueueInfo(queueName string) (*QueueInfo, error) {
	qinfo, err := m.inspector.GetQueueInfo(queueName)
	if err != nil {
		return nil, mapManagerError(err)
	}
	return toQueueInfo(qinfo), nil
}

// ListQueueStats lists statistics for a queue over the past n days.
func (m *Manager) ListQueueStats(queueName string, days int) ([]*QueueDailyStats, error) {
	dstats, err := m.inspector.History(queueName, days)
	if err != nil {
		return nil, mapManagerError(err)
	}

	dailyStats := make([]*QueueDailyStats, len(dstats))
	for i, d := range dstats {
		dailyStats[i] = toQueueDailyStats(d)
	}

	return dailyStats, nil
}

// DeleteQueue deletes a queue by its name.
func (m *Manager) DeleteQueue(queueName string, force bool) error {
	return mapManagerError(m.inspector.DeleteQueue(queueName, force))
}

// PauseQueue pauses a queue by its name.
func (m *Manager) PauseQueue(queueName string) error {
	if err := m.inspector.PauseQueue(queueName); err != nil {
		return mapManagerError(err)
	}
	return nil
}

// ResumeQueue resumes a paused queue by its name.
func (m *Manager) ResumeQueue(queueName string) error {
	if err := m.inspector.UnpauseQueue(queueName); err != nil {
		return mapManagerError(err)
	}
	return nil
}

// ListJobsByState lists jobs in a specified queue filtered by their state.
func (m *Manager) ListJobsByState(queue string, state JobState, size, page int) ([]*JobInfo, error) {
	if state == StateActive {
		return m.ListActiveJobs(queue, size, page)
	}

	rule, err := managerRuleForState(state)
	if err != nil {
		return nil, err
	}
	if rule.list == nil {
		return nil, managerUnsupportedAction(rule.listErr)
	}

	tasks, err := rule.list(m.inspector, queue, size, page)
	if err != nil {
		return nil, mapManagerError(err)
	}
	return toJobInfoList(tasks), nil
}

// ListAggregatingJobs lists aggregating jobs in a specified queue and group.
func (m *Manager) ListAggregatingJobs(queue, group string, size, page int) ([]*JobInfo, error) {
	if err := requireGroup(group); err != nil {
		return nil, err
	}

	tasks, err := m.inspector.ListAggregatingTasks(queue, group, managerPageOptions(size, page)...)
	if err != nil {
		return nil, mapManagerError(err)
	}
	return toJobInfoList(tasks), nil
}

// ListActiveJobs lists active (currently processing) jobs for a given queue.
func (m *Manager) ListActiveJobs(queue string, size, page int) ([]*JobInfo, error) {
	tasks, err := m.inspector.ListActiveTasks(queue, managerPageOptions(size, page)...)
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

func batchJobOperation(jobIDs []string, operation func(string) error) (BatchJobResult, error) {
	result := BatchJobResult{
		Succeeded: make([]string, 0, len(jobIDs)),
		Failed:    make([]BatchJobError, 0),
	}

	for _, jobID := range jobIDs {
		if err := operation(jobID); err != nil {
			result.Failed = append(result.Failed, BatchJobError{JobID: jobID, Err: err})
			continue
		}
		result.Succeeded = append(result.Succeeded, jobID)
	}

	return result, result.Err()
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

// CancelActiveJobs cancels all active jobs in the specified queue.
func (m *Manager) CancelActiveJobs(queue string, size, page int) (int, error) {
	var totalCount int

	for {
		tasks, err := m.inspector.ListActiveTasks(queue, managerPageOptions(size, page)...)
		if err != nil {
			return totalCount, mapManagerError(err)
		}

		if len(tasks) == 0 {
			break
		}

		for _, task := range tasks {
			if err := m.CancelJob(task.ID); err != nil {
				return totalCount, err
			}
			totalCount++
		}

		if len(tasks) < size {
			break
		}

		page++
	}

	return totalCount, nil
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

// RedisInfo retrieves information from the Redis server or cluster.
func (m *Manager) RedisInfo(ctx context.Context) (*RedisInfo, error) {
	if ctx == nil {
		return nil, ErrInvalidContext
	}

	switch client := m.client.(type) {
	case *redis.ClusterClient:
		return m.getRedisClusterInfo(ctx, client)
	case *redis.Client:
		return getRedisStandardInfo(ctx, client)
	default:
		return nil, ErrRedisClientNotSupported
	}
}

func getRedisStandardInfo(ctx context.Context, client *redis.Client) (*RedisInfo, error) {
	rawInfo, err := client.Info(ctx, "all").Result()
	if err != nil {
		return nil, mapRedisInfoError(err)
	}
	info := parseRedisInfo(rawInfo)
	return &RedisInfo{
		Address:   client.Options().Addr,
		Info:      info,
		RawInfo:   rawInfo,
		IsCluster: false,
	}, nil
}

func (m *Manager) getRedisClusterInfo(ctx context.Context, client *redis.ClusterClient) (*RedisInfo, error) {
	rawInfo, err := client.Info(ctx).Result()
	if err != nil {
		return nil, mapRedisInfoError(err)
	}
	clusterNodes, err := client.ClusterNodes(ctx).Result()
	if err != nil {
		return nil, mapRedisInfoError(err)
	}
	info := parseRedisInfo(rawInfo)

	queueLocations, err := m.fetchQueueLocations()
	if err != nil {
		return nil, mapRedisInfoError(err)
	}

	return &RedisInfo{
		Address:        strings.Join(client.Options().Addrs, ","),
		Info:           info,
		RawInfo:        rawInfo,
		IsCluster:      true,
		ClusterNodes:   clusterNodes,
		QueueLocations: queueLocations,
	}, nil
}

func (m *Manager) fetchQueueLocations() ([]*QueueLocation, error) {
	queues, err := m.inspector.Queues()
	if err != nil {
		return nil, mapManagerError(err)
	}

	locations := make([]*QueueLocation, len(queues))
	for i, queue := range queues {
		keySlot, err := m.inspector.ClusterKeySlot(queue)
		if err != nil {
			return nil, mapManagerError(err)
		}

		nodes, err := m.inspector.ClusterNodes(queue)
		if err != nil {
			return nil, mapManagerError(err)
		}

		nodeAddrs := make([]string, len(nodes))
		for j, node := range nodes {
			nodeAddrs[j] = node.Addr
		}

		locations[i] = &QueueLocation{
			Queue:   queue,
			KeySlot: keySlot,
			Nodes:   nodeAddrs,
		}
	}

	return locations, nil
}

func parseRedisInfo(infoStr string) map[string]string {
	info := make(map[string]string)
	for line := range strings.SplitSeq(infoStr, "\r\n") {
		if key, value, ok := strings.Cut(line, ":"); ok {
			info[key] = value
		}
	}
	return info
}

func (m *Manager) taskInfo(queue, jobID string) (*asynq.TaskInfo, error) {
	taskInfo, err := m.inspector.GetTaskInfo(queue, jobID)
	if err != nil {
		return nil, mapManagerError(err)
	}
	return taskInfo, nil
}

func mapManagerError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, asynq.ErrTaskNotFound):
		return ErrJobNotFound
	case errors.Is(err, asynq.ErrQueueNotFound):
		return ErrQueueNotFound
	case errors.Is(err, asynq.ErrQueueNotEmpty):
		return ErrQueueNotEmpty
	case isAsynqNotFound(err, "cannot find task"):
		return ErrJobNotFound
	case isAsynqNotFound(err, "queue ") && strings.Contains(err.Error(), "does not exist"):
		return ErrQueueNotFound
	default:
		return err
	}
}

func isAsynqNotFound(err error, text string) bool {
	var debug interface{ DebugString() string }
	return errors.As(err, &debug) &&
		strings.Contains(debug.DebugString(), "NOT_FOUND") &&
		strings.Contains(debug.DebugString(), text)
}

func mapRedisInfoError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, ErrQueueNotFound), errors.Is(err, ErrJobNotFound),
		errors.Is(err, ErrQueueNotEmpty), errors.Is(err, ErrRedisClientNotSupported):
		return err
	default:
		return fmt.Errorf("%w: %w", ErrRedisUnavailable, err)
	}
}
