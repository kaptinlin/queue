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
func HandleExampleJob(ctx context.Context, delivery *queue.Delivery) error {
	var payload ExampleJobPayload
	if err := delivery.DecodePayload(&payload); err != nil {
		return err
	}

	fmt.Printf("Processing job with input: %s\n", payload.Input)
	return nil
}
