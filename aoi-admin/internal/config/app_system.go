package config

// SystemConfig 控制系统管理模块的启动期默认数据行为。
type SystemConfig struct {
	SeedDefaultsOnStart *bool `mapstructure:"seed_defaults_on_start" envname:"SYSTEM_SEED_DEFAULTS_ON_START" json:"seed_defaults_on_start" yaml:"seed_defaults_on_start" toml:"seed_defaults_on_start"`
}

func (c *SystemConfig) ValidateName() string {
	return AppSystemName
}

func (c *SystemConfig) ValidateRequired() bool {
	return false
}

func (c *SystemConfig) Validate() error {
	return nil
}

// SeedDefaultsOnStartValue 返回启动期默认数据补齐开关。
func (c SystemConfig) SeedDefaultsOnStartValue() bool {
	if c.SeedDefaultsOnStart == nil {
		return true
	}
	return *c.SeedDefaultsOnStart
}

func copySystemConfig(src SystemConfig) SystemConfig {
	dst := src
	if src.SeedDefaultsOnStart != nil {
		value := *src.SeedDefaultsOnStart
		dst.SeedDefaultsOnStart = &value
	}
	return dst
}
