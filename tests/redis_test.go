package tests

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kaptinlin/queue"
)

func TestNewRedisConfigValidates(t *testing.T) {
	t.Parallel()

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	tests := []struct {
		name    string
		options []queue.RedisOption
		wantErr error
	}{
		{
			name: "valid configuration",
			options: []queue.RedisOption{
				queue.WithRedisNetwork("tcp"),
				queue.WithRedisAddress("localhost:6379"),
			},
		},
		{
			name: "empty address",
			options: []queue.RedisOption{
				queue.WithRedisAddress(""),
			},
			wantErr: queue.ErrRedisEmptyAddress,
		},
		{
			name: "unsupported network type",
			options: []queue.RedisOption{
				queue.WithRedisNetwork("unsupported"),
			},
			wantErr: queue.ErrRedisUnsupportedNetwork,
		},
		{
			name: "invalid address format",
			options: []queue.RedisOption{
				queue.WithRedisAddress("invalid-format"),
			},
			wantErr: queue.ErrRedisInvalidAddress,
		},
		{
			name: "missing TLS config for secure connection",
			options: []queue.RedisOption{
				queue.WithRedisAddress("rediss://localhost:6379"),
			},
			wantErr: queue.ErrRedisTLSRequired,
		},
		{
			name: "secure connection still requires host port syntax",
			options: []queue.RedisOption{
				queue.WithRedisAddress("rediss://localhost:6379"),
				queue.WithRedisTLSConfig(tlsConfig),
			},
			wantErr: queue.ErrRedisInvalidAddress,
		},
		{
			name: "unix socket address",
			options: []queue.RedisOption{
				queue.WithRedisNetwork("unix"),
				queue.WithRedisAddress("/tmp/redis.sock"),
			},
		},
		{
			name: "negative DB",
			options: []queue.RedisOption{
				queue.WithRedisDB(-1),
			},
			wantErr: queue.ErrRedisInvalidDB,
		},
		{
			name: "negative pool size",
			options: []queue.RedisOption{
				queue.WithRedisPoolSize(-1),
			},
			wantErr: queue.ErrRedisInvalidPoolSize,
		},
		{
			name: "negative timeout",
			options: []queue.RedisOption{
				queue.WithRedisDialTimeout(-time.Second),
			},
			wantErr: queue.ErrRedisInvalidTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config, err := queue.NewRedisConfig(tt.options...)
			if tt.wantErr != nil {
				assert.Nil(t, config)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NotNil(t, config)
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisConfigCopiesTLSConfig(t *testing.T) {
	t.Parallel()

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	config, err := queue.NewRedisConfig(queue.WithRedisTLSConfig(tlsConfig))
	assert.NoError(t, err)

	got := config.TLSConfig()
	assert.NotNil(t, got)
	assert.NotSame(t, tlsConfig, got)
	assert.Equal(t, uint16(tls.VersionTLS12), got.MinVersion)

	tlsConfig.MinVersion = tls.VersionTLS13
	assert.Equal(t, uint16(tls.VersionTLS12), config.TLSConfig().MinVersion)
}

func TestRedisConfigOptions(t *testing.T) {
	tests := []struct {
		name     string
		option   queue.RedisOption
		validate func(t *testing.T, config *queue.RedisConfig)
	}{
		{
			name:   "WithRedisAddress",
			option: queue.WithRedisAddress("127.0.0.1:6379"),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.Equal(t, "127.0.0.1:6379", config.Addr(), "WithRedisAddress() should set correct address")
			},
		},
		{
			name:   "WithRedisPassword",
			option: queue.WithRedisPassword("secret"),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.Equal(t, "secret", config.Password(), "WithRedisPassword() should set correct password")
			},
		},
		{
			name:   "WithRedisDB",
			option: queue.WithRedisDB(1),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.Equal(t, 1, config.DB(), "WithRedisDB() should set correct DB")
			},
		},
		{
			name:   "WithRedisTLSConfig",
			option: queue.WithRedisTLSConfig(&tls.Config{}),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.NotNil(t, config.TLSConfig(), "WithRedisTLSConfig() should set TLS config")
			},
		},
		{
			name:   "WithRedisDialTimeout",
			option: queue.WithRedisDialTimeout(10 * time.Second),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.Equal(t, 10*time.Second, config.DialTimeout(), "WithRedisDialTimeout() should set correct timeout")
			},
		},
		{
			name:   "WithRedisReadTimeout",
			option: queue.WithRedisReadTimeout(10 * time.Second),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.Equal(t, 10*time.Second, config.ReadTimeout(), "WithRedisReadTimeout() should set correct timeout")
			},
		},
		{
			name:   "WithRedisWriteTimeout",
			option: queue.WithRedisWriteTimeout(10 * time.Second),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.Equal(t, 10*time.Second, config.WriteTimeout(), "WithRedisWriteTimeout() should set correct timeout")
			},
		},
		{
			name:   "WithRedisPoolSize",
			option: queue.WithRedisPoolSize(20),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.Equal(t, 20, config.PoolSize(), "WithRedisPoolSize() should set correct pool size")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := queue.NewRedisConfig(tt.option)
			assert.NoError(t, err)
			tt.validate(t, config)
		})
	}
}
