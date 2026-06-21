package config

// EnvOverrideConfig 记录需要按配置文件值优先处理的配置路径。
type EnvOverrideConfig struct {
	DisabledPaths []string `mapstructure:"disabled_paths" json:"disabled_paths" yaml:"disabled_paths" toml:"disabled_paths"`
}

func (c *EnvOverrideConfig) ValidateName() string {
	return "env_override"
}

func (c *EnvOverrideConfig) ValidateRequired() bool {
	return false
}

func (c *EnvOverrideConfig) Validate() error {
	c.DisabledPaths = normalizeConfigPaths(c.DisabledPaths)
	return nil
}
