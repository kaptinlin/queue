package queue

import (
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToWorkerInfo(t *testing.T) {
	now := time.Now()
	info := &asynq.ServerInfo{
		ID: "w1", Host: "h1", PID: 42,
		Concurrency: 8, Queues: map[string]int{"q": 1},
		StrictPriority: true, Started: now, Status: "active",
		ActiveWorkers: []*asynq.WorkerInfo{{
			TaskID: "t1", TaskType: "job", Queue: "q",
			TaskPayload: []byte(`{}`),
			Started:     now, Deadline: now.Add(time.Hour),
		}},
	}
	w := toWorkerInfo(info)
	require.NotNil(t, w)
	assert.Equal(t, "w1", w.ID)
	assert.Equal(t, "h1", w.Host)
	assert.Equal(t, 42, w.PID)
	assert.Equal(t, 8, w.Concurrency)
	assert.True(t, w.StrictPriority)
	assert.Equal(t, "active", w.Status)
	assert.Len(t, w.ActiveJobs, 1)
	assert.Equal(t, "t1", w.ActiveJobs[0].JobID)
}

func TestToWorkerInfo_Nil(t *testing.T) {
	assert.Nil(t, toWorkerInfo(nil))
}

func TestToActiveJobInfo(t *testing.T) {
	now := time.Now()
	wi := &asynq.WorkerInfo{
		TaskID: "t1", TaskType: "email",
		TaskPayload: []byte(`{"k":"v"}`),
		Queue:       "critical", Started: now,
		Deadline: now.Add(time.Hour),
	}
	a := toActiveJobInfo(wi)
	require.NotNil(t, a)
	assert.Equal(t, "t1", a.JobID)
	assert.Equal(t, "email", a.JobType)
	assert.Equal(t, `{"k":"v"}`, a.JobPayload)
	assert.Equal(t, "critical", a.Queue)
}

func TestToActiveJobInfo_Nil(t *testing.T) {
	assert.Nil(t, toActiveJobInfo(nil))
}

func TestToActiveJobInfoList(t *testing.T) {
	list := toActiveJobInfoList([]*asynq.WorkerInfo{
		{TaskID: "a"}, {TaskID: "b"},
	})
	assert.Len(t, list, 2)
	assert.Equal(t, "a", list[0].JobID)
}

func TestToActiveJobInfoList_Empty(t *testing.T) {
	assert.Empty(t, toActiveJobInfoList(nil))
}

func TestToQueueInfo(t *testing.T) {
	now := time.Now()
	qi := &asynq.QueueInfo{
		Queue: "q1", MemoryUsage: 1024, Size: 10,
		Groups: 2, Latency: time.Second,
		Active: 1, Pending: 2, Scheduled: 3,
		Retry: 4, Archived: 5, Completed: 6,
		Aggregating: 7, ProcessedTotal: 100,
		FailedTotal: 10, Paused: true, Timestamp: now,
	}
	info := toQueueInfo(qi)
	require.NotNil(t, info)
	assert.Equal(t, "q1", info.Queue)
	assert.Equal(t, int64(1024), info.MemoryUsage)
	assert.Equal(t, 10, info.Size)
	assert.Equal(t, 2, info.Groups)
	assert.Equal(t, 100, info.Processed)
	assert.Equal(t, 90, info.Succeeded)
	assert.Equal(t, 10, info.Failed)
	assert.True(t, info.Paused)
}

func TestToQueueInfo_Nil(t *testing.T) {
	assert.Nil(t, toQueueInfo(nil))
}

func TestToQueueDailyStats(t *testing.T) {
	now := time.Now()
	ds := &asynq.DailyStats{
		Queue: "q1", Processed: 50, Failed: 5, Date: now,
	}
	s := toQueueDailyStats(ds)
	require.NotNil(t, s)
	assert.Equal(t, "q1", s.Queue)
	assert.Equal(t, 50, s.Processed)
	assert.Equal(t, 45, s.Succeeded)
	assert.Equal(t, 5, s.Failed)
	assert.Equal(t, now, s.Date)
}

func TestToQueueDailyStats_Nil(t *testing.T) {
	assert.Nil(t, toQueueDailyStats(nil))
}

func TestToJobInfo_Nil(t *testing.T) {
	assert.Nil(t, toJobInfo(nil, nil))
}

func TestToJobInfo_Basic(t *testing.T) {
	ti := &asynq.TaskInfo{
		ID: "j1", Type: "email", Queue: "q",
		Payload: []byte(`{"k":"v"}`),
		State:   asynq.TaskStatePending, MaxRetry: 3,
	}
	info := toJobInfo(ti, nil)
	require.NotNil(t, info)
	assert.Equal(t, "j1", info.ID)
	assert.Equal(t, "email", info.Type)
	assert.Equal(t, "q", info.Queue)
	assert.Equal(t, 3, info.MaxRetry)
	assert.Nil(t, info.StartedAt)
	assert.Nil(t, info.DeadlineAt)
	assert.Nil(t, info.Group)
	assert.Nil(t, info.Result)
}

func TestToJobInfo_WithTimestamps(t *testing.T) {
	now := time.Now()
	ti := &asynq.TaskInfo{
		ID: "j2", Type: "t", Queue: "q",
		State:         asynq.TaskStatePending,
		NextProcessAt: now,
		LastFailedAt:  now.Add(-time.Hour),
		CompletedAt:   now.Add(-2 * time.Hour),
		LastErr:       "some error",
		Retried:       2,
	}
	info := toJobInfo(ti, nil)
	require.NotNil(t, info.NextProcessAt)
	require.NotNil(t, info.LastFailedAt)
	require.NotNil(t, info.CompletedAt)
	assert.Equal(t, "some error", info.LastError)
	assert.Equal(t, 2, info.Retried)
}

func TestToJobInfo_WithGroup(t *testing.T) {
	ti := &asynq.TaskInfo{
		ID: "j3", Type: "t", Queue: "q",
		State: asynq.TaskStateAggregating,
		Group: "my-group",
	}
	info := toJobInfo(ti, nil)
	require.NotNil(t, info.Group)
	assert.Equal(t, "my-group", *info.Group)
}

func TestToJobInfo_WithWorkerInfo(t *testing.T) {
	now := time.Now()
	ti := &asynq.TaskInfo{
		ID: "j4", Type: "t", Queue: "q",
		State: asynq.TaskStateActive,
	}
	wi := &asynq.WorkerInfo{
		Started:  now,
		Deadline: now.Add(time.Hour),
	}
	info := toJobInfo(ti, wi)
	require.NotNil(t, info.StartedAt)
	require.NotNil(t, info.DeadlineAt)
}

func TestToJobInfo_WithResult(t *testing.T) {
	ti := &asynq.TaskInfo{
		ID: "j5", Type: "t", Queue: "q",
		State:  asynq.TaskStateCompleted,
		Result: []byte(`{"status":"ok"}`),
	}
	info := toJobInfo(ti, nil)
	require.NotNil(t, info.Result)
	assert.Equal(t, `{"status":"ok"}`, *info.Result)
}
