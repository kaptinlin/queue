package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kaptinlin/queue"
)

// --- WithRedisUsername ---

func TestWithRedisUsername(t *testing.T) {
	config := queue.NewRedisConfig(
		queue.WithRedisUsername("testuser"),
	)
	assert.Equal(t, "testuser", config.Username)
}
