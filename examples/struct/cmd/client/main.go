package main

import (
	"log"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/kaptinlin/queue/examples/struct/jobs"
)

func main() {
	// Set up Redis configuration and client.
	redisConfig := queue.NewRedisConfig(queue.WithRedisAddress("localhost:6379"))
	client, err := queue.NewClient(redisConfig, queue.WithClientRetention(24*time.Hour))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Prepare and enqueue an ExampleJob.
	payload := jobs.ExampleJobPayload{Input: "Hello, Queue!"}
	job := jobs.NewExampleJob(payload)
	_, err = client.EnqueueJob(job)
	if err != nil {
		log.Fatalf("Failed to enqueue job: %v", err)
	}

	log.Println("Job enqueued successfully")
}
