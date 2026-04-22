// Package jobs contains the structured example job definitions.
package jobs

import (
	"context"
	"fmt"

	"github.com/kaptinlin/queue"
)

// ExampleJobType identifies the structured example job.
const ExampleJobType = "example_job"

// ExampleJobPayload carries the input for the structured example job.
type ExampleJobPayload struct {
	Input string `json:"input"`
}

// NewExampleJob creates a structured example job.
func NewExampleJob(payload ExampleJobPayload) *queue.Job {
	return queue.NewJob(ExampleJobType, payload)
}

// NewExampleHandler creates the structured example handler.
func NewExampleHandler() *queue.Handler {
	return queue.NewHandler(ExampleJobType, HandleExampleJob)
}

// HandleExampleJob processes the structured example job.
func HandleExampleJob(ctx context.Context, job *queue.Job) error {
	var payload ExampleJobPayload
	if err := job.DecodePayload(&payload); err != nil {
		return err
	}
	fmt.Printf("Processing job with input: %s\n", payload.Input)
	return nil
}
