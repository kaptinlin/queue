package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/go-json-experiment/json"
	"github.com/hibiken/asynq"
)

// Delivery describes one runtime delivery of a job to a handler.
type Delivery struct {
	id           string
	jobType      string
	queue        string
	retryCount   int
	maxRetry     int
	deadline     *time.Time
	rawPayload   []byte
	resultWriter *asynq.ResultWriter
}

func newDeliveryFromTask(ctx context.Context, task *asynq.Task, queue string) (*Delivery, error) {
	if task == nil {
		return nil, ErrInvalidAsynqTask
	}
	if ctx == nil {
		ctx = context.Background()
	}

	resultWriter := task.ResultWriter()
	var id string
	if resultWriter != nil {
		id = resultWriter.TaskID()
	}
	if taskID, ok := asynq.GetTaskID(ctx); ok {
		id = taskID
	}
	if queueName, ok := asynq.GetQueueName(ctx); ok {
		queue = queueName
	}

	var retryCount int
	if n, ok := asynq.GetRetryCount(ctx); ok {
		retryCount = n
	}

	var maxRetry int
	if n, ok := asynq.GetMaxRetry(ctx); ok {
		maxRetry = n
	}

	var deadline *time.Time
	if t, ok := ctx.Deadline(); ok {
		deadline = &t
	}

	return &Delivery{
		id:           id,
		jobType:      task.Type(),
		queue:        queue,
		retryCount:   retryCount,
		maxRetry:     maxRetry,
		deadline:     deadline,
		rawPayload:   append([]byte{}, task.Payload()...),
		resultWriter: resultWriter,
	}, nil
}

// ID returns the runtime task ID assigned by the queue backend.
func (d *Delivery) ID() string {
	if d == nil {
		return ""
	}
	return d.id
}

// Type returns the job type handled by this delivery.
func (d *Delivery) Type() string {
	if d == nil {
		return ""
	}
	return d.jobType
}

// Queue returns the queue that delivered the job.
func (d *Delivery) Queue() string {
	if d == nil {
		return ""
	}
	return d.queue
}

// Attempt returns the one-based processing attempt number.
func (d *Delivery) Attempt() int {
	if d == nil {
		return 0
	}
	return d.retryCount + 1
}

// RetryCount returns the number of times this delivery has already been retried.
func (d *Delivery) RetryCount() int {
	if d == nil {
		return 0
	}
	return d.retryCount
}

// MaxRetry returns the maximum retry count configured for this delivery.
func (d *Delivery) MaxRetry() int {
	if d == nil {
		return 0
	}
	return d.maxRetry
}

// Deadline returns the effective processing deadline for this delivery.
func (d *Delivery) Deadline() (time.Time, bool) {
	if d == nil || d.deadline == nil {
		return time.Time{}, false
	}
	return *d.deadline, true
}

// DecodePayload decodes the raw delivery payload into v.
func (d *Delivery) DecodePayload(v any) error {
	payloadBytes, err := d.payloadBytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(payloadBytes, v)
}

func (d *Delivery) payloadBytes() ([]byte, error) {
	if d == nil {
		return nil, ErrInvalidAsynqTask
	}
	return append([]byte{}, d.rawPayload...), nil
}

// WriteResult writes the result of the delivery to the queue backend.
func (d *Delivery) WriteResult(result any) error {
	if d == nil || d.resultWriter == nil {
		return ErrResultWriterNotSet
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to serialize result: %w: %w", ErrSerializationFailure, err)
	}

	if _, err = d.resultWriter.Write(resultBytes); err != nil {
		return fmt.Errorf("failed to write result: %w: %w", ErrFailedToWriteResult, err)
	}

	return nil
}
