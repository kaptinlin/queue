// Package main shows how to enqueue a structured example job.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/kaptinlin/queue/examples/struct/jobs"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Set up Redis configuration and client.
	redisConfig := queue.NewRedisConfig(queue.WithRedisAddress("localhost:6379"))
	client, err := queue.NewClient(redisConfig, queue.WithClientRetention(24*time.Hour))
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("failed to close client: %v", err)
		}
	}()

	// Prepare and enqueue an ExampleJob.
	payload := jobs.ExampleJobPayload{Input: "Hello, Queue!"}
	job, err := jobs.NewExampleJob(payload)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}
	_, err = client.EnqueueJob(job)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	log.Println("Job enqueued successfully")
	return nil
}
