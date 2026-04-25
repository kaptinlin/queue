package queue

import (
	"context"
	"errors"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// ManagerInterface defines operations for managing and retrieving information about workers and their jobs.
type ManagerInterface interface {
	ListWorkers() ([]*WorkerInfo, error)
	WorkerInfo(workerID string) (*WorkerInfo, error)
	ListQueues() ([]*QueueInfo, error)
	QueueInfo(queueName string) (*QueueInfo, error)
	ListQueueStats(queueName string, days int) ([]*QueueDailyStats, error)
	DeleteQueue(queueName string, force bool) error
	PauseQueue(queueName string) error
	ResumeQueue(queueName string) error
	ListJobsByState(queue string, state JobState, size, page int) ([]*JobInfo, error)
	ListActiveJobs(queue string, size, page int) ([]*JobInfo, error)
	JobInfo(queue, jobID string) (*JobInfo, error)
	RunJob(queue, jobID string) error
	RunJobsByState(queue string, state JobState) (int, error)
	BatchRunJobs(queue string, jobIDs []string) ([]string, []string, error)
	ArchiveJob(queue, jobID string) error
	ArchiveJobsByState(queue string, state JobState) (int, error)
	BatchArchiveJobs(queue string, jobIDs []string) ([]string, []string, error)
	CancelJob(jobID string) error
	CancelActiveJobs(queue string, size, page int) (int, error)
	BatchCancelJobs(jobIDs []string) ([]string, []string, error)
	DeleteJob(queue, jobID string) error
	DeleteJobsByState(queue string, state JobState) (int, error)
	BatchDeleteJobs(queue string, jobIDs []string) ([]string, []string, error)
	RunAggregatingJobs(queue, group string) (int, error)
	ArchiveAggregatingJobs(queue, group string) (int, error)
	DeleteAggregatingJobs(queue, group string) (int, error)
	RedisInfo(ctx context.Context) (*RedisInfo, error)
}

// Manager provides an implementation for the ManagerInterface.
type Manager struct {
	client    redis.UniversalClient
	inspector *asynq.Inspector
}

// NewManager creates a new instance of Manager.
func NewManager(client redis.UniversalClient, inspector *asynq.Inspector) *Manager {
	return &Manager{
		client:    client,
		inspector: inspector,
	}
}

// ListWorkers retrieves information about all Asynq servers (workers) and the jobs they are currently processing.
func (m *Manager) ListWorkers() ([]*WorkerInfo, error) {
	servers, err := m.inspector.Servers()
	if err != nil {
		return nil, err
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
		return nil, err
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
		return nil, err
	}

	snapshots := make([]*QueueInfo, len(queues))
	for i, queue := range queues {
		qinfo, err := m.inspector.GetQueueInfo(queue)
		if err != nil {
			return nil, err
		}
		snapshots[i] = toQueueInfo(qinfo)
	}
	return snapshots, nil
}

// QueueInfo retrieves detailed information about a queue.
func (m *Manager) QueueInfo(queueName string) (*QueueInfo, error) {
	qinfo, err := m.inspector.GetQueueInfo(queueName)
	if err != nil {
		return nil, handleQueueError(err)
	}
	return toQueueInfo(qinfo), nil
}

// ListQueueStats lists statistics for a queue over the past n days.
func (m *Manager) ListQueueStats(queueName string, days int) ([]*QueueDailyStats, error) {
	dstats, err := m.inspector.History(queueName, days)
	if err != nil {
		return nil, handleQueueError(err)
	}

	dailyStats := make([]*QueueDailyStats, len(dstats))
	for i, d := range dstats {
		dailyStats[i] = toQueueDailyStats(d)
	}

	return dailyStats, nil
}

// DeleteQueue deletes a queue by its name.
func (m *Manager) DeleteQueue(queueName string, force bool) error {
	err := m.inspector.DeleteQueue(queueName, force)
	if errors.Is(err, asynq.ErrQueueNotEmpty) {
		return ErrQueueNotEmpty
	}
	if err != nil {
		return handleQueueError(err)
	}
	return nil
}

// PauseQueue pauses a queue by its name.
func (m *Manager) PauseQueue(queueName string) error {
	if err := m.inspector.PauseQueue(queueName); err != nil {
		return handleQueueError(err)
	}
	return nil
}

// ResumeQueue resumes a paused queue by its name.
func (m *Manager) ResumeQueue(queueName string) error {
	if err := m.inspector.UnpauseQueue(queueName); err != nil {
		return handleQueueError(err)
	}
	return nil
}

// ListJobsByState lists jobs in a specified queue filtered by their state.
func (m *Manager) ListJobsByState(queue string, state JobState, size, page int) ([]*JobInfo, error) {
	if !IsValidJobState(state) {
		return nil, ErrInvalidJobState
	}

	if state == StateActive {
		return m.ListActiveJobs(queue, size, page)
	}

	var tasks []*asynq.TaskInfo
	var err error
	switch state {
	case StatePending:
		tasks, err = m.inspector.ListPendingTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateRetry:
		tasks, err = m.inspector.ListRetryTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateArchived:
		tasks, err = m.inspector.ListArchivedTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateCompleted:
		tasks, err = m.inspector.ListCompletedTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateScheduled:
		tasks, err = m.inspector.ListScheduledTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateAggregating:
		tasks, err = m.inspector.ListAggregatingTasks(queue, "", asynq.PageSize(size), asynq.Page(page))
	default:
		return nil, ErrUnsupportedJobStateForAction
	}

	if err != nil {
		return nil, err
	}

	jobInfos := make([]*JobInfo, len(tasks))
	for i, task := range tasks {
		jobInfos[i] = toJobInfo(task, nil)
	}
	return jobInfos, nil
}

// ListActiveJobs lists active (currently processing) jobs for a given queue.
func (m *Manager) ListActiveJobs(queue string, size, page int) ([]*JobInfo, error) {
	tasks, err := m.inspector.ListActiveTasks(queue, asynq.PageSize(size), asynq.Page(page))
	if err != nil {
		return nil, err
	}

	servers, err := m.inspector.Servers()
	if err != nil {
		return nil, err
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
	taskInfo, err := m.inspector.GetTaskInfo(queue, jobID)
	if errors.Is(err, asynq.ErrTaskNotFound) {
		return nil, ErrJobNotFound
	}
	if err != nil {
		return nil, err
	}
	return toJobInfo(taskInfo, nil), nil
}

// RunJob triggers immediate execution of a job with the specified ID.
func (m *Manager) RunJob(queue, jobID string) error {
	err := m.inspector.RunTask(queue, jobID)
	if errors.Is(err, asynq.ErrTaskNotFound) {
		return ErrJobNotFound
	}
	return err
}

// RunJobsByState triggers all jobs in a specified queue and state to run immediately.
func (m *Manager) RunJobsByState(queue string, state JobState) (int, error) {
	if !IsValidJobState(state) {
		return 0, ErrInvalidJobState
	}

	var count int
	var err error
	switch state {
	case StateScheduled:
		count, err = m.inspector.RunAllScheduledTasks(queue)
	case StateRetry:
		count, err = m.inspector.RunAllRetryTasks(queue)
	case StateArchived:
		count, err = m.inspector.RunAllArchivedTasks(queue)
	case StateAggregating:
		return 0, ErrGroupRequiredForAggregation
	case StateActive, StatePending, StateCompleted:
		return 0, ErrOperationNotSupported
	default:
		return 0, ErrUnsupportedJobStateForAction
	}

	if err != nil {
		return 0, err
	}

	return count, nil
}

// batchOperation performs a batch operation on items and returns succeeded, failed items, and aggregated errors.
func batchOperation[T any](items []T, operation func(T) error) (succeeded, failed []T, err error) {
	succeeded = make([]T, 0, len(items))
	failed = make([]T, 0, len(items))
	var errs []error

	for _, item := range items {
		if opErr := operation(item); opErr != nil {
			failed = append(failed, item)
			errs = append(errs, opErr)
			continue
		}
		succeeded = append(succeeded, item)
	}

	if len(errs) > 0 {
		return succeeded, failed, errors.Join(errs...)
	}
	return succeeded, failed, nil
}

// BatchRunJobs triggers immediate execution of multiple jobs identified by their IDs.
func (m *Manager) BatchRunJobs(queue string, jobIDs []string) ([]string, []string, error) {
	return batchOperation(jobIDs, func(jobID string) error {
		return m.inspector.RunTask(queue, jobID)
	})
}

// ArchiveJob moves a job with the specified ID to the archive.
func (m *Manager) ArchiveJob(queue, jobID string) error {
	err := m.inspector.ArchiveTask(queue, jobID)
	if errors.Is(err, asynq.ErrTaskNotFound) {
		return ErrJobNotFound
	}
	return err
}

// ArchiveJobsByState archives all jobs in a specified queue based on their state.
func (m *Manager) ArchiveJobsByState(queue string, state JobState) (int, error) {
	if !IsValidJobState(state) {
		return 0, ErrInvalidJobState
	}

	var count int
	var err error
	switch state {
	case StatePending:
		count, err = m.inspector.ArchiveAllPendingTasks(queue)
	case StateScheduled:
		count, err = m.inspector.ArchiveAllScheduledTasks(queue)
	case StateRetry:
		count, err = m.inspector.ArchiveAllRetryTasks(queue)
	case StateArchived, StateCompleted:
		return 0, ErrOperationNotSupported
	case StateActive:
		return 0, ErrArchivingActiveJobs
	case StateAggregating:
		return 0, ErrGroupRequiredForAggregation
	default:
		return 0, ErrUnsupportedJobStateForAction
	}

	if err != nil {
		return 0, err
	}
	return count, nil
}

// BatchArchiveJobs archives multiple jobs identified by their IDs.
func (m *Manager) BatchArchiveJobs(queue string, jobIDs []string) ([]string, []string, error) {
	return batchOperation(jobIDs, func(jobID string) error {
		return m.inspector.ArchiveTask(queue, jobID)
	})
}

// CancelJob cancels a job with the specified ID.
func (m *Manager) CancelJob(jobID string) error {
	err := m.inspector.CancelProcessing(jobID)
	if errors.Is(err, asynq.ErrTaskNotFound) {
		return ErrJobNotFound
	}
	return err
}

// CancelActiveJobs cancels all active jobs in the specified queue.
func (m *Manager) CancelActiveJobs(queue string, size, page int) (int, error) {
	var totalCount int

	for {
		tasks, err := m.inspector.ListActiveTasks(queue, asynq.PageSize(size), asynq.Page(page))
		if err != nil {
			return totalCount, err
		}

		if len(tasks) == 0 {
			break
		}

		for _, task := range tasks {
			if err := m.inspector.CancelProcessing(task.ID); err != nil {
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
func (m *Manager) BatchCancelJobs(jobIDs []string) ([]string, []string, error) {
	return batchOperation(jobIDs, func(jobID string) error {
		return m.inspector.CancelProcessing(jobID)
	})
}

// DeleteJob deletes a job with the specified ID from its queue.
func (m *Manager) DeleteJob(queue, jobID string) error {
	err := m.inspector.DeleteTask(queue, jobID)
	if errors.Is(err, asynq.ErrTaskNotFound) {
		return ErrJobNotFound
	}
	return err
}

// DeleteJobsByState deletes all jobs in a specified queue based on their state.
func (m *Manager) DeleteJobsByState(queue string, state JobState) (int, error) {
	if !IsValidJobState(state) {
		return 0, ErrInvalidJobState
	}

	var count int
	var err error
	switch state {
	case StatePending:
		count, err = m.inspector.DeleteAllPendingTasks(queue)
	case StateArchived:
		count, err = m.inspector.DeleteAllArchivedTasks(queue)
	case StateCompleted:
		count, err = m.inspector.DeleteAllCompletedTasks(queue)
	case StateScheduled:
		count, err = m.inspector.DeleteAllScheduledTasks(queue)
	case StateRetry:
		count, err = m.inspector.DeleteAllRetryTasks(queue)
	case StateActive, StateAggregating:
		return 0, ErrOperationNotSupported
	default:
		return 0, ErrUnsupportedJobStateForAction
	}

	if err != nil {
		return 0, err
	}
	return count, nil
}

// BatchDeleteJobs deletes multiple jobs identified by their IDs.
func (m *Manager) BatchDeleteJobs(queue string, jobIDs []string) ([]string, []string, error) {
	return batchOperation(jobIDs, func(jobID string) error {
		return m.inspector.DeleteTask(queue, jobID)
	})
}

// RunAggregatingJobs triggers all aggregating jobs to run immediately in a specified queue and group.
func (m *Manager) RunAggregatingJobs(queue, group string) (int, error) {
	return m.inspector.RunAllAggregatingTasks(queue, group)
}

// ArchiveAggregatingJobs archives all aggregating jobs in a specified queue and group.
func (m *Manager) ArchiveAggregatingJobs(queue, group string) (int, error) {
	return m.inspector.ArchiveAllAggregatingTasks(queue, group)
}

// DeleteAggregatingJobs deletes all aggregating tasks in a specified queue and group.
func (m *Manager) DeleteAggregatingJobs(queue, group string) (int, error) {
	return m.inspector.DeleteAllAggregatingTasks(queue, group)
}

// RedisInfo retrieves information from the Redis server or cluster.
func (m *Manager) RedisInfo(ctx context.Context) (*RedisInfo, error) {
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
		return nil, err
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
		return nil, err
	}
	clusterNodes, err := client.ClusterNodes(ctx).Result()
	if err != nil {
		return nil, err
	}
	info := parseRedisInfo(rawInfo)

	queueLocations, err := m.fetchQueueLocations()
	if err != nil {
		return nil, err
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
		return nil, err
	}

	locations := make([]*QueueLocation, len(queues))
	for i, queue := range queues {
		keySlot, err := m.inspector.ClusterKeySlot(queue)
		if err != nil {
			return nil, err
		}

		nodes, err := m.inspector.ClusterNodes(queue)
		if err != nil {
			return nil, err
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

func handleQueueError(err error) error {
	if errors.Is(err, asynq.ErrQueueNotFound) || isQueueNotFoundError(err) {
		return ErrQueueNotFound
	}
	return err
}

// isQueueNotFoundError uses string matching as a workaround because asynq
// does not expose a sentinel error for all queue-not-found scenarios.
func isQueueNotFoundError(err error) bool {
	return strings.Contains(err.Error(), "does not exist")
}
