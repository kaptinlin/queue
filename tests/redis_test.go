package tests

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	"github.com/kaptinlin/queue"
)

func TestRedisConfigValidate(t *testing.T) {
	t.Parallel()

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	tests := []struct {
		name    string
		config  queue.RedisConfig
		wantErr error
	}{
		{
			name: "Valid configuration",
			config: queue.RedisConfig{
				Network: "tcp",
				Addr:    "localhost:6379",
			},
		},
		{
			name: "Empty address",
			config: queue.RedisConfig{
				Network: "tcp",
				Addr:    "",
			},
			wantErr: queue.ErrRedisEmptyAddress,
		},
		{
			name: "Unsupported network type",
			config: queue.RedisConfig{
				Network: "unsupported",
				Addr:    "localhost:6379",
			},
			wantErr: queue.ErrRedisUnsupportedNetwork,
		},
		{
			name: "Invalid address format",
			config: queue.RedisConfig{
				Network: "tcp",
				Addr:    "invalid-format",
			},
			wantErr: queue.ErrRedisInvalidAddress,
		},
		{
			name: "Missing TLS config for secure connection",
			config: queue.RedisConfig{
				Network: "tcp",
				Addr:    "rediss://localhost:6379",
			},
			wantErr: queue.ErrRedisTLSRequired,
		},
		{
			name: "Secure connection with TLS config",
			config: queue.RedisConfig{
				Network:   "tcp",
				Addr:      "rediss://localhost:6379",
				TLSConfig: tlsConfig,
			},
			wantErr: queue.ErrRedisInvalidAddress,
		},
		{
			name: "Unix socket address",
			config: queue.RedisConfig{
				Network: "unix",
				Addr:    "/tmp/redis.sock",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisConfigToAsynqRedisOpt(t *testing.T) {
	t.Parallel()

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	config := &queue.RedisConfig{
		Network:      "tcp",
		Addr:         "redis.example:6380",
		Username:     "user",
		Password:     "secret",
		DB:           3,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 4 * time.Second,
		PoolSize:     12,
		TLSConfig:    tlsConfig,
	}

	got := config.ToAsynqRedisOpt()
	type redisOptSnapshot struct {
		Network      string
		Addr         string
		Username     string
		Password     string
		DB           int
		DialTimeout  time.Duration
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		PoolSize     int
	}
	want := redisOptSnapshot{
		Network:      config.Network,
		Addr:         config.Addr,
		Username:     config.Username,
		Password:     config.Password,
		DB:           config.DB,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolSize:     config.PoolSize,
	}
	gotSnapshot := redisOptSnapshot{
		Network:      got.Network,
		Addr:         got.Addr,
		Username:     got.Username,
		Password:     got.Password,
		DB:           got.DB,
		DialTimeout:  got.DialTimeout,
		ReadTimeout:  got.ReadTimeout,
		WriteTimeout: got.WriteTimeout,
		PoolSize:     got.PoolSize,
	}
	if diff := cmp.Diff(want, gotSnapshot); diff != "" {
		t.Errorf("asynq redis option mismatch (-want +got):\n%s", diff)
	}
	assert.Same(t, tlsConfig, got.TLSConfig)
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
				assert.Equal(t, "127.0.0.1:6379", config.Addr, "WithRedisAddress() should set correct address")
			},
		},
		{
			name:   "WithRedisPassword",
			option: queue.WithRedisPassword("secret"),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.Equal(t, "secret", config.Password, "WithRedisPassword() should set correct password")
			},
		},
		{
			name:   "WithRedisDB",
			option: queue.WithRedisDB(1),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.Equal(t, 1, config.DB, "WithRedisDB() should set correct DB")
			},
		},
		{
			name:   "WithRedisTLSConfig",
			option: queue.WithRedisTLSConfig(&tls.Config{}),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.NotNil(t, config.TLSConfig, "WithRedisTLSConfig() should set TLS config")
			},
		},
		{
			name:   "WithRedisDialTimeout",
			option: queue.WithRedisDialTimeout(10 * time.Second),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.Equal(t, 10*time.Second, config.DialTimeout, "WithRedisDialTimeout() should set correct timeout")
			},
		},
		{
			name:   "WithRedisReadTimeout",
			option: queue.WithRedisReadTimeout(10 * time.Second),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.Equal(t, 10*time.Second, config.ReadTimeout, "WithRedisReadTimeout() should set correct timeout")
			},
		},
		{
			name:   "WithRedisWriteTimeout",
			option: queue.WithRedisWriteTimeout(10 * time.Second),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.Equal(t, 10*time.Second, config.WriteTimeout, "WithRedisWriteTimeout() should set correct timeout")
			},
		},
		{
			name:   "WithRedisPoolSize",
			option: queue.WithRedisPoolSize(20),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				assert.Equal(t, 20, config.PoolSize, "WithRedisPoolSize() should set correct pool size")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply option to a fresh configuration based on default settings
			config := queue.NewRedisConfig()
			tt.option(config)
			tt.validate(t, config) // Use a validation function to check the result
		})
	}
}
