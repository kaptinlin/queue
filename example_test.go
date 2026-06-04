package queue_test

import (
	"context"
	"fmt"
	"time"

	"github.com/kaptinlin/queue"
)

func ExampleNewJob() {
	job, err := queue.NewJob("email:send", map[string]string{
		"to":      "user@example.com",
		"subject": "Welcome",
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(job.Type())
	fmt.Println(job.Options().Queue)
	// Output:
	// email:send
	// default
}

func ExampleNewJob_withOptions() {
	job, err := queue.NewJob("email:send", map[string]string{
		"to": "user@example.com",
	},
		queue.WithQueue("critical"),
		queue.WithMaxRetries(3),
		queue.WithDelay(5*time.Second),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	options := job.Options()

	fmt.Println(job.Type())
	fmt.Println(options.Queue)
	fmt.Println(options.MaxRetries)
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

	job, err := queue.NewJob("email:send", map[string]string{
		"to":      "user@example.com",
		"subject": "Welcome",
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

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
	handler, err := queue.NewHandler("email:send",
		func(ctx context.Context, delivery *queue.Delivery) error {
			fmt.Println("processing:", delivery.Type())
			return nil
		},
		queue.WithJobQueue("critical"),
		queue.WithJobTimeout(30*time.Second),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(handler.Type())
	fmt.Println(handler.Queue())
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
	// skip retry due to: invalid payload format: skip retry
}

func ExampleNewMemoryConfigProvider() {
	provider := queue.NewMemoryConfigProvider()

	job, err := queue.NewJob("report:generate", nil,
		queue.WithQueue("reports"),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	id, err := provider.RegisterCronJob("hourly-report", "0 * * * *", job)
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
