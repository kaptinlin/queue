package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kaptinlin/queue"
)

func TestManagerRedisInfo_UnsupportedClient(t *testing.T) {
	manager := queue.NewManager(nil, nil)

	assert.NotPanics(t, func() {
		info, err := manager.RedisInfo(context.Background())
		assert.Nil(t, info)
		assert.ErrorIs(t, err, queue.ErrRedisClientNotSupported)
	})
}
