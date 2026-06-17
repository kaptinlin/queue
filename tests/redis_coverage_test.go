package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kaptinlin/queue"
)

// --- WithRedisUsername ---

func TestWithRedisUsername(t *testing.T) {
	config, err := queue.NewRedisConfig(
		queue.WithRedisUsername("testuser"),
	)
	assert.NoError(t, err)
	assert.Equal(t, "testuser", config.Username())
}
