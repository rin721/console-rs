**输出结果:**

```go
// config.go
type Config struct {
	Server ServerConfig `mapstructure:"server"`
}
```

```go
// server_config.go
type ServerConfig struct {
    Required bool `json:"required" yaml:"required" mapstructure:"required" toml:"required"`
    Host string `json:"host" yaml:"host" mapstructure:"host" toml:"host"`
    Port int64 `json:"port" yaml:"port" mapstructure:"port" toml:"port"`
}

// ValidateName 返回配置名称
func (c *ServerConfig) ValidateName() string {
	return "server"
}

// Validate 为开发者生成一个默认的验证器接口
func (c *ServerConfig) Validate() error {
	return nil
}

// DefaultConfig 为开发者生成一个默认的默认值接口
func (c *ServerConfig) DefaultConfig() *ServerConfig {
	// yaml文件中的数据可做默认值参考
	return &ServerConfig{
		Host: "localhost",
		Port: 8080,
		Required: true,
	}
}

// OverrideConfig 使用环境变量覆盖配置
func (cfg *ServerConfig) OverrideConfig(prefix string) {
	// Host
	if val := os.Getenv(prefix + "SERVER_HOST"); val != "" {
		cfg.Host = val
	}

	// Port
	if val := os.Getenv(prefix + "SERVER_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			cfg.Port = int64(port)
		}
	}

	// Required
	if val := os.Getenv(prefix + "SERVER_REQUIRED"); val != "" {
		if required, err := strconv.ParseBool(val); err == nil {
			cfg.Required = required
		}
	}
}
```

**使用示例:**

```go
// 不使用前缀
cfg := &ServerConfig{}
cfg.OverrideConfig("") // 环境变量: SERVER_HOST, SERVER_PORT, SERVER_REQUIRED

// 使用前缀
cfg.OverrideConfig("APP_") // 环境变量: APP_SERVER_HOST, APP_SERVER_PORT, APP_SERVER_REQUIRED
```
