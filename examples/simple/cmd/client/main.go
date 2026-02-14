package main

import (
	"log"
	"time"

	"github.com/kaptinlin/queue"
)

func main() {
	redisConfig := queue.NewRedisConfig(
		queue.WithRedisAddress("localhost:6379"),
	)

	client, err := queue.NewClient(redisConfig)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	jobType := "example_job"
	payload := map[string]any{
		"input": "Hello, Queue!",
	}
	_, err = client.Enqueue(jobType, payload, queue.WithQueue("critical"), queue.WithRetention(time.Hour))
	if err != nil {
		log.Fatalf("Failed to enqueue job: %v", err)
	}

	log.Println("Job enqueued successfully")
}
