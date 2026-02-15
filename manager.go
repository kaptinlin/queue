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
	GetWorkerInfo(workerID string) (*WorkerInfo, error)
	ListQueues() ([]*QueueInfo, error)
	GetQueueInfo(queueName string) (*QueueInfo, error)
	ListQueueStats(queueName string, days int) ([]*QueueDailyStats, error)
	DeleteQueue(queueName string, force bool) error
	PauseQueue(queueName string) error
	ResumeQueue(queueName string) error
	ListJobsByState(queue string, state JobState, size, page int) ([]*JobInfo, error)
	ListActiveJobs(queue string, size, page int) ([]*JobInfo, error)
	GetJobInfo(queue, jobID string) (*JobInfo, error)
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
	GetRedisInfo(ctx context.Context) (*RedisInfo, error)
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
func (s *Manager) ListWorkers() ([]*WorkerInfo, error) {
	servers, err := s.inspector.Servers()
	if err != nil {
		return nil, err
	}

	workers := make([]*WorkerInfo, len(servers))
	for i, server := range servers {
		workers[i] = toWorkerInfo(server)
	}
	return workers, nil
}

// GetWorkerInfo retrieves detailed information about a single worker using its ID.
func (s *Manager) GetWorkerInfo(workerID string) (*WorkerInfo, error) {
	servers, err := s.inspector.Servers()
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
func (s *Manager) ListQueues() ([]*QueueInfo, error) {
	queues, err := s.inspector.Queues()
	if err != nil {
		return nil, err
	}

	snapshots := make([]*QueueInfo, len(queues))
	for i, queue := range queues {
		qinfo, err := s.inspector.GetQueueInfo(queue)
		if err != nil {
			return nil, err
		}
		snapshots[i] = toQueueInfo(qinfo)
	}
	return snapshots, nil
}

// GetQueueInfo gets detailed information about a queue.
func (s *Manager) GetQueueInfo(queueName string) (*QueueInfo, error) {
	qinfo, err := s.inspector.GetQueueInfo(queueName)
	if err != nil {
		return nil, s.handleQueueError(err)
	}

	snapshot := toQueueInfo(qinfo)

	return snapshot, nil
}

// ListQueueStats lists statistics for a queue over the past n days.
func (s *Manager) ListQueueStats(queueName string, days int) ([]*QueueDailyStats, error) {
	dstats, err := s.inspector.History(queueName, days)
	if err != nil {
		return nil, s.handleQueueError(err)
	}

	dailyStats := make([]*QueueDailyStats, len(dstats))
	for i, d := range dstats {
		dailyStats[i] = toQueueDailyStats(d)
	}

	return dailyStats, nil
}

// DeleteQueue deletes a queue by its name.
func (s *Manager) DeleteQueue(queueName string, force bool) error {
	err := s.inspector.DeleteQueue(queueName, force)
	if err != nil {
		if errors.Is(err, asynq.ErrQueueNotEmpty) {
			return ErrQueueNotEmpty
		}
		return s.handleQueueError(err)
	}
	return nil
}

// PauseQueue pauses a queue by its name.
func (s *Manager) PauseQueue(queueName string) error {
	err := s.inspector.PauseQueue(queueName)
	if err != nil {
		return s.handleQueueError(err)
	}
	return nil
}

// ResumeQueue resumes a paused queue by its name.
func (s *Manager) ResumeQueue(queueName string) error {
	err := s.inspector.UnpauseQueue(queueName)
	if err != nil {
		return s.handleQueueError(err)
	}
	return nil
}

// ListJobsByState lists jobs in a specified queue filtered by their state.
func (s *Manager) ListJobsByState(queue string, state JobState, size, page int) ([]*JobInfo, error) {
	if !IsValidJobState(state) {
		return nil, ErrInvalidJobState
	}

	if state == StateActive {
		return s.ListActiveJobs(queue, size, page)
	}

	var tasks []*asynq.TaskInfo
	var err error
	switch state { //nolint:exhaustive
	case StatePending:
		tasks, err = s.inspector.ListPendingTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateRetry:
		tasks, err = s.inspector.ListRetryTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateArchived:
		tasks, err = s.inspector.ListArchivedTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateCompleted:
		tasks, err = s.inspector.ListCompletedTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateScheduled:
		tasks, err = s.inspector.ListScheduledTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateAggregating:
		tasks, err = s.inspector.ListAggregatingTasks(queue, "", asynq.PageSize(size), asynq.Page(page))
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
func (s *Manager) ListActiveJobs(queue string, size, page int) ([]*JobInfo, error) {
	tasks, err := s.inspector.ListActiveTasks(queue, asynq.PageSize(size), asynq.Page(page))
	if err != nil {
		return nil, err
	}

	servers, err := s.inspector.Servers()
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

// GetJobInfo retrieves information for a single job using its ID and queue name.
func (s *Manager) GetJobInfo(queue, jobID string) (*JobInfo, error) {
	taskInfo, err := s.inspector.GetTaskInfo(queue, jobID)
	if err != nil {
		if errors.Is(err, asynq.ErrTaskNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	return toJobInfo(taskInfo, nil), nil
}

// RunJob triggers immediate execution of a job with the specified ID.
func (s *Manager) RunJob(queue, jobID string) error {
	err := s.inspector.RunTask(queue, jobID)
	if err != nil {
		if errors.Is(err, asynq.ErrTaskNotFound) {
			return ErrJobNotFound
		}
		return err
	}
	return nil
}

// RunJobsByState triggers all jobs in a specified queue and state to run immediately.
func (s *Manager) RunJobsByState(queue string, state JobState) (int, error) {
	if !IsValidJobState(state) {
		return 0, ErrInvalidJobState
	}

	var count int
	var err error
	switch state {
	case StateScheduled:
		count, err = s.inspector.RunAllScheduledTasks(queue)
	case StateRetry:
		count, err = s.inspector.RunAllRetryTasks(queue)
	case StateArchived:
		count, err = s.inspector.RunAllArchivedTasks(queue)
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
		} else {
			succeeded = append(succeeded, item)
		}
	}

	if len(errs) > 0 {
		return succeeded, failed, errors.Join(errs...)
	}
	return succeeded, failed, nil
}

// BatchRunJobs triggers immediate execution of multiple jobs identified by their IDs.
func (s *Manager) BatchRunJobs(queue string, jobIDs []string) ([]string, []string, error) {
	return batchOperation(jobIDs, func(jobID string) error {
		return s.inspector.RunTask(queue, jobID)
	})
}

// ArchiveJob moves a job with the specified ID to the archive.
func (s *Manager) ArchiveJob(queue, jobID string) error {
	err := s.inspector.ArchiveTask(queue, jobID)
	if err != nil {
		if errors.Is(err, asynq.ErrTaskNotFound) {
			return ErrJobNotFound
		}
		return err
	}
	return nil
}

// ArchiveJobsByState archives all jobs in a specified queue based on their state.
func (s *Manager) ArchiveJobsByState(queue string, state JobState) (int, error) {
	if !IsValidJobState(state) {
		return 0, ErrInvalidJobState
	}

	var count int
	var err error
	switch state {
	case StatePending:
		count, err = s.inspector.ArchiveAllPendingTasks(queue)
	case StateScheduled:
		count, err = s.inspector.ArchiveAllScheduledTasks(queue)
	case StateRetry:
		count, err = s.inspector.ArchiveAllRetryTasks(queue)
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
func (s *Manager) BatchArchiveJobs(queue string, jobIDs []string) ([]string, []string, error) {
	return batchOperation(jobIDs, func(jobID string) error {
		return s.inspector.ArchiveTask(queue, jobID)
	})
}

// CancelJob cancels a job with the specified ID.
func (s *Manager) CancelJob(jobID string) error {
	err := s.inspector.CancelProcessing(jobID)
	if err != nil {
		if errors.Is(err, asynq.ErrTaskNotFound) {
			return ErrJobNotFound
		}
		return err
	}
	return nil
}

// CancelActiveJobs cancels all active jobs in the specified queue.
func (s *Manager) CancelActiveJobs(queue string, size, page int) (int, error) {
	var totalCount int

	for {
		tasks, err := s.inspector.ListActiveTasks(queue, asynq.PageSize(size), asynq.Page(page))
		if err != nil {
			return totalCount, err
		}

		if len(tasks) == 0 {
			break
		}

		for _, task := range tasks {
			if err := s.inspector.CancelProcessing(task.ID); err != nil {
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
func (s *Manager) BatchCancelJobs(jobIDs []string) ([]string, []string, error) {
	return batchOperation(jobIDs, func(jobID string) error {
		return s.inspector.CancelProcessing(jobID)
	})
}

// DeleteJob deletes a job with the specified ID from its queue.
func (s *Manager) DeleteJob(queue, jobID string) error {
	err := s.inspector.DeleteTask(queue, jobID)
	if err != nil {
		if errors.Is(err, asynq.ErrTaskNotFound) {
			return ErrJobNotFound
		}
		return err
	}
	return nil
}

// DeleteJobsByState deletes all jobs in a specified queue based on their state.
func (s *Manager) DeleteJobsByState(queue string, state JobState) (int, error) {
	if !IsValidJobState(state) {
		return 0, ErrInvalidJobState
	}

	var count int
	var err error
	switch state {
	case StatePending:
		count, err = s.inspector.DeleteAllPendingTasks(queue)
	case StateArchived:
		count, err = s.inspector.DeleteAllArchivedTasks(queue)
	case StateCompleted:
		count, err = s.inspector.DeleteAllCompletedTasks(queue)
	case StateScheduled:
		count, err = s.inspector.DeleteAllScheduledTasks(queue)
	case StateRetry:
		count, err = s.inspector.DeleteAllRetryTasks(queue)
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
func (s *Manager) BatchDeleteJobs(queue string, jobIDs []string) ([]string, []string, error) {
	return batchOperation(jobIDs, func(jobID string) error {
		return s.inspector.DeleteTask(queue, jobID)
	})
}

// RunAggregatingJobs triggers all aggregating jobs to run immediately in a specified queue and group.
func (s *Manager) RunAggregatingJobs(queue, group string) (int, error) {
	return s.inspector.RunAllAggregatingTasks(queue, group)
}

// ArchiveAggregatingJobs archives all aggregating jobs in a specified queue and group.
func (s *Manager) ArchiveAggregatingJobs(queue, group string) (int, error) {
	return s.inspector.ArchiveAllAggregatingTasks(queue, group)
}

// DeleteAggregatingJobs deletes all aggregating tasks in a specified queue and group.
func (s *Manager) DeleteAggregatingJobs(queue, group string) (int, error) {
	return s.inspector.DeleteAllAggregatingTasks(queue, group)
}

// GetRedisInfo retrieves information from the Redis server or cluster.
func (s *Manager) GetRedisInfo(ctx context.Context) (*RedisInfo, error) {
	switch client := s.client.(type) {
	case *redis.ClusterClient:
		return s.getRedisClusterInfo(ctx, client)
	case *redis.Client:
		return s.getRedisStandardInfo(ctx, client)
	default:
		return nil, ErrRedisClientNotSupported
	}
}

func (s *Manager) getRedisStandardInfo(ctx context.Context, client *redis.Client) (*RedisInfo, error) {
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

func (s *Manager) getRedisClusterInfo(ctx context.Context, client *redis.ClusterClient) (*RedisInfo, error) {
	rawInfo, err := client.Info(ctx).Result()
	if err != nil {
		return nil, err
	}
	clusterNodes, err := client.ClusterNodes(ctx).Result()
	if err != nil {
		return nil, err
	}
	info := parseRedisInfo(rawInfo)

	queueLocations, err := s.fetchQueueLocations()
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

func (s *Manager) fetchQueueLocations() ([]*QueueLocation, error) {
	queues, err := s.inspector.Queues()
	if err != nil {
		return nil, err
	}

	locations := make([]*QueueLocation, 0, len(queues))
	for _, queue := range queues {
		keySlot, err := s.inspector.ClusterKeySlot(queue)
		if err != nil {
			return nil, err
		}

		nodes, err := s.inspector.ClusterNodes(queue)
		if err != nil {
			return nil, err
		}

		nodeAddrs := make([]string, 0, len(nodes))
		for _, node := range nodes {
			nodeAddrs = append(nodeAddrs, node.Addr)
		}

		location := &QueueLocation{
			Queue:   queue,
			KeySlot: keySlot,
			Nodes:   nodeAddrs,
		}

		locations = append(locations, location)
	}

	return locations, nil
}

// parseRedisInfo parses the INFO command's output into a key-value map.
func parseRedisInfo(infoStr string) map[string]string {
	info := make(map[string]string)
	for line := range strings.SplitSeq(infoStr, "\r\n") {
		if key, value, ok := strings.Cut(line, ":"); ok {
			info[key] = value
		}
	}
	return info
}

func (s *Manager) handleQueueError(err error) error {
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
