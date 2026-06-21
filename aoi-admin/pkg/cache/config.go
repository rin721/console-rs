package cache

// 本文件属于 Redis 缓存适配器，说明连接生命周期、键值操作、批处理或热重载边界。

import (
	"fmt"
	"time"
)

type Driver string

const (
	DriverDisabled Driver = "disabled"
	DriverLocal    Driver = "local"
	DriverRedis    Driver = "redis"
	DriverHybrid   Driver = "hybrid"
)

type FactoryConfig struct {
	Driver Driver
	Local  LocalConfig
	Redis  *Config
}

type LocalConfig struct {
	MaxCost     int64
	NumCounters int64
	BufferItems int64
	DefaultTTL  time.Duration
}

func DefaultLocalConfig() LocalConfig {
	return LocalConfig{
		MaxCost:     64 << 20,
		NumCounters: 1_000_000,
		BufferItems: 64,
		DefaultTTL:  0,
	}
}

func (c LocalConfig) normalized() LocalConfig {
	defaults := DefaultLocalConfig()
	if c.MaxCost <= 0 {
		c.MaxCost = defaults.MaxCost
	}
	if c.NumCounters <= 0 {
		c.NumCounters = defaults.NumCounters
	}
	if c.BufferItems <= 0 {
		c.BufferItems = defaults.BufferItems
	}
	return c
}

// Config Redis 缓存配置
// 包含所有 Redis 连接和池管理相关的配置参数
type Config struct {
	// Host Redis 服务器主机地址
	// 可以是 IP 地址或域名
	// 例如: "localhost", "127.0.0.1", "redis.example.com"
	Host string

	// Port Redis 服务器端口
	// 默认 Redis 端口是 6379
	Port int

	Username string

	// Password Redis 访问密码
	// 如果 Redis 没有设置密码,留空即可
	// 生产环境强烈建议设置密码
	Password string

	// DB 数据库索引
	// Redis 支持 0-15 共 16 个数据库
	// 不同的应用可以使用不同的数据库索引来隔离数据
	// 默认使用 0
	DB int

	// PoolSize 连接池大小
	// 决定最多可以同时建立多少个连接
	// 设置建议:
	//   - 小型应用: 10
	//   - 中型应用: 20-50
	//   - 大型应用: 100+
	PoolSize int

	// MinIdleConns 最小空闲连接数
	// 连接池中始终保持的空闲连接数量
	// 保持空闲连接可以减少连接建立的延迟
	// 通常设置为 PoolSize 的 30-50%
	MinIdleConns int

	// MaxRetries 最大重试次数
	// 当命令执行失败时,自动重试的次数
	// 0 表示不重试
	// 建议设置 2-3 次,应对临时性网络问题
	MaxRetries int

	// DialTimeout 连接超时时间
	// 建立 TCP 连接的最大等待时间
	// 如果在此时间内无法建立连接,返回超时错误
	DialTimeout time.Duration

	// ReadTimeout 读取超时时间
	// 从 Redis 读取响应的最大等待时间
	// 包括网络传输时间和 Redis 执行命令的时间
	ReadTimeout time.Duration

	// WriteTimeout 写入超时时间
	// 向 Redis 写入命令的最大等待时间
	// 通常设置得比 ReadTimeout 短一些
	WriteTimeout time.Duration
}

// DefaultConfig 返回默认配置
// 这些默认值适合大多数开发和测试环境
// 生产环境建议根据实际负载调整参数
//
// 返回:
//
//	*Config: 包含合理默认值的配置实例
//
// 使用示例:
//
//	cfg := cache.DefaultConfig()
//	// 可以根据需要调整部分参数
//	cfg.Host = "redis.example.com"
//	cfg.Password = "your-password"
func DefaultConfig() *Config {
	return &Config{
		Host:         DefaultHost,
		Port:         DefaultPort,
		Password:     "", // 空密码,适用于开发环境
		DB:           DefaultDB,
		PoolSize:     DefaultPoolSize,
		MinIdleConns: DefaultMinIdleConns,
		MaxRetries:   DefaultMaxRetries,
		// 将秒转换为 time.Duration
		DialTimeout:  time.Duration(DefaultDialTimeout) * time.Second,
		ReadTimeout:  time.Duration(DefaultReadTimeout) * time.Second,
		WriteTimeout: time.Duration(DefaultWriteTimeout) * time.Second,
	}
}

// Validate 验证配置的有效性
// 检查配置参数是否合理,避免无效配置导致运行时错误
//
// 返回:
//
//	error: 如果配置无效,返回错误描述;否则返回 nil
//
// 验证规则:
//   - Host 不能为空
//   - Port 必须在有效范围内 (1-65535)
//   - DB 必须在有效范围内 (0-15)
//   - PoolSize 必须大于 0
//   - MinIdleConns 不能大于 PoolSize
//   - 超时时间必须大于 0
func (c *Config) Validate() error {
	// 验证 Host
	if c.Host == "" {
		return fmt.Errorf("redis host cannot be empty")
	}

	// 验证 Port
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid redis port: %d, must be between 1 and 65535", c.Port)
	}

	// 验证 DB
	if c.DB < 0 || c.DB > 15 {
		return fmt.Errorf("invalid redis db: %d, must be between 0 and 15", c.DB)
	}

	// 验证 PoolSize
	if c.PoolSize <= 0 {
		return fmt.Errorf("redis pool size must be greater than 0")
	}

	// 验证 MinIdleConns
	if c.MinIdleConns < 0 {
		return fmt.Errorf("redis min idle conns cannot be negative")
	}
	if c.MinIdleConns > c.PoolSize {
		return fmt.Errorf("redis min idle conns (%d) cannot be greater than pool size (%d)",
			c.MinIdleConns, c.PoolSize)
	}

	// 验证超时时间
	if c.DialTimeout <= 0 {
		return fmt.Errorf("redis dial timeout must be greater than 0")
	}
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("redis read timeout must be greater than 0")
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("redis write timeout must be greater than 0")
	}

	return nil
}

func New(config FactoryConfig, logger Logger) (Cache, error) {
	switch config.Driver {
	case "", DriverDisabled:
		return nil, nil
	case DriverLocal:
		return NewLocal(config.Local)
	case DriverRedis:
		return NewRedis(config.Redis, logger)
	case DriverHybrid:
		return NewHybrid(config.Local, config.Redis, logger)
	default:
		return nil, fmt.Errorf("unsupported cache driver %q", config.Driver)
	}
}
