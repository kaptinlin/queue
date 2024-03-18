package tests

import (
	"flag"
	"os"
	"testing"

	"github.com/kaptinlin/queue"
)

var (
	redisAddr = "localhost:6379"
	redisDB   = 0
)

func TestMain(m *testing.M) {
	flag.StringVar(&redisAddr, "redis_addr", "localhost:6379", "Redis address to use in testing")
	flag.IntVar(&redisDB, "redis_db", 0, "Redis DB number to use in testing")
	flag.Parse()

	os.Exit(m.Run())
}

func getRedisConfig() *queue.RedisConfig {
	return queue.NewRedisConfig(
		queue.WithRedisAddress(redisAddr),
		queue.WithRedisDB(redisDB),
	)
}
