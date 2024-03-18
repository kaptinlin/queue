package queue

import (
	"crypto/tls"
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/hibiken/asynq"
)

// RedisConfig holds the configuration for the Redis connection.
type RedisConfig struct {
	Network      string
	Addr         string
	Username     string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	TLSConfig    *tls.Config
}

// validate checks if the RedisConfig's fields are correctly set.
func (c *RedisConfig) validate() error {
	if c.Addr == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if c.Network != "tcp" && c.Network != "unix" {
		return fmt.Errorf("unsupported network type %q", c.Network)
	}
	if _, _, err := net.SplitHostPort(c.Addr); err != nil && c.Network == "tcp" {
		return fmt.Errorf("invalid address format: %w", err)
	}
	if c.TLSConfig == nil && strings.HasPrefix(c.Addr, "rediss://") {
		return fmt.Errorf("TLS config is required for secure Redis connections")
	}
	return nil
}

// NewRedisConfig creates a new RedisConfig with the given options applied.
func NewRedisConfig(opts ...RedisOption) *RedisConfig {
	config := DefaultRedisConfig()
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// DefaultRedisConfig returns a RedisConfig initialized with default values.
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Network:      "tcp",
		Addr:         "localhost:6379",
		Username:     "",
		Password:     "",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     runtime.NumCPU() * 10,
		TLSConfig:    nil,
	}
}

// ToAsynqRedisOpt converts RedisConfig to asynq.RedisClientOpt.
// This utility function bridges Redis configuration to asynq's expected format.
func (c *RedisConfig) ToAsynqRedisOpt() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Network:      c.Network,
		Addr:         c.Addr,
		Username:     c.Username,
		Password:     c.Password,
		DB:           c.DB,
		DialTimeout:  c.DialTimeout,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,
		PoolSize:     c.PoolSize,
		TLSConfig:    c.TLSConfig,
	}
}

// RedisOption defines a function signature for configuring RedisConfig.
type RedisOption func(*RedisConfig)

// WithRedisAddress sets the Redis server address.
func WithRedisAddress(addr string) RedisOption {
	return func(c *RedisConfig) {
		c.Addr = addr
	}
}

// WithRedisPassword sets the password for Redis authentication.
func WithRedisPassword(password string) RedisOption {
	return func(c *RedisConfig) {
		c.Password = password
	}
}

// WithRedisDB sets the Redis database number.
func WithRedisDB(db int) RedisOption {
	return func(c *RedisConfig) {
		c.DB = db
	}
}

// WithRedisTLSConfig sets the TLS configuration for the Redis connection.
func WithRedisTLSConfig(tlsConfig *tls.Config) RedisOption {
	return func(c *RedisConfig) {
		c.TLSConfig = tlsConfig
	}
}

// WithRedisDialTimeout sets the timeout for connecting to Redis.
func WithRedisDialTimeout(timeout time.Duration) RedisOption {
	return func(c *RedisConfig) {
		c.DialTimeout = timeout
	}
}

// WithRedisReadTimeout sets the timeout for reading from Redis.
func WithRedisReadTimeout(timeout time.Duration) RedisOption {
	return func(c *RedisConfig) {
		c.ReadTimeout = timeout
	}
}

// WithRedisWriteTimeout sets the timeout for writing to Redis.
func WithRedisWriteTimeout(timeout time.Duration) RedisOption {
	return func(c *RedisConfig) {
		c.WriteTimeout = timeout
	}
}

// WithRedisPoolSize sets the size of the connection pool for Redis.
func WithRedisPoolSize(size int) RedisOption {
	return func(c *RedisConfig) {
		c.PoolSize = size
	}
}
