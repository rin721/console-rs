package config

import (
	"fmt"
	"os"
	"strings"

	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/configloader"
)

// ApplyPrivacyRuntimeEnvOnly 校验并应用真实环境变量中的隐私配置，不改写配置文件。
//
// 通过 Manager.Update 复用配置校验与 env-managed 持久化策略，确保运行时快照与文件元数据一致。
func ApplyPrivacyRuntimeEnvOnly(configPath string, paths []string) error {
	normalized := normalizePrivacyPaths(paths)
	if len(normalized) == 0 {
		return nil
	}

	manager := appconfig.NewManager()
	if err := manager.Load(configPath); err != nil {
		return err
	}
	return manager.Update(func(*appconfig.Config) {}, appconfig.WithPersistedPaths(normalized...), appconfig.WithEnvManagedPersistMode(appconfig.EnvManagedPersistRuntimeEnvOnly))
}

// applyPrivacyRuntimeEnvOnlyDirect 只更新配置文件中的 env_override 元数据。
//
// 该 helper 用于启动前修复流程：先确认真实环境变量可用，再从 disabled_paths 移除路径，让下一次加载重新接受 env。
func applyPrivacyRuntimeEnvOnlyDirect(configPath string, paths []string) error {
	normalized := normalizePrivacyPaths(paths)
	if len(normalized) == 0 {
		return nil
	}
	for _, path := range normalized {
		if _, _, err := requirePrivacyRuntimeEnv(path); err != nil {
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

// requirePrivacyRuntimeEnv 要求隐私路径存在可用且满足强度要求的环境变量值。
func requirePrivacyRuntimeEnv(path string) (string, string, error) {
	for _, envName := range appconfig.EnvNamesForPath(path) {
		if value, ok := os.LookupEnv(envName); ok && strings.TrimSpace(value) != "" {
			value = strings.TrimSpace(value)
			if err := validatePrivacyRuntimeEnvValue(path, value); err != nil {
				return envName, value, err
			}
			return envName, value, nil
		}
	}
	names := appconfig.EnvNamesForPath(path)
	if len(names) == 0 {
		return "", "", fmt.Errorf("%s is managed by environment placeholder but has no environment variable mapping", path)
	}
	return "", "", fmt.Errorf("%s is managed by environment placeholder; set one of %s or choose to write generated values to the config file", path, strings.Join(names, ", "))
}

// validatePrivacyRuntimeEnvValue 校验隐私相关环境变量的最低可用性要求。
func validatePrivacyRuntimeEnvValue(path string, value string) error {
	switch path {
	case "auth.signing_key":
		if len(value) < 32 {
			return fmt.Errorf("environment value for %s must be at least 32 bytes", path)
		}
	case "auth.refresh_token_pepper":
		if value == "" {
			return fmt.Errorf("environment value for %s is required", path)
		}
	case "auth.mfa_secret_key":
		if len(value) < 32 {
			return fmt.Errorf("environment value for %s must be at least 32 bytes", path)
		}
	}
	return nil
}
