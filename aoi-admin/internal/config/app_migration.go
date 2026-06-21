package config

import "github.com/rei0721/go-scaffold/pkg/migrator"

type MigrationConfig struct {
	AutoApply bool   `mapstructure:"auto_apply" envname:"MIGRATION_AUTO_APPLY" json:"auto_apply" yaml:"auto_apply" toml:"auto_apply"`
	Dir       string `mapstructure:"dir" envname:"MIGRATION_DIR" json:"dir" yaml:"dir" toml:"dir"`
}

func (c *MigrationConfig) ValidateName() string {
	return AppMigrationName
}

func (c *MigrationConfig) ValidateRequired() bool {
	return false
}

func (c *MigrationConfig) Validate() error {
	return nil
}

func (c *MigrationConfig) ApplyDefaults() {
	if c.Dir == "" {
		c.Dir = migrator.DefaultDir
	}
}
