package queue

import (
	"time"

	"github.com/hibiken/asynq"
)

// WorkerInfo wraps detailed information about an Asynq server, which we treat as a "worker."
type WorkerInfo struct {
	ID             string           `json:"id"`
	Host           string           `json:"host"`
	PID            int              `json:"pid"`
	Concurrency    int              `json:"concurrency"`
	Queues         map[string]int   `json:"queues"`
	StrictPriority bool             `json:"strict_priority"`
	Started        time.Time        `json:"started"`
	Status         string           `json:"status"`
	ActiveJobs     []*ActiveJobInfo `json:"active_jobs"`
}

// Convert Asynq ServerInfo to WorkerInfo.
func toWorkerInfo(info *asynq.ServerInfo) *WorkerInfo {
	if info == nil {
		return nil
	}
	return &WorkerInfo{
		ID:             info.ID,
		Host:           info.Host,
		PID:            info.PID,
		Concurrency:    info.Concurrency,
		Queues:         info.Queues,
		StrictPriority: info.StrictPriority,
		Started:        info.Started,
		Status:         info.Status,
		ActiveJobs:     toActiveJobInfoList(info.ActiveWorkers),
	}
}

// ActiveJobInfo wraps detailed information about a job currently being processed by a worker.
type ActiveJobInfo struct {
	JobID      string    `json:"job_id"`
	JobType    string    `json:"job_type"`
	JobPayload string    `json:"job_payload"`
	Queue      string    `json:"queue"`
	StartedAt  time.Time `json:"started_at"`
	DeadlineAt time.Time `json:"deadline_at,omitempty"`
}

// Convert Asynq WorkerInfo to ActiveJobInfo.
func toActiveJobInfo(info *asynq.WorkerInfo) *ActiveJobInfo {
	if info == nil {
		return nil
	}
	return &ActiveJobInfo{
		JobID:      info.TaskID,
		JobType:    info.TaskType,
		JobPayload: string(info.TaskPayload),
		Queue:      info.Queue,
		StartedAt:  info.Started,
		DeadlineAt: info.Deadline,
	}
}

// Convert a slice of Asynq WorkerInfo to a slice of ActiveJobInfo.
func toActiveJobInfoList(infos []*asynq.WorkerInfo) []*ActiveJobInfo {
	var activeJobs []*ActiveJobInfo
	for _, info := range infos {
		activeJobs = append(activeJobs, toActiveJobInfo(info))
	}
	return activeJobs
}

// QueueInfo includes detailed queue information.
type QueueInfo struct {
	Queue       string        `json:"queue"`
	MemoryUsage int64         `json:"memory_usage"`
	Size        int           `json:"size"`
	Groups      int           `json:"groups"`
	Latency     time.Duration `json:"latency"`
	Active      int           `json:"active"`
	Pending     int           `json:"pending"`
	Scheduled   int           `json:"scheduled"`
	Retry       int           `json:"retry"`
	Archived    int           `json:"archived"`
	Completed   int           `json:"completed"`
	Processed   int           `json:"processed"`
	Succeeded   int           `json:"succeeded"`
	Failed      int           `json:"failed"`
	Paused      bool          `json:"paused"`
	Timestamp   time.Time     `json:"timestamp"`
}

// Convert Asynq QueueInfo to QueueInfo.
func toQueueInfo(info *asynq.QueueInfo) *QueueInfo {
	if info == nil {
		return nil
	}
	return &QueueInfo{
		Queue:       info.Queue,
		MemoryUsage: info.MemoryUsage,
		Size:        info.Size,
		Groups:      info.Groups,
		Latency:     info.Latency,
		Active:      info.Active,
		Pending:     info.Pending,
		Scheduled:   info.Scheduled,
		Retry:       info.Retry,
		Archived:    info.Archived,
		Completed:   info.Completed,
		Processed:   info.Processed,
		Succeeded:   info.Processed - info.Failed,
		Failed:      info.Failed,
		Paused:      info.Paused,
		Timestamp:   info.Timestamp,
	}
}

// QueueDailyStats includes detailed daily statistics for a queue.
type QueueDailyStats struct {
	Queue     string    `json:"queue"`
	Processed int       `json:"processed"`
	Succeeded int       `json:"succeeded"`
	Failed    int       `json:"failed"`
	Date      time.Time `json:"date"`
}

// Convert Asynq QueueDailyStats to our QueueDailyStats structure.
func toQueueDailyStats(s *asynq.DailyStats) *QueueDailyStats {
	if s == nil {
		return nil
	}
	return &QueueDailyStats{
		Queue:     s.Queue,
		Processed: s.Processed,
		Succeeded: s.Processed - s.Failed,
		Failed:    s.Failed,
		Date:      s.Date,
	}
}

// JobInfo includes detailed information for a job, mirroring relevant parts of asynq's TaskInfo and WorkerInfo for active jobs.
type JobInfo struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"`
	State         JobState   `json:"state"`
	Payload       string     `json:"payload"`
	Queue         string     `json:"queue"`
	MaxRetry      int        `json:"max_retry"`
	Retried       int        `json:"retried"`
	LastError     string     `json:"last_error"`
	NextProcessAt *time.Time `json:"next_process_at,omitempty"`
	LastFailedAt  *time.Time `json:"last_failed_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	// Additional fields for active tasks.
	StartedAt  *time.Time `json:"started_at,omitempty"`
	DeadlineAt *time.Time `json:"deadline_at,omitempty"`
	IsOrphaned bool       `json:"is_orphaned,omitempty"`
	// Addtional fields for aggregating tasks.
	Group *string `json:"group,omitempty"`
	// Result field for completed tasks.
	Result *string `json:"result,omitempty"`
}

// toJobInfo converts asynq.TaskInfo and optional asynq.WorkerInfo (for active tasks) to a JobInfo.
// WorkerInfo is nil for non-active tasks.
func toJobInfo(ti *asynq.TaskInfo, wi *asynq.WorkerInfo) *JobInfo {
	if ti == nil {
		return nil
	}
	jobInfo := &JobInfo{
		ID:         ti.ID,
		Type:       ti.Type,
		State:      toJobState(ti.State),
		Payload:    string(ti.Payload),
		Queue:      ti.Queue,
		MaxRetry:   ti.MaxRetry,
		Retried:    ti.Retried,
		LastError:  ti.LastErr,
		IsOrphaned: ti.IsOrphaned,
	}

	if !ti.NextProcessAt.IsZero() {
		jobInfo.NextProcessAt = &ti.NextProcessAt
	}

	if !ti.LastFailedAt.IsZero() {
		jobInfo.LastFailedAt = &ti.LastFailedAt
	}

	if !ti.CompletedAt.IsZero() {
		jobInfo.CompletedAt = &ti.CompletedAt
	}

	if ti.Group != "" {
		jobInfo.Group = &ti.Group
	}

	if wi != nil { // Handling active tasks specific fields.
		if !wi.Started.IsZero() {
			jobInfo.StartedAt = &wi.Started
		}
		if !wi.Deadline.IsZero() {
			jobInfo.DeadlineAt = &wi.Deadline
		}
	}

	if ti.Result != nil {
		result := string(ti.Result)
		jobInfo.Result = &result
	}

	return jobInfo
}

// RedisInfo contains detailed information about the Redis instance or cluster.
type RedisInfo struct {
	Address        string            `json:"address"`
	Info           map[string]string `json:"info"`
	RawInfo        string            `json:"raw_info"`
	IsCluster      bool              `json:"is_cluster"`
	ClusterNodes   string            `json:"cluster_nodes,omitempty"`
	QueueLocations []*QueueLocation  `json:"queue_locations,omitempty"`
}

// QueueLocation contains information about the location of queues in a Redis cluster.
type QueueLocation struct {
	Queue   string   `json:"queue"`
	KeySlot int64    `json:"key_slot"`
	Nodes   []string `json:"nodes"`
}
