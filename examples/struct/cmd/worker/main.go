// Package main shows how to run a worker for the structured example.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	handler, err := jobs.NewExampleHandler()
	if err != nil {
		log.Fatalf("Failed to create job handler: %v", err)
	}
	if err = worker.RegisterHandler(handler); err != nil {
		log.Fatalf("Failed to register job handler: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err = worker.Run(ctx); err != nil {
		log.Printf("Worker stopped with error: %v", err)
		return
	}
}
