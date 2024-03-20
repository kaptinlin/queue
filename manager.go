package queue

import (
	"context"
	"errors"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidJobState              = errors.New("invalid job state provided")
	ErrOperationNotSupported        = errors.New("operation not supported for the given job state")
	ErrArchivingActiveJobsDirectly  = errors.New("archiving active jobs directly is not supported")
	ErrGroupRequiredForAggregation  = errors.New("group identifier required for aggregating jobs operation")
	ErrUnsupportedJobStateForAction = errors.New("unsupported job state for the requested action")
	ErrRedisClientTypeNotSupported  = errors.New("redis client type not supported")
)

// ManagerInterface defines operations for managing and retrieving information about workers and their jobs.
type ManagerInterface interface {
	ListWorkers() ([]*WorkerInfo, error)
	ListQueues() ([]*QueueInfo, error)
	GetQueueInfo(queueName string) (*QueueInfo, []*QueueDailyStats, error)
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
	DeleteAggregatingTasks(queue, group string) (int, error)
	GetRedisInfo() (*RedisInfo, error)
}

// Manager provides an implementation for the ManagerInterface.
type Manager struct {
	Client    redis.UniversalClient
	Inspector *asynq.Inspector
}

// NewManager creates a new instance of Manager.
func NewManager(client redis.UniversalClient, inspector *asynq.Inspector) *Manager {
	return &Manager{
		Client:    client,
		Inspector: inspector,
	}
}

// ListWorkers retrieves information about all Asynq servers (workers) and the jobs they are currently processing.
func (s *Manager) ListWorkers() ([]*WorkerInfo, error) {
	servers, err := s.Inspector.Servers()
	if err != nil {
		return nil, err
	}

	var workers []*WorkerInfo
	for _, server := range servers {
		workers = append(workers, toWorkerInfo(server))
	}
	return workers, nil
}

// GetWorkerInfo retrieves detailed information about a single worker using its ID.
func (s *Manager) GetWorkerInfo(workerID string) (*WorkerInfo, error) {
	servers, err := s.Inspector.Servers()
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		if server.ID == workerID {
			return toWorkerInfo(server), nil
		}
	}

	return nil, nil
}

// ListQueues lists all queue names.
func (s *Manager) ListQueues() ([]*QueueInfo, error) {
	queues, err := s.Inspector.Queues()
	if err != nil {
		return nil, err
	}

	var snapshots []*QueueInfo
	for _, queue := range queues {
		qinfo, err := s.Inspector.GetQueueInfo(queue)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, toQueueInfo(qinfo))
	}
	return snapshots, nil
}

// GetQueueInfo gets detailed information about a queue.
func (s *Manager) GetQueueInfo(queueName string) (*QueueInfo, []*QueueDailyStats, error) {
	qinfo, err := s.Inspector.GetQueueInfo(queueName)
	if err != nil {
		return nil, nil, err
	}

	snapshot := toQueueInfo(qinfo)

	// Assuming a fixed number of days for history stats.
	const days = 7
	dstats, err := s.Inspector.History(queueName, days)
	if err != nil {
		return nil, nil, err
	}

	var QueuedailyStats []*QueueDailyStats
	for _, d := range dstats {
		QueuedailyStats = append(QueuedailyStats, toQueueDailyStats(d))
	}

	return snapshot, QueuedailyStats, nil
}

// ListQueueStats lists statistics for a queue over the past n days.
func (s *Manager) ListQueueStats(queueName string, days int) ([]*QueueDailyStats, error) {
	dstats, err := s.Inspector.History(queueName, days)
	if err != nil {
		return nil, err
	}

	var QueuedailyStats []*QueueDailyStats
	for _, d := range dstats {
		QueuedailyStats = append(QueuedailyStats, toQueueDailyStats(d))
	}

	return QueuedailyStats, nil
}

// DeleteQueue deletes a queue by its name.
func (s *Manager) DeleteQueue(queueName string, force bool) error {
	return s.Inspector.DeleteQueue(queueName, force)
}

// PauseQueue pauses a queue by its name.
func (s *Manager) PauseQueue(queueName string) error {
	return s.Inspector.PauseQueue(queueName)
}

// ResumeQueue resumes a paused queue by its name.
func (s *Manager) ResumeQueue(queueName string) error {
	return s.Inspector.UnpauseQueue(queueName)
}

// ListJobsByState lists jobs in a specified queue filtered by their state.
func (s *Manager) ListJobsByState(queue string, state JobState, size, page int) ([]*JobInfo, error) {
	if !IsValidJobState(state) {
		return nil, ErrInvalidJobState
	}

	// Handle active jobs separately to attach WorkerInfo.
	if state == StateActive {
		return s.ListActiveJobs(queue, size, page)
	}

	// For all other states, list jobs without WorkerInfo.
	var tasks []*asynq.TaskInfo
	var err error
	switch state {
	case StatePending:
		tasks, err = s.Inspector.ListPendingTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateRetry:
		tasks, err = s.Inspector.ListRetryTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateArchived:
		tasks, err = s.Inspector.ListArchivedTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateCompleted:
		tasks, err = s.Inspector.ListCompletedTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateScheduled:
		tasks, err = s.Inspector.ListScheduledTasks(queue, asynq.PageSize(size), asynq.Page(page))
	case StateAggregating:
		tasks, err = s.Inspector.ListAggregatingTasks(queue, "", asynq.PageSize(size), asynq.Page(page))
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
	tasks, err := s.Inspector.ListActiveTasks(queue, asynq.PageSize(size), asynq.Page(page))
	if err != nil {
		return nil, err
	}

	// Retrieve servers to map tasks to their corresponding active workers.
	servers, err := s.Inspector.Servers()
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

	// Convert tasks to JobInfo and attach WorkerInfo for active jobs.
	jobInfos := make([]*JobInfo, len(tasks))
	for i, task := range tasks {
		wi := workerInfoMap[task.ID]
		jobInfos[i] = toJobInfo(task, wi)
	}
	return jobInfos, nil
}

// GetJobInfo retrieves information for a single job using its ID and queue name.
func (s *Manager) GetJobInfo(queue, jobID string) (*JobInfo, error) {
	taskInfo, err := s.Inspector.GetTaskInfo(queue, jobID)
	if err != nil {
		return nil, err
	}

	return toJobInfo(taskInfo, nil), nil
}

// RunJob triggers immediate execution of a job with the specified ID.
func (s *Manager) RunJob(queue, jobID string) error {
	return s.Inspector.RunTask(queue, jobID)
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
		count, err = s.Inspector.RunAllScheduledTasks(queue)
	case StateRetry:
		count, err = s.Inspector.RunAllRetryTasks(queue)
	case StateArchived:
		count, err = s.Inspector.RunAllArchivedTasks(queue)
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

// BatchRunJobs triggers immediate execution of multiple jobs identified by their IDs.
func (s *Manager) BatchRunJobs(queue string, jobIDs []string) ([]string, []string, error) {
	var pendingIDs, errorIDs []string
	for _, jobID := range jobIDs {
		if err := s.Inspector.RunTask(queue, jobID); err != nil {
			errorIDs = append(errorIDs, jobID)
		} else {
			pendingIDs = append(pendingIDs, jobID)
		}
	}
	return pendingIDs, errorIDs, nil
}

// ArchiveJob moves a job with the specified ID to the archive.
func (s *Manager) ArchiveJob(queue, jobID string) error {
	return s.Inspector.ArchiveTask(queue, jobID)
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
		count, err = s.Inspector.ArchiveAllPendingTasks(queue)
	case StateArchived:
		// It does not make sense to archive already archived jobs.
		return 0, ErrOperationNotSupported
	case StateCompleted:
		// Directly archiving completed jobs may not be supported depending on the system design.
		return 0, ErrOperationNotSupported
	case StateScheduled:
		count, err = s.Inspector.ArchiveAllScheduledTasks(queue)
	case StateRetry:
		count, err = s.Inspector.ArchiveAllRetryTasks(queue)
	case StateActive:
		// Archiving active jobs directly is typically not supported as they are currently being processed.
		return 0, ErrArchivingActiveJobsDirectly
	case StateAggregating:
		// Archiving aggregating jobs requires specifying a group identifier.
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
	var archivedIDs, errorIDs []string
	for _, jobID := range jobIDs {
		if err := s.Inspector.ArchiveTask(queue, jobID); err != nil {
			errorIDs = append(errorIDs, jobID)
		} else {
			archivedIDs = append(archivedIDs, jobID)
		}
	}
	return archivedIDs, errorIDs, nil
}

// CancelJob cancels a job with the specified ID.
func (s *Manager) CancelJob(jobID string) error {
	return s.Inspector.CancelProcessing(jobID)
}

// CancelActiveJobs cancels all active jobs in the specified queue.
func (s *Manager) CancelActiveJobs(queue string, size, page int) (int, error) {
	var totalCount int

	for {
		tasks, err := s.Inspector.ListActiveTasks(queue, asynq.PageSize(size), asynq.Page(page))
		if err != nil {
			return totalCount, err
		}

		if len(tasks) == 0 {
			break
		}

		for _, task := range tasks {
			if err := s.Inspector.CancelProcessing(task.ID); err != nil {
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
	var canceledIDs, errorIDs []string
	for _, id := range jobIDs {
		if err := s.Inspector.CancelProcessing(id); err != nil {
			errorIDs = append(errorIDs, id)
		} else {
			canceledIDs = append(canceledIDs, id)
		}
	}
	return canceledIDs, errorIDs, nil
}

// DeleteJob deletes a job with the specified ID from its queue.
func (s *Manager) DeleteJob(queue, jobID string) error {
	return s.Inspector.DeleteTask(queue, jobID)
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
		count, err = s.Inspector.DeleteAllPendingTasks(queue)
	case StateArchived:
		count, err = s.Inspector.DeleteAllArchivedTasks(queue)
	case StateCompleted:
		count, err = s.Inspector.DeleteAllCompletedTasks(queue)
	case StateScheduled:
		count, err = s.Inspector.DeleteAllScheduledTasks(queue)
	case StateRetry:
		count, err = s.Inspector.DeleteAllRetryTasks(queue)
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
	var deletedIDs, failedIDs []string
	for _, jobID := range jobIDs {
		if err := s.Inspector.DeleteTask(queue, jobID); err != nil {
			failedIDs = append(failedIDs, jobID)
		} else {
			deletedIDs = append(deletedIDs, jobID)
		}
	}
	return deletedIDs, failedIDs, nil
}

// RunAggregatingJobs triggers all aggregating jobs to run immediately in a specified queue and group.
func (s *Manager) RunAggregatingJobs(queue, group string) (int, error) {
	return s.Inspector.RunAllAggregatingTasks(queue, group)
}

// ArchiveAggregatingJobs archives all aggregating jobs in a specified queue and group.
func (s *Manager) ArchiveAggregatingJobs(queue, group string) (int, error) {
	return s.Inspector.ArchiveAllAggregatingTasks(queue, group)
}

// DeleteAggregatingTasks deletes all aggregating tasks in a specified queue and group.
func (s *Manager) DeleteAggregatingTasks(queue, group string) (int, error) {
	return s.Inspector.DeleteAllAggregatingTasks(queue, group)
}

// GetRedisInfo retrieves information from the Redis server or cluster.
func (s *Manager) GetRedisInfo() (*RedisInfo, error) {
	switch client := s.Client.(type) {
	case *redis.ClusterClient:
		return s.getRedisClusterInfo(client)
	case *redis.Client:
		return s.getRedisStandardInfo(client)
	default:
		return nil, ErrRedisClientTypeNotSupported
	}
}

func (s *Manager) getRedisStandardInfo(client *redis.Client) (*RedisInfo, error) {
	rawInfo, err := client.Info(context.Background(), "all").Result()
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

func (s *Manager) getRedisClusterInfo(client *redis.ClusterClient) (*RedisInfo, error) {
	ctx := context.Background()
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
	queues, err := s.Inspector.Queues()
	if err != nil {
		return nil, err
	}

	var locations []*QueueLocation
	for _, queue := range queues {
		keySlot, err := s.Inspector.ClusterKeySlot(queue)
		if err != nil {
			return nil, err
		}

		nodes, err := s.Inspector.ClusterNodes(queue)
		if err != nil {
			return nil, err
		}

		var nodeAddrs []string
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
	lines := strings.Split(infoStr, "\r\n")
	for _, line := range lines {
		if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
			info[parts[0]] = parts[1]
		}
	}
	return info
}
