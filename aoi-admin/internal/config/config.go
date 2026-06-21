package config

import "fmt"

type Configurable interface {
	Validate() error
}

// Config aggregates every runtime configuration section published by the
// configuration manager.
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Cache       CacheConfig       `mapstructure:"cache"`
	Logger      LoggerConfig      `mapstructure:"logger"`
	I18n        I18nConfig        `mapstructure:"i18n"`
	Brand       BrandConfig       `mapstructure:"brand"`
	Executor    ExecutorConfig    `mapstructure:"executor"`
	Storage     StorageConfig     `mapstructure:"storage"`
	CORS        CORSConfig        `mapstructure:"cors"`
	RPC         RPCConfig         `mapstructure:"rpc"`
	Auth        AuthConfig        `mapstructure:"auth"`
	System      SystemConfig      `mapstructure:"system"`
	Migration   MigrationConfig   `mapstructure:"migration"`
	WebUI       WebUIConfig       `mapstructure:"webui"`
	Plugins     PluginsConfig     `mapstructure:"plugins"`
	EnvOverride EnvOverrideConfig `mapstructure:"env_override" json:"env_override" yaml:"env_override" toml:"env_override"`
}

type Validator interface {
	Validate() error
	ValidateName() string
	ValidateRequired() bool
}

func (c *Config) Validate() error {
	validators := []Validator{
		&c.Server,
		&c.Database,
		&c.Cache,
		&c.Logger,
		&c.I18n,
		&c.Brand,
		&c.Executor,
		&c.Storage,
		&c.CORS,
		&c.RPC,
		&c.Auth,
		&c.System,
		&c.Migration,
		&c.WebUI,
		&c.Plugins,
		&c.EnvOverride,
	}
	for _, validator := range validators {
		if validator == nil {
			continue
		}
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("%s config: %w", validator.ValidateName(), err)
		}
	}
	if c.Plugins.Enabled && !c.Auth.Enabled {
		return fmt.Errorf("%s config: auth must be enabled when plugins are enabled", c.Plugins.ValidateName())
	}
	return nil
}
