package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
)

func TestNewManager_ValidatesRedisConfig(t *testing.T) {
	t.Parallel()

	_, err := queue.NewManager(nil)
	assert.ErrorIs(t, err, queue.ErrInvalidRedisConfig)

	_, err = queue.NewRedisConfig(
		queue.WithRedisAddress("127.0.0.1:1"),
		queue.WithRedisDialTimeout(-time.Millisecond),
	)
	assert.ErrorIs(t, err, queue.ErrRedisInvalidTimeout)
}

func TestNewManager_ConstructsManager(t *testing.T) {
	t.Parallel()

	redisConfig, err := queue.NewRedisConfig(
		queue.WithRedisAddress("127.0.0.1:1"),
		queue.WithRedisDialTimeout(time.Millisecond),
		queue.WithRedisReadTimeout(time.Millisecond),
		queue.WithRedisWriteTimeout(time.Millisecond),
		queue.WithRedisPoolSize(1),
	)
	require.NoError(t, err)

	manager, err := queue.NewManager(redisConfig)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, manager.Close())
	})
	assert.NotNil(t, manager)
}
