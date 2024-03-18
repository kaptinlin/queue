package jobs

import (
	"context"
	"fmt"

	"github.com/kaptinlin/queue"
)

const ExampleJobType = "example_job"

type ExampleJobPayload struct {
	Input string `json:"input"`
}

// Creates a new ExampleJob instance.
func NewExampleJob(payload ExampleJobPayload) *queue.Job {
	return queue.NewJob(ExampleJobType, payload)
}

// Creates a handler for ExampleJob.
func NewExampleHandler() *queue.Handler {
	return queue.NewHandler(ExampleJobType, HandleExampleJob)
}

// Processes ExampleJob.
func HandleExampleJob(ctx context.Context, job *queue.Job) error {
	var payload ExampleJobPayload
	if err := job.DecodePayload(&payload); err != nil {
		return err
	}
	fmt.Printf("Processing job with input: %s\n", payload.Input)
	return nil
}
