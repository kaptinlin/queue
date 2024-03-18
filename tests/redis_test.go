package tests

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/kaptinlin/queue"
)

func TestRedisConfig_Validate(t *testing.T) {
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
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
func TestRedisConfig_Options(t *testing.T) {
	tests := []struct {
		name     string
		option   queue.RedisOption
		validate func(t *testing.T, config *queue.RedisConfig)
	}{
		{
			name:   "WithRedisAddress",
			option: queue.WithRedisAddress("127.0.0.1:6379"),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				if config.Addr != "127.0.0.1:6379" {
					t.Errorf("WithRedisAddress() Addr = %v, want %v", config.Addr, "127.0.0.1:6379")
				}
			},
		},
		{
			name:   "WithRedisPassword",
			option: queue.WithRedisPassword("secret"),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				if config.Password != "secret" {
					t.Errorf("WithRedisPassword() Password = %v, want %v", config.Password, "secret")
				}
			},
		},
		{
			name:   "WithRedisDB",
			option: queue.WithRedisDB(1),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				if config.DB != 1 {
					t.Errorf("WithRedisDB() DB = %v, want %v", config.DB, 1)
				}
			},
		},
		{
			name:   "WithRedisTLSConfig",
			option: queue.WithRedisTLSConfig(&tls.Config{}),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				if config.TLSConfig == nil {
					t.Errorf("WithRedisTLSConfig() TLSConfig = %v, want %v", config.TLSConfig, "&tls.Config{}")
				}
			},
		},
		{
			name:   "WithRedisDialTimeout",
			option: queue.WithRedisDialTimeout(10 * time.Second),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				if config.DialTimeout != 10*time.Second {
					t.Errorf("WithRedisDialTimeout() DialTimeout = %v, want %v", config.DialTimeout, 10*time.Second)
				}
			},
		},
		{
			name:   "WithRedisReadTimeout",
			option: queue.WithRedisReadTimeout(10 * time.Second),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				if config.ReadTimeout != 10*time.Second {
					t.Errorf("WithRedisReadTimeout() ReadTimeout = %v, want %v", config.ReadTimeout, 10*time.Second)
				}
			},
		},
		{
			name:   "WithRedisWriteTimeout",
			option: queue.WithRedisWriteTimeout(10 * time.Second),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				if config.WriteTimeout != 10*time.Second {
					t.Errorf("WithRedisWriteTimeout() WriteTimeout = %v, want %v", config.WriteTimeout, 10*time.Second)
				}
			},
		},
		{
			name:   "WithRedisPoolSize",
			option: queue.WithRedisPoolSize(20),
			validate: func(t *testing.T, config *queue.RedisConfig) {
				if config.PoolSize != 20 {
					t.Errorf("WithRedisPoolSize() PoolSize = %v, want %v", config.PoolSize, 20)
				}
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
