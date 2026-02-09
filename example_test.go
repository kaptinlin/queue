package queue_test

import (
	"context"
	"fmt"
	"time"

	"github.com/kaptinlin/queue"
)

func ExampleNewJob() {
	job := queue.NewJob("email:send", map[string]string{
		"to":      "user@example.com",
		"subject": "Welcome",
	})

	fmt.Println(job.Type)
	fmt.Println(job.Options.Queue)
	// Output:
	// email:send
	// default
}

func ExampleNewJob_withOptions() {
	job := queue.NewJob("email:send", map[string]string{
		"to": "user@example.com",
	},
		queue.WithQueue("critical"),
		queue.WithMaxRetries(3),
		queue.WithDelay(5*time.Second),
	)

	fmt.Println(job.Type)
	fmt.Println(job.Options.Queue)
	fmt.Println(job.Options.MaxRetries)
	// Output:
	// email:send
	// critical
	// 3
}

func ExampleJob_DecodePayload() {
	type EmailPayload struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
	}

	job := queue.NewJob("email:send", map[string]string{
		"to":      "user@example.com",
		"subject": "Welcome",
	})

	var payload EmailPayload
	if err := job.DecodePayload(&payload); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(payload.To)
	fmt.Println(payload.Subject)
	// Output:
	// user@example.com
	// Welcome
}

func ExampleNewRedisConfig() {
	config := queue.NewRedisConfig(
		queue.WithRedisAddress("localhost:6379"),
		queue.WithRedisDB(1),
		queue.WithRedisPassword("secret"),
	)

	fmt.Println(config.Addr)
	fmt.Println(config.DB)
	// Output:
	// localhost:6379
	// 1
}

func ExampleDefaultRedisConfig() {
	config := queue.DefaultRedisConfig()

	fmt.Println(config.Addr)
	fmt.Println(config.Network)
	fmt.Println(config.DB)
	// Output:
	// localhost:6379
	// tcp
	// 0
}

func ExampleNewHandler() {
	handler := queue.NewHandler("email:send",
		func(ctx context.Context, job *queue.Job) error {
			fmt.Println("processing:", job.Type)
			return nil
		},
		queue.WithJobQueue("critical"),
		queue.WithJobTimeout(30*time.Second),
	)

	fmt.Println(handler.JobType)
	fmt.Println(handler.JobQueue)
	// Output:
	// email:send
	// critical
}

func ExampleIsValidJobState() {
	fmt.Println(queue.IsValidJobState(queue.StateActive))
	fmt.Println(queue.IsValidJobState(queue.StatePending))
	fmt.Println(queue.IsValidJobState("invalid"))
	// Output:
	// true
	// true
	// false
}

func ExampleNewErrRateLimit() {
	err := queue.NewErrRateLimit(10 * time.Second)

	fmt.Println(err.Error())
	fmt.Println(queue.IsErrRateLimit(err))
	// Output:
	// rate limited: retry after 10s
	// true
}

func ExampleNewSkipRetryError() {
	err := queue.NewSkipRetryError("invalid payload format")

	fmt.Println(err)
	// Output:
	// skip retry due to: invalid payload format: skip retry for the task
}

func ExampleNewMemoryConfigProvider() {
	provider := queue.NewMemoryConfigProvider()

	job := queue.NewJob("report:generate", nil,
		queue.WithQueue("reports"),
	)

	id, err := provider.RegisterCronJob("0 * * * *", job)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("registered:", id != "")
	// Output:
	// registered: true
}

func ExampleNewDefaultLogger() {
	logger := queue.NewDefaultLogger()
	logger.Info("queue started")
	// The logger outputs via slog; no captured output.
}
