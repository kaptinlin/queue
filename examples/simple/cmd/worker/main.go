package main

import (
	"log"

	"github.com/kaptinlin/queue/examples/simple/jobs"

	"github.com/kaptinlin/queue"
)

func main() {
	redisConfig := queue.NewRedisConfig(
		queue.WithRedisAddress("localhost:6379"),
	)

	worker, err := queue.NewWorker(redisConfig, queue.WithWorkerQueues(
		map[string]int{
			"critical": 6,
			"default":  3,
			"low":      1,
		},
	))
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	err = worker.Register("example_job", jobs.HandleExampleJob, queue.WithJobQueue("critical"))
	if err != nil {
		log.Fatalf("Failed to register job handler: %v", err)
	}

	if err := worker.Start(); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}
}
