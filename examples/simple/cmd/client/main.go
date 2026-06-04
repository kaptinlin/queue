// Package main shows how to enqueue a job with the simple example client.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/kaptinlin/queue"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	redisConfig := queue.NewRedisConfig(
		queue.WithRedisAddress("localhost:6379"),
	)

	client, err := queue.NewClient(redisConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("failed to close client: %v", err)
		}
	}()

	jobType := "example_job"
	payload := map[string]any{
		"input": "Hello, Queue!",
	}
	_, err = client.Enqueue(jobType, payload, queue.WithQueue("critical"), queue.WithRetention(time.Hour))
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	log.Println("Job enqueued successfully")
	return nil
}
