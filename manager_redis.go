package queue

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"
)

// RedisInfo retrieves information from the Redis server or cluster.
func (m *Manager) RedisInfo(ctx context.Context) (*RedisInfo, error) {
	if ctx == nil {
		return nil, ErrInvalidContext
	}

	switch client := m.client.(type) {
	case *redis.ClusterClient:
		return m.getRedisClusterInfo(ctx, client)
	case *redis.Client:
		return getRedisStandardInfo(ctx, client)
	default:
		return nil, ErrRedisClientNotSupported
	}
}

func getRedisStandardInfo(ctx context.Context, client *redis.Client) (*RedisInfo, error) {
	rawInfo, err := client.Info(ctx, "all").Result()
	if err != nil {
		return nil, mapRedisInfoError(err)
	}
	info := parseRedisInfo(rawInfo)
	return &RedisInfo{
		Address:   client.Options().Addr,
		Info:      info,
		RawInfo:   rawInfo,
		IsCluster: false,
	}, nil
}

func (m *Manager) getRedisClusterInfo(ctx context.Context, client *redis.ClusterClient) (*RedisInfo, error) {
	rawInfo, err := client.Info(ctx).Result()
	if err != nil {
		return nil, mapRedisInfoError(err)
	}
	clusterNodes, err := client.ClusterNodes(ctx).Result()
	if err != nil {
		return nil, mapRedisInfoError(err)
	}
	info := parseRedisInfo(rawInfo)

	queueLocations, err := m.fetchQueueLocations()
	if err != nil {
		return nil, mapRedisInfoError(err)
	}

	return &RedisInfo{
		Address:        strings.Join(client.Options().Addrs, ","),
		Info:           info,
		RawInfo:        rawInfo,
		IsCluster:      true,
		ClusterNodes:   clusterNodes,
		QueueLocations: queueLocations,
	}, nil
}

func (m *Manager) fetchQueueLocations() ([]*QueueLocation, error) {
	queues, err := m.inspector.Queues()
	if err != nil {
		return nil, mapManagerError(err)
	}

	locations := make([]*QueueLocation, len(queues))
	for i, queue := range queues {
		keySlot, err := m.inspector.ClusterKeySlot(queue)
		if err != nil {
			return nil, mapManagerError(err)
		}

		nodes, err := m.inspector.ClusterNodes(queue)
		if err != nil {
			return nil, mapManagerError(err)
		}

		nodeAddrs := make([]string, len(nodes))
		for j, node := range nodes {
			nodeAddrs[j] = node.Addr
		}

		locations[i] = &QueueLocation{
			Queue:   queue,
			KeySlot: keySlot,
			Nodes:   nodeAddrs,
		}
	}

	return locations, nil
}

func parseRedisInfo(infoStr string) map[string]string {
	info := make(map[string]string)
	for line := range strings.SplitSeq(infoStr, "\r\n") {
		if key, value, ok := strings.Cut(line, ":"); ok {
			info[key] = value
		}
	}
	return info
}
