package main

import (
	"log"

	"github.com/kaptinlin/queue"
	"github.com/kaptinlin/queue/examples/struct/jobs"
)

func main() {
	// Initialize Redis configuration and worker.
	redisConfig := queue.NewRedisConfig(queue.WithRedisAddress("localhost:6379"))
	worker, err := queue.NewWorker(redisConfig)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	// Register handler and start the worker.
	handler := jobs.NewExampleHandler()
	if err = worker.RegisterHandler(handler); err != nil {
		log.Fatalf("Failed to register job handler: %v", err)
	}
	if err = worker.Start(); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}
}
