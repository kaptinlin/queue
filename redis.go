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

// Validate checks if the RedisConfig fields are correctly set.
func (c *RedisConfig) Validate() error {
	if c.Addr == "" {
		return ErrRedisEmptyAddress
	}
	if c.Network != "tcp" && c.Network != "unix" {
		return fmt.Errorf("%w: %q", ErrRedisUnsupportedNetwork, c.Network)
	}
	if c.TLSConfig == nil && strings.HasPrefix(c.Addr, "rediss://") {
		return ErrRedisTLSRequired
	}
	if _, _, err := net.SplitHostPort(c.Addr); err != nil && c.Network == "tcp" {
		return fmt.Errorf("%w: %w", ErrRedisInvalidAddress, err)
	}
	if c.DB < 0 {
		return fmt.Errorf("%w: %d", ErrRedisInvalidDB, c.DB)
	}
	if c.PoolSize < 0 {
		return fmt.Errorf("%w: %d", ErrRedisInvalidPoolSize, c.PoolSize)
	}
	if c.DialTimeout < 0 {
		return fmt.Errorf("%w: dial timeout %s", ErrRedisInvalidTimeout, c.DialTimeout)
	}
	if c.ReadTimeout < 0 {
		return fmt.Errorf("%w: read timeout %s", ErrRedisInvalidTimeout, c.ReadTimeout)
	}
	if c.WriteTimeout < 0 {
		return fmt.Errorf("%w: write timeout %s", ErrRedisInvalidTimeout, c.WriteTimeout)
	}
	return nil
}

// NewRedisConfig creates a new RedisConfig with the given options applied.
func NewRedisConfig(opts ...RedisOption) *RedisConfig {
	config := DefaultRedisConfig()
	for _, opt := range opts {
		opt.applyRedisOption(config)
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
		TLSConfig:    cloneTLSConfig(c.TLSConfig),
	}
}

// RedisOption configures RedisConfig.
type RedisOption interface {
	applyRedisOption(*RedisConfig)
}

type redisOption func(*RedisConfig)

func (f redisOption) applyRedisOption(config *RedisConfig) {
	f(config)
}

// WithRedisAddress sets the Redis server address.
func WithRedisAddress(addr string) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.Addr = addr
	})
}

// WithRedisUsername sets the username for Redis authentication.
func WithRedisUsername(username string) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.Username = username
	})
}

// WithRedisPassword sets the password for Redis authentication.
func WithRedisPassword(password string) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.Password = password
	})
}

// WithRedisDB sets the Redis database number.
func WithRedisDB(db int) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.DB = db
	})
}

// WithRedisTLSConfig sets the TLS configuration for the Redis connection.
func WithRedisTLSConfig(tlsConfig *tls.Config) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.TLSConfig = cloneTLSConfig(tlsConfig)
	})
}

// WithRedisDialTimeout sets the timeout for connecting to Redis.
func WithRedisDialTimeout(timeout time.Duration) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.DialTimeout = timeout
	})
}

// WithRedisReadTimeout sets the timeout for reading from Redis.
func WithRedisReadTimeout(timeout time.Duration) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.ReadTimeout = timeout
	})
}

// WithRedisWriteTimeout sets the timeout for writing to Redis.
func WithRedisWriteTimeout(timeout time.Duration) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.WriteTimeout = timeout
	})
}

// WithRedisPoolSize sets the size of the connection pool for Redis.
func WithRedisPoolSize(size int) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.PoolSize = size
	})
}

func cloneTLSConfig(config *tls.Config) *tls.Config {
	if config == nil {
		return nil
	}
	return config.Clone()
}
