// Package main shows how to inspect queue state with Manager.
package main

import (
	"fmt"
	"log"

	"github.com/kaptinlin/queue"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Initialize the Redis client with the necessary configuration.
	redisConfig, err := queue.NewRedisConfig(queue.WithRedisAddress("localhost:6379"))
	if err != nil {
		return fmt.Errorf("invalid redis config: %w", err)
	}

	// Create an instance of Manager using the Redis configuration.
	manager, err := queue.NewManager(redisConfig)
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}
	defer func() {
		if err := manager.Close(); err != nil {
			log.Printf("failed to close manager: %v", err)
		}
	}()

	// Example operation: Listing all workers.
	workers, err := manager.ListWorkers()
	if err != nil {
		return fmt.Errorf("error listing workers: %w", err)
	}
	for i, worker := range workers {
		fmt.Printf("Worker %d: ID=%s, Host=%s, Status=%s\n", i+1, worker.ID, worker.Host, worker.Status)
	}

	// Example operation: Listing all queues.
	queues, err := manager.ListQueues()
	if err != nil {
		return fmt.Errorf("error listing queues: %w", err)
	}

	for i, queue := range queues {
		fmt.Printf("Queue %d: Name=%s, Size=%d, Processed=%d, Succeeded=%d, Failed=%d\n", i+1,
			queue.Queue, queue.Size, queue.Processed, queue.Succeeded, queue.Failed)
	}

	// Example operation: Getting queue information.
	queueInfo, err := manager.QueueInfo("default")
	if err != nil {
		return fmt.Errorf("error getting queue information: %w", err)
	}
	fmt.Printf("Queue Info: Name=%s, Size=%d, Processed=%d, Succeeded=%d, Failed=%d\n",
		queueInfo.Queue, queueInfo.Size, queueInfo.Processed, queueInfo.Succeeded, queueInfo.Failed)

	// Example operation: Listing queue stats.
	dailyStats, err := manager.ListQueueStats("default", 7)
	if err != nil {
		return fmt.Errorf("error getting queue stats: %w", err)
	}
	fmt.Println("Daily Stats:")
	for i, stats := range dailyStats {
		fmt.Printf("Day %d: Date=%s, Processed=%d, Succeeded=%d, Failed=%d\n",
			i+1, stats.Date.Format("2006-01-02"), stats.Processed, stats.Succeeded, stats.Failed)
	}
	return nil
}
