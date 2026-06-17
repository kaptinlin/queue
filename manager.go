package queue

import (
	"errors"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// Manager provides operational inspection and management APIs for queues and jobs.
type Manager struct {
	client         redis.UniversalClient
	inspector      *asynq.Inspector
	closeClient    bool
	closeInspector bool
}

// NewManager creates a Manager for operational inspection and state changes.
func NewManager(redisConfig *RedisConfig) (*Manager, error) {
	if redisConfig == nil {
		return nil, ErrInvalidRedisConfig
	}

	redisOpt := asynqRedisOpt(redisConfig)
	client, ok := redisOpt.MakeRedisClient().(redis.UniversalClient)
	if !ok {
		return nil, errInvalidManagerClient
	}
	inspector := asynq.NewInspector(redisOpt)

	manager, err := newManager(client, inspector)
	if err != nil {
		_ = inspector.Close()
		_ = client.Close()
		return nil, err
	}
	manager.closeClient = true
	manager.closeInspector = true
	return manager, nil
}

func newManager(client redis.UniversalClient, inspector *asynq.Inspector) (*Manager, error) {
	if client == nil {
		return nil, errInvalidManagerClient
	}
	if inspector == nil {
		return nil, errInvalidManagerInspector
	}
	return &Manager{
		client:    client,
		inspector: inspector,
	}, nil
}

// Close releases the Redis resources owned by the manager.
func (m *Manager) Close() error {
	if m == nil {
		return nil
	}

	var errs []error
	if m.closeInspector && m.inspector != nil {
		errs = append(errs, m.inspector.Close())
	}
	if m.closeClient && m.client != nil {
		errs = append(errs, m.client.Close())
	}
	return errors.Join(errs...)
}
