package tests

import (
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaptinlin/queue"
)

func TestNewManager_ValidatesDependencies(t *testing.T) {
	t.Parallel()

	_, err := queue.NewManager(nil, nil)
	assert.ErrorIs(t, err, queue.ErrInvalidManagerClient)

	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: time.Millisecond,
	})
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})

	_, err = queue.NewManager(client, nil)
	assert.ErrorIs(t, err, queue.ErrInvalidManagerInspector)

	inspector := asynq.NewInspector(getRedisConfig().ToAsynqRedisOpt())
	manager, err := queue.NewManager(client, inspector)
	require.NoError(t, err)
	assert.NotNil(t, manager)
}
