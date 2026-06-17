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
	network      string
	addr         string
	username     string
	password     string
	db           int
	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
	poolSize     int
	tlsConfig    *tls.Config
}

func (c *RedisConfig) validate() error {
	if c == nil {
		return ErrInvalidRedisConfig
	}
	if c.addr == "" {
		return ErrRedisEmptyAddress
	}
	if c.network != "tcp" && c.network != "unix" {
		return fmt.Errorf("%w: %q", ErrRedisUnsupportedNetwork, c.network)
	}
	if c.tlsConfig == nil && strings.HasPrefix(c.addr, "rediss://") {
		return ErrRedisTLSRequired
	}
	if _, _, err := net.SplitHostPort(c.addr); err != nil && c.network == "tcp" {
		return fmt.Errorf("%w: %w", ErrRedisInvalidAddress, err)
	}
	if c.db < 0 {
		return fmt.Errorf("%w: %d", ErrRedisInvalidDB, c.db)
	}
	if c.poolSize < 0 {
		return fmt.Errorf("%w: %d", ErrRedisInvalidPoolSize, c.poolSize)
	}
	if c.dialTimeout < 0 {
		return fmt.Errorf("%w: dial timeout %s", ErrRedisInvalidTimeout, c.dialTimeout)
	}
	if c.readTimeout < 0 {
		return fmt.Errorf("%w: read timeout %s", ErrRedisInvalidTimeout, c.readTimeout)
	}
	if c.writeTimeout < 0 {
		return fmt.Errorf("%w: write timeout %s", ErrRedisInvalidTimeout, c.writeTimeout)
	}
	return nil
}

// NewRedisConfig creates a new RedisConfig with the given options applied.
func NewRedisConfig(opts ...RedisOption) (*RedisConfig, error) {
	config := DefaultRedisConfig()
	for _, opt := range opts {
		opt.applyRedisOption(config)
	}
	if err := config.validate(); err != nil {
		return nil, err
	}
	return config, nil
}

// DefaultRedisConfig returns a RedisConfig initialized with default values.
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		network:      "tcp",
		addr:         "localhost:6379",
		username:     "",
		password:     "",
		db:           0,
		dialTimeout:  5 * time.Second,
		readTimeout:  3 * time.Second,
		writeTimeout: 3 * time.Second,
		poolSize:     runtime.NumCPU() * 10,
		tlsConfig:    nil,
	}
}

// Network returns the Redis network.
func (c *RedisConfig) Network() string {
	if c == nil {
		return ""
	}
	return c.network
}

// Addr returns the Redis server address.
func (c *RedisConfig) Addr() string {
	if c == nil {
		return ""
	}
	return c.addr
}

// Username returns the Redis username.
func (c *RedisConfig) Username() string {
	if c == nil {
		return ""
	}
	return c.username
}

// Password returns the Redis password.
func (c *RedisConfig) Password() string {
	if c == nil {
		return ""
	}
	return c.password
}

// DB returns the Redis database number.
func (c *RedisConfig) DB() int {
	if c == nil {
		return 0
	}
	return c.db
}

// DialTimeout returns the Redis dial timeout.
func (c *RedisConfig) DialTimeout() time.Duration {
	if c == nil {
		return 0
	}
	return c.dialTimeout
}

// ReadTimeout returns the Redis read timeout.
func (c *RedisConfig) ReadTimeout() time.Duration {
	if c == nil {
		return 0
	}
	return c.readTimeout
}

// WriteTimeout returns the Redis write timeout.
func (c *RedisConfig) WriteTimeout() time.Duration {
	if c == nil {
		return 0
	}
	return c.writeTimeout
}

// PoolSize returns the Redis connection pool size.
func (c *RedisConfig) PoolSize() int {
	if c == nil {
		return 0
	}
	return c.poolSize
}

// TLSConfig returns a copy of the Redis TLS configuration.
func (c *RedisConfig) TLSConfig() *tls.Config {
	if c == nil {
		return nil
	}
	return cloneTLSConfig(c.tlsConfig)
}

func asynqRedisOpt(c *RedisConfig) asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Network:      c.network,
		Addr:         c.addr,
		Username:     c.username,
		Password:     c.password,
		DB:           c.db,
		DialTimeout:  c.dialTimeout,
		ReadTimeout:  c.readTimeout,
		WriteTimeout: c.writeTimeout,
		PoolSize:     c.poolSize,
		TLSConfig:    cloneTLSConfig(c.tlsConfig),
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

// WithRedisNetwork sets the Redis network.
func WithRedisNetwork(network string) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.network = network
	})
}

// WithRedisAddress sets the Redis server address.
func WithRedisAddress(addr string) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.addr = addr
	})
}

// WithRedisUsername sets the username for Redis authentication.
func WithRedisUsername(username string) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.username = username
	})
}

// WithRedisPassword sets the password for Redis authentication.
func WithRedisPassword(password string) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.password = password
	})
}

// WithRedisDB sets the Redis database number.
func WithRedisDB(db int) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.db = db
	})
}

// WithRedisTLSConfig sets the TLS configuration for the Redis connection.
func WithRedisTLSConfig(tlsConfig *tls.Config) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.tlsConfig = cloneTLSConfig(tlsConfig)
	})
}

// WithRedisDialTimeout sets the timeout for connecting to Redis.
func WithRedisDialTimeout(timeout time.Duration) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.dialTimeout = timeout
	})
}

// WithRedisReadTimeout sets the timeout for reading from Redis.
func WithRedisReadTimeout(timeout time.Duration) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.readTimeout = timeout
	})
}

// WithRedisWriteTimeout sets the timeout for writing to Redis.
func WithRedisWriteTimeout(timeout time.Duration) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.writeTimeout = timeout
	})
}

// WithRedisPoolSize sets the size of the connection pool for Redis.
func WithRedisPoolSize(size int) RedisOption {
	return redisOption(func(c *RedisConfig) {
		c.poolSize = size
	})
}

func cloneTLSConfig(config *tls.Config) *tls.Config {
	if config == nil {
		return nil
	}
	return config.Clone()
}
