package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/configloader"
)

// ApplyPrivacyUpdates 使用配置管理器持久化隐私配置。
//
// updates 中只有受支持且非空的隐私路径会被写入；实际文件更新仍通过 Manager.Update 完成校验和持久化。
func ApplyPrivacyUpdates(configPath string, updates map[string]string, options ...appconfig.UpdateOption) error {
	paths, normalized := normalizePrivacyUpdates(updates)
	if len(paths) == 0 {
		return nil
	}
	manager := appconfig.NewManager()
	if err := manager.Load(configPath); err != nil {
		return err
	}
	updateOptions := []appconfig.UpdateOption{appconfig.WithPersistedPaths(paths...)}
	updateOptions = append(updateOptions, options...)
	err := manager.Update(func(cfg *appconfig.Config) {
		for path, value := range normalized {
			applyPrivacyValue(cfg, path, value)
		}
	}, updateOptions...)
	if err != nil {
		return err
	}
	return nil
}

// applyPrivacyForceFileUpdates 强制把隐私值写入配置文件并禁用对应 env override。
//
// 该路径用于用户明确选择写文件时覆盖原有环境变量占位符，确保后续启动读取到刚生成或输入的值。
func applyPrivacyForceFileUpdates(configPath string, updates map[string]string) error {
	paths, normalized := normalizePrivacyUpdates(updates)
	if len(paths) == 0 {
		return nil
	}

	yamlUpdates := make([]configloader.YAMLScalarUpdate, 0, len(paths)+1)
	for _, path := range paths {
		yamlUpdates = append(yamlUpdates, configloader.YAMLScalarUpdate{
			Kind:  configloader.YAMLScalarString,
			Path:  path,
			Value: normalized[path],
		})
	}
	disabledPaths, err := configloader.YAMLStringSlice(configPath, "env_override.disabled_paths")
	if err != nil {
		return err
	}
	disabledPaths = append(disabledPaths, paths...)
	yamlUpdates = append(yamlUpdates, configloader.YAMLScalarUpdate{
		Kind:          configloader.YAMLScalarStringSlice,
		Path:          "env_override.disabled_paths",
		Values:        disabledPaths,
		CreateMissing: true,
	})
	return configloader.UpdateYAMLScalars(configPath, yamlUpdates, configloader.WithEnvPlaceholderOverwrite())
}

// normalizePrivacyUpdates 过滤不支持或空值的隐私更新，并按路径排序保证写回顺序稳定。
func normalizePrivacyUpdates(updates map[string]string) ([]string, map[string]string) {
	paths := make([]string, 0, len(updates))
	normalized := make(map[string]string, len(updates))
	for path, value := range updates {
		path = strings.TrimSpace(path)
		value = strings.TrimSpace(value)
		if value == "" || !supportedPrivacyPath(path) {
			continue
		}
		normalized[path] = value
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, normalized
}

// applyPrivacyValue 将受支持的隐私配置路径写入运行时配置副本。
func applyPrivacyValue(cfg *appconfig.Config, path string, value string) bool {
	switch path {
	case "auth.signing_key":
		cfg.Auth.SigningKey = value
	case "auth.refresh_token_pepper":
		cfg.Auth.RefreshTokenPepper = value
	case "auth.mfa_secret_key":
		cfg.Auth.MFASecretKey = value
	case "database.mysql.password":
		cfg.Database.MySQL.Password = value
	case "database.postgres.password":
		cfg.Database.Postgres.Password = value
	case "cache.redis.password":
		cfg.Cache.Redis.Password = value
	case "auth.smtp.password":
		cfg.Auth.SMTP.Password = value
	default:
		return false
	}
	return true
}

// supportedPrivacyPath 限定启动向导允许生成或写入的敏感配置范围。
func supportedPrivacyPath(path string) bool {
	switch path {
	case "auth.signing_key", "auth.refresh_token_pepper", "auth.mfa_secret_key", "database.mysql.password", "database.postgres.password", "cache.redis.password", "auth.smtp.password":
		return true
	default:
		return false
	}
}

// isExampleConfig 判断当前配置文件是否为示例文件。
func IsExampleConfig(path string) bool {
	return strings.Contains(strings.ToLower(filepath.Base(path)), ".example.")
}

// privacyPathIsEnvManaged 判断隐私路径当前是否由环境变量或 YAML 占位符管理。
//
// disabled_paths 中的路径表示文件值被强制使用，也需要向用户暴露为“已有管理状态”，以便选择覆盖或恢复 env。
func privacyPathIsEnvManaged(configPath string, path string) (bool, error) {
	cfg, err := LoadConfig(configPath)
	if err == nil {
		for _, disabledPath := range cfg.EnvOverride.DisabledPaths {
			if disabledPath == path {
				return true, nil
			}
		}
	}
	for _, envName := range appconfig.EnvNamesForPath(path) {
		if value, ok := os.LookupEnv(envName); ok && value != "" {
			return true, nil
		}
	}
	return configloader.YAMLPathContainsEnvPlaceholder(configPath, path)
}
