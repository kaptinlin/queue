package queue

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestIsQueueNotFoundError_True(t *testing.T) {
	//nolint:err113
	err := errors.New("queue does not exist")
	assert.True(t, isQueueNotFoundError(err))
}

func TestIsQueueNotFoundError_False(t *testing.T) {
	//nolint:err113
	err := errors.New("some other error")
	assert.False(t, isQueueNotFoundError(err))
}
