package tests

import (
	"testing"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
)

// --- WithRedisUsername ---

func TestWithRedisUsername(t *testing.T) {
	config := queue.NewRedisConfig(
		queue.WithRedisUsername("testuser"),
	)
	assert.Equal(t, "testuser", config.Username)
}
