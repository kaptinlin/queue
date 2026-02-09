package tests

import (
	"testing"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- WithClientLogger ---

func TestWithClientLogger(t *testing.T) {
	logger := &mockLogger{}
	client, err := queue.NewClient(getRedisConfig(),
		queue.WithClientLogger(logger),
	)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NoError(t, client.Stop())
}

// --- NewClient validation ---

func TestNewClient_NilRedisConfig(t *testing.T) {
	_, err := queue.NewClient(nil)
	assert.ErrorIs(t, err, queue.ErrInvalidRedisConfig)
}

func TestNewClient_InvalidRedisConfig(t *testing.T) {
	cfg := &queue.RedisConfig{Network: "bad", Addr: ""}
	_, err := queue.NewClient(cfg)
	assert.Error(t, err)
}

// --- EnqueueJob with client retention ---

func TestClientEnqueueJob_WithClientRetention(t *testing.T) {
	client, err := queue.NewClient(getRedisConfig(),
		queue.WithClientRetention(1),
	)
	require.NoError(t, err)
	defer func() { assert.NoError(t, client.Stop()) }()

	id, err := client.Enqueue("retention_test",
		map[string]string{"k": "v"})
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}
