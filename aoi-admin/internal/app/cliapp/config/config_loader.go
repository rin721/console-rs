package config

import appconfig "github.com/rei0721/go-scaffold/internal/config"

// LoadConfig 使用标准配置管理器加载并校验配置。
func LoadConfig(configPath string) (*appconfig.Config, error) {
	manager := appconfig.NewManager()
	if err := manager.Load(configPath); err != nil {
		return nil, err
	}
	return manager.Get(), nil
}
