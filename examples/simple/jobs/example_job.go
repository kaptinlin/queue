// Package jobs contains the simple example job definitions.
package jobs

import (
	"context"
	"fmt"

	"github.com/kaptinlin/queue"
)

// ExampleJobPayload carries the input for the simple example job.
type ExampleJobPayload struct {
	Input string `json:"input"`
}

// HandleExampleJob processes the simple example job.
func HandleExampleJob(ctx context.Context, job *queue.Job) error {
	var payload ExampleJobPayload
	if err := job.DecodePayload(&payload); err != nil {
		return err
	}

	fmt.Printf("Processing job with input: %s\n", payload.Input)
	return nil
}
