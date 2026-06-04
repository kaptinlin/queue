package queue

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRedisInfo(t *testing.T) {
	raw := "redis_version:7.0.0\r\nused_memory:1024\r\n# Server\r\n"
	info := parseRedisInfo(raw)
	assert.Equal(t, "7.0.0", info["redis_version"])
	assert.Equal(t, "1024", info["used_memory"])
}

func TestParseRedisInfo_Empty(t *testing.T) {
	info := parseRedisInfo("")
	assert.Empty(t, info)
}

func TestMapManagerError_Nil(t *testing.T) {
	assert.NoError(t, mapManagerError(nil))
}

func TestMapManagerError_AsynqQueueNotFound(t *testing.T) {
	err := fmt.Errorf("inspector: %w", asynq.ErrQueueNotFound)
	assert.ErrorIs(t, mapManagerError(err), ErrQueueNotFound)
}

func TestMapManagerError_AsynqTaskNotFound(t *testing.T) {
	err := fmt.Errorf("inspector: %w", asynq.ErrTaskNotFound)
	assert.ErrorIs(t, mapManagerError(err), ErrJobNotFound)
}

func TestMapManagerError_AsynqQueueNotEmpty(t *testing.T) {
	err := fmt.Errorf("inspector: %w", asynq.ErrQueueNotEmpty)
	assert.ErrorIs(t, mapManagerError(err), ErrQueueNotEmpty)
}

func TestMapManagerError_Other(t *testing.T) {
	//nolint:err113 // Test the negative case with a one-off error value.
	err := errors.New("some other error")
	assert.ErrorIs(t, mapManagerError(err), err)
}

func TestMapRedisInfoError_Nil(t *testing.T) {
	assert.NoError(t, mapRedisInfoError(nil))
}

func TestMapRedisInfoError_Context(t *testing.T) {
	err := fmt.Errorf("redis info: %w", context.Canceled)
	assert.ErrorIs(t, mapRedisInfoError(err), context.Canceled)
	assert.NotErrorIs(t, mapRedisInfoError(err), ErrRedisUnavailable)
}

func TestMapRedisInfoError_PreservesManagerSentinel(t *testing.T) {
	err := fmt.Errorf("queue location: %w", ErrQueueNotFound)
	assert.ErrorIs(t, mapRedisInfoError(err), ErrQueueNotFound)
	assert.NotErrorIs(t, mapRedisInfoError(err), ErrRedisUnavailable)
}

func TestMapRedisInfoError_WrapsUnavailable(t *testing.T) {
	//nolint:err113 // Test the boundary behavior with a representative driver error.
	err := errors.New("connection refused")
	got := mapRedisInfoError(err)

	assert.ErrorIs(t, got, ErrRedisUnavailable)
	assert.ErrorIs(t, got, err)
}

func TestRedisInfo_NilContext(t *testing.T) {
	redisOpt := DefaultRedisConfig().ToAsynqRedisOpt()
	inspector := asynq.NewInspector(redisOpt)
	client := redisOpt.MakeRedisClient().(redis.UniversalClient)
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})
	manager, err := NewManager(client, inspector)
	require.NoError(t, err)

	info, err := manager.RedisInfo(nil) //nolint:staticcheck // Exercise nil context validation.

	assert.Nil(t, info)
	assert.ErrorIs(t, err, ErrInvalidContext)
}
