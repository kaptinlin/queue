package main

import (
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/kaptinlin/queue"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Initialize the Redis client with the necessary configuration.
	redisConfig := &queue.RedisConfig{
		Addr: "localhost:6379",
		DB:   0, // Default DB
	}

	// Convert the RedisConfig to asynq.RedisClientOpt to use with asynq.NewInspector.
	asynqRedisOpt := redisConfig.ToAsynqRedisOpt()

	// Initialize the asynq Inspector using the Redis client options.
	inspector := asynq.NewInspector(asynqRedisOpt)

	// Make Redis Client
	redisClient := asynqRedisOpt.MakeRedisClient().(redis.UniversalClient)

	// Create an instance of Manager using the Redis client and the asynq Inspector.
	manager := queue.NewManager(redisClient, inspector)

	// Example operation: Listing all workers.
	workers, err := manager.ListWorkers()
	if err != nil {
		fmt.Printf("Error listing workers: %v\n", err)
		return
	}
	for i, worker := range workers {
		fmt.Printf("Worker %d: ID=%s, Host=%s, Status=%s\n", i+1, worker.ID, worker.Host, worker.Status)
	}

	// Example operation: Listing all queues.
	queues, err := manager.ListQueues()
	if err != nil {
		fmt.Printf("Error listing queues: %v\n", err)
		return
	}

	for i, queue := range queues {
		fmt.Printf("Queue %d: Name=%s, Size=%d, Processed=%d, Succeeded=%d, Failed=%d\n", i+1,
			queue.Queue, queue.Size, queue.Processed, queue.Succeeded, queue.Failed)
	}

	// Example operation: Getting queue information.
	queueInfo, err := manager.GetQueueInfo("default")
	if err != nil {
		fmt.Printf("Error getting queue information: %v\n", err)
		return
	}
	fmt.Printf("Queue Info: Name=%s, Size=%d, Processed=%d, Succeeded=%d, Failed=%d\n",
		queueInfo.Queue, queueInfo.Size, queueInfo.Processed, queueInfo.Succeeded, queueInfo.Failed)

	// Example operation: Listing queue stats.
	dailyStats, err := manager.ListQueueStats("default", 7)
	if err != nil {
		fmt.Printf("Error getting queue stats: %v\n", err)
		return
	}
	fmt.Println("Daily Stats:")
	for i, stats := range dailyStats {
		fmt.Printf("Day %d: Date=%s, Processed=%d, Succeeded=%d, Failed=%d\n",
			i+1, stats.Date.Format("2006-01-02"), stats.Processed, stats.Succeeded, stats.Failed)
	}
}
