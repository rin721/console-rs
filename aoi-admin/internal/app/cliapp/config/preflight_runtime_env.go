package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cli"
	"github.com/rei0721/go-scaffold/pkg/configloader"
)

func promptConfigStringValue(ctx *cli.Context, path string, fallback string) (string, error) {
	value, err := cli.InputKey(ctx.Context, ctx.UI, "privacy."+path+".value", path, strings.TrimSpace(fallback))
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", path)
	}
	return value, nil
}

func promptConfigIntValue(ctx *cli.Context, path string, fallback int) (int, error) {
	value, err := cli.InputKey(ctx.Context, ctx.UI, "privacy."+path+".value", path, strconv.Itoa(fallback))
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", path)
	}
	if err := validatePreflightRuntimeEnvValue(path, strconv.Itoa(parsed)); err != nil {
		return 0, err
	}
	return parsed, nil
}

func defaultInt(value int, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

// applyConfigForceFileScalarUpdates 强制把修复值写入配置文件。
//
// 同时把相关路径加入 env_override.disabled_paths，并允许覆盖 YAML 中的 env 占位符，确保下次加载文件值不会再被旧 env 覆盖。
func applyConfigForceFileScalarUpdates(configPath string, updates []configloader.YAMLScalarUpdate, disabledPaths []string) error {
	if len(updates) == 0 {
		return nil
	}
	disabled, err := configloader.YAMLStringSlice(configPath, "env_override.disabled_paths")
	if err != nil {
		return err
	}
	disabled = append(disabled, disabledPaths...)
	nextUpdates := append([]configloader.YAMLScalarUpdate(nil), updates...)
	nextUpdates = append(nextUpdates, configloader.YAMLScalarUpdate{
		Kind:          configloader.YAMLScalarStringSlice,
		Path:          "env_override.disabled_paths",
		Values:        normalizeConfigPathList(disabled),
		CreateMissing: true,
	})
	return configloader.UpdateYAMLScalars(configPath, nextUpdates, configloader.WithEnvPlaceholderOverwrite())
}

// applyRuntimeEnvOnlyConfigPathsDirect 切换指定路径回运行时环境变量管理。
//
// 函数会先确认所有路径都有可用且合法的环境变量，再从 disabled_paths 移除它们，避免配置进入半修复状态。
func applyRuntimeEnvOnlyConfigPathsDirect(configPath string, paths []string, validate func(string, string) error) error {
	normalized := normalizeConfigPathList(paths)
	if len(normalized) == 0 {
		return nil
	}
	for _, path := range normalized {
		if _, _, err := requireConfigRuntimeEnv(path, validate); err != nil {
			return err
		}
	}
	disabledPaths, err := configloader.YAMLStringSlice(configPath, "env_override.disabled_paths")
	if err != nil {
		return err
	}
	remove := make(map[string]struct{}, len(normalized))
	for _, path := range normalized {
		remove[path] = struct{}{}
	}
	kept := make([]string, 0, len(disabledPaths))
	changed := false
	for _, path := range disabledPaths {
		if _, ok := remove[path]; ok {
			changed = true
			continue
		}
		kept = append(kept, path)
	}
	if !changed {
		return nil
	}
	return configloader.UpdateYAMLScalars(configPath, []configloader.YAMLScalarUpdate{
		{
			Kind:          configloader.YAMLScalarStringSlice,
			Path:          "env_override.disabled_paths",
			Values:        kept,
			CreateMissing: true,
		},
	})
}

// requireConfigRuntimeEnv 要求配置路径至少有一个当前可用的环境变量值。
func requireConfigRuntimeEnv(path string, validate func(string, string) error) (string, string, error) {
	for _, envName := range appconfig.EnvNamesForPath(path) {
		if value, ok := os.LookupEnv(envName); ok && strings.TrimSpace(value) != "" {
			value = strings.TrimSpace(value)
			if validate != nil {
				if err := validate(path, value); err != nil {
					return envName, value, err
				}
			}
			return envName, value, nil
		}
	}
	names := appconfig.EnvNamesForPath(path)
	if len(names) == 0 {
		return "", "", fmt.Errorf("%s has no environment variable mapping", path)
	}
	return "", "", fmt.Errorf("%s requires one of %s or choose to write values to the config file", path, strings.Join(names, ", "))
}

// validatePreflightRuntimeEnvValue 校验 preflight 修复依赖的环境变量值。
func validatePreflightRuntimeEnvValue(path string, value string) error {
	value = strings.TrimSpace(value)
	switch path {
	case "database.mysql.port", "database.postgres.port", "auth.smtp.port":
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("environment value for %s must be an integer", path)
		}
		if port <= 0 || port > 65535 {
			return fmt.Errorf("environment value for %s must be between 1 and 65535", path)
		}
	case "auth.signing_key", "auth.refresh_token_pepper", "auth.mfa_secret_key":
		return validatePrivacyRuntimeEnvValue(path, value)
	default:
		if value == "" {
			return fmt.Errorf("environment value for %s is required", path)
		}
	}
	return nil
}

// normalizeConfigPathList 清理、去重并保留配置路径的原始顺序。
func normalizeConfigPathList(paths []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		normalized = append(normalized, path)
	}
	return normalized
}

func exampleConfigWriteError(configPath string) error {
	return fmt.Errorf("example config %s is read-only for generated or repaired values; copy it to a real config file or set the listed environment variables", configPath)
}
