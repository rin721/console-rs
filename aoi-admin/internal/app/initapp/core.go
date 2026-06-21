package initapp

// 本文件属于应用初始化装配层，负责把配置、基础设施、业务模块或传输层拼接为可运行的分层对象。

import (
	"fmt"

	"github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/i18n"
	"github.com/rei0721/go-scaffold/pkg/logger"
	"github.com/rei0721/go-scaffold/pkg/utils"
)

// NewCore 创建后续所有装配层共享的核心服务集合。
//
// 配置必须最先加载，因为日志、i18n 和 ID 生成器都依赖配置提供运行参数。
func NewCore(configPath string) (Core, error) {
	configManager, cfg, err := LoadConfig(configPath)
	if err != nil {
		return Core{}, err
	}

	log, err := NewLogger(cfg)
	if err != nil {
		return Core{}, err
	}

	i18nApp, i18nUtils, err := NewI18n(cfg)
	if err != nil {
		return Core{}, err
	}

	return Core{
		Config:        cfg,
		ConfigManager: configManager,
		Logger:        log,
		I18n:          i18nApp,
		I18nUtils:     i18nUtils,
		IDGenerator:   NewIDGenerator(),
	}, nil
}

// LoadConfig 创建配置管理器并加载指定配置文件。
//
// 返回的 Config 是管理器当前快照；后续热更新仍以 ConfigManager 作为权威来源。
func LoadConfig(configPath string) (config.Manager, *config.Config, error) {
	configManager := config.NewManager()
	if err := configManager.Load(configPath); err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}
	return configManager, configManager.Get(), nil
}

// NewLogger 根据应用配置创建日志器。
func NewLogger(cfg *config.Config) (logger.Logger, error) {
	log, err := logger.New(LoggerConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	log.Info("logger initialized successfully")
	return log, nil
}

// NewI18n 创建 i18n 应用和业务侧便捷工具。
//
// I18nUtils 固定使用配置中的默认语言，避免业务层重复读取配置并产生语言回退分歧。
func NewI18n(cfg *config.Config) (i18n.I18n, *utils.I18nUtils, error) {
	i18nCfg := &i18n.Config{
		DefaultLocale:  cfg.I18n.DefaultLocale,
		FallbackLocale: cfg.I18n.FallbackLocale,
		Supported:      cfg.I18n.Supported,
		Resources:      cfg.I18n.Resources,
	}

	i18nApp, err := i18n.New(i18nCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create i18n: %w", err)
	}
	return i18nApp, utils.NewI18nUtils(i18nApp, cfg.I18n.DefaultLocale), nil
}

// NewIDGenerator 创建脚手架默认的 Snowflake ID 生成器。
func NewIDGenerator() utils.IDGenerator {
	return utils.DefaultSnowflake()
}

// DebugConfig 输出启动期关键配置摘要。
//
// 日志中只记录定位运行态所需的配置字段，不输出密码等敏感值。
func DebugConfig(core Core, configPath string) {
	core.Logger.Debug("configuration loaded successfully",
		"config_file", configPath,
		"env_support", "enabled")

	core.Logger.Debug("server configuration",
		"port", core.Config.Server.Port,
		"mode", core.Config.Server.Mode)

	core.Logger.Debug("database configuration",
		"driver", core.Config.Database.Driver,
		"database", DatabaseSummary(core.Config))

	core.Logger.Debug("cache configuration",
		"driver", core.Config.Cache.Driver,
		"redis_addr", core.Config.Cache.Redis.Addr,
		"redis_db", core.Config.Cache.Redis.DB)
}
