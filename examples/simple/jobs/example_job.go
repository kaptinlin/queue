package jobs

import (
	"context"
	"fmt"

	"github.com/kaptinlin/queue"
)

type ExampleJobPayload struct {
	Input string `json:"input"`
}

func HandleExampleJob(ctx context.Context, job *queue.Job) error {
	var payload ExampleJobPayload
	if err := job.DecodePayload(&payload); err != nil {
		return err
	}

	fmt.Printf("Processing job with input: %s\n", payload.Input)
	return nil
}
