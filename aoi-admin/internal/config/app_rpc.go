package config

import "fmt"

const (
	DefaultRPCPort         = 10099
	DefaultRPCReadTimeout  = 10
	DefaultRPCWriteTimeout = 10
	DefaultRPCIdleTimeout  = 30
)

// RPCConfig 控制独立 JSON-RPC 入口。
type RPCConfig struct {
	Enabled      bool   `mapstructure:"enabled" envname:"RPC_ENABLED" json:"enabled" yaml:"enabled" toml:"enabled"`
	Host         string `mapstructure:"host" envname:"RPC_HOST" json:"host" yaml:"host" toml:"host"`
	Port         int    `mapstructure:"port" envname:"RPC_PORT" json:"port" yaml:"port" toml:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout" envname:"RPC_READ_TIMEOUT" json:"read_timeout" yaml:"read_timeout" toml:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout" envname:"RPC_WRITE_TIMEOUT" json:"write_timeout" yaml:"write_timeout" toml:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout" envname:"RPC_IDLE_TIMEOUT" json:"idle_timeout" yaml:"idle_timeout" toml:"idle_timeout"`
}

// ValidateName 返回 RPC 配置分区名称。
func (c *RPCConfig) ValidateName() string {
	return AppRPCName
}

// ValidateRequired 声明 RPC 配置为可选分区。
func (c *RPCConfig) ValidateRequired() bool {
	return false
}

// Validate 校验 RPC 配置；关闭时允许零值通过。
func (c *RPCConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	c.ApplyDefaults()
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("read_timeout must be positive")
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("write_timeout must be positive")
	}
	if c.IdleTimeout < 0 {
		return fmt.Errorf("idle_timeout must be non-negative")
	}
	return nil
}

// ApplyDefaults 补齐 RPC 默认配置。
func (c *RPCConfig) ApplyDefaults() {
	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	if c.Port == 0 {
		c.Port = DefaultRPCPort
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = DefaultRPCReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = DefaultRPCWriteTimeout
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = DefaultRPCIdleTimeout
	}
}
