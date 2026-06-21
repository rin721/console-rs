package config

import (
	"errors"
	"fmt"
	"strings"
)

const (
	CacheDriverDisabled = "disabled"
	CacheDriverLocal    = "local"
	CacheDriverRedis    = "redis"
	CacheDriverHybrid   = "hybrid"
)

type CacheConfig struct {
	Driver string           `mapstructure:"driver" envname:"CACHE_DRIVER"`
	Local  LocalCacheConfig `mapstructure:"local"`
	Redis  RedisCacheConfig `mapstructure:"redis"`
}

type LocalCacheConfig struct {
	MaxCost           int64 `mapstructure:"maxCost" envname:"CACHE_LOCAL_MAX_COST"`
	NumCounters       int64 `mapstructure:"numCounters" envname:"CACHE_LOCAL_NUM_COUNTERS"`
	BufferItems       int64 `mapstructure:"bufferItems" envname:"CACHE_LOCAL_BUFFER_ITEMS"`
	DefaultTTLSeconds int   `mapstructure:"defaultTtlSeconds" envname:"CACHE_LOCAL_DEFAULT_TTL_SECONDS"`
}

type RedisCacheConfig struct {
	Addr         string `mapstructure:"addr" envname:"CACHE_REDIS_ADDR"`
	Username     string `mapstructure:"username" envname:"CACHE_REDIS_USERNAME"`
	Password     string `mapstructure:"password" envname:"CACHE_REDIS_PASSWORD"`
	DB           int    `mapstructure:"db" envname:"CACHE_REDIS_DB"`
	PoolSize     int    `mapstructure:"poolSize" envname:"CACHE_REDIS_POOL_SIZE"`
	MinIdleConns int    `mapstructure:"minIdleConns" envname:"CACHE_REDIS_MIN_IDLE_CONNS"`
	MaxRetries   int    `mapstructure:"maxRetries" envname:"CACHE_REDIS_MAX_RETRIES"`
	DialTimeout  int    `mapstructure:"dialTimeout" envname:"CACHE_REDIS_DIAL_TIMEOUT"`
	ReadTimeout  int    `mapstructure:"readTimeout" envname:"CACHE_REDIS_READ_TIMEOUT"`
	WriteTimeout int    `mapstructure:"writeTimeout" envname:"CACHE_REDIS_WRITE_TIMEOUT"`
}

func (c *CacheConfig) ValidateName() string {
	return AppCacheName
}

func (c *CacheConfig) ValidateRequired() bool {
	return false
}

func (c *CacheConfig) Validate() error {
	c.Driver = strings.ToLower(strings.TrimSpace(c.Driver))
	switch c.Driver {
	case CacheDriverDisabled, CacheDriverLocal:
	case CacheDriverRedis, CacheDriverHybrid:
		if err := c.Redis.Validate(); err != nil {
			return err
		}
	case "":
		return errors.New("driver is required")
	default:
		return fmt.Errorf("driver must be one of %s, %s, %s or %s", CacheDriverDisabled, CacheDriverLocal, CacheDriverRedis, CacheDriverHybrid)
	}
	if c.Driver == CacheDriverLocal || c.Driver == CacheDriverHybrid {
		if c.Local.MaxCost < 0 {
			return errors.New("local.maxCost must be non-negative")
		}
		if c.Local.NumCounters < 0 {
			return errors.New("local.numCounters must be non-negative")
		}
		if c.Local.BufferItems < 0 {
			return errors.New("local.bufferItems must be non-negative")
		}
		if c.Local.DefaultTTLSeconds < 0 {
			return errors.New("local.defaultTtlSeconds must be non-negative")
		}
	}
	return nil
}

func (c RedisCacheConfig) Validate() error {
	if strings.TrimSpace(c.Addr) == "" {
		return errors.New("redis.addr is required")
	}
	if c.DB < 0 || c.DB > 15 {
		return errors.New("redis.db must be between 0 and 15")
	}
	if c.PoolSize < 0 {
		return errors.New("redis.poolSize must be non-negative")
	}
	if c.MinIdleConns < 0 {
		return errors.New("redis.minIdleConns must be non-negative")
	}
	if c.MaxRetries < 0 {
		return errors.New("redis.maxRetries must be non-negative")
	}
	if c.DialTimeout < 0 || c.ReadTimeout < 0 || c.WriteTimeout < 0 {
		return errors.New("redis timeouts must be non-negative")
	}
	return nil
}

func overrideCacheConfig(cfg *CacheConfig) {
	overrideConfigFromEnv(cfg)
}
