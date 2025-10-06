package tests

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
	"github.com/stretchr/testify/assert"
)

func TestRedisConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  queue.RedisConfig
		wantErr bool
	}{
		{
			name: "Valid configuration",
			config: queue.RedisConfig{
				Network: "tcp",
				Addr:    "localhost:6379",
			},
			wantErr: false,
		},
		{
			name: "Empty address",
			config: queue.RedisConfig{
				Network: "tcp",
				Addr:    "",
			},
			wantErr: true,
		},
		{
			name: "Unsupported network type",
			config: queue.RedisConfig{
				Network: "unsupported",
				Addr:    "localhost:6379",
			},
			wantErr: true,
		},
		{
			name: "Invalid address format",
			config: queue.RedisConfig{
				Network: "tcp",
				Addr:    "invalid-format",
			},
			wantErr: true,
		},
		{
			name: "Missing TLS config for secure connection",
			config: queue.RedisConfig{
				Network: "tcp",
				Addr:    "rediss://localhost:6379",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err, "Validate() should return error")
			} else {
				assert.NoError(t, err, "Validate() should not return error")
			}
		})
	}
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
