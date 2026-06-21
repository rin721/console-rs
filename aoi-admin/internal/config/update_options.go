package config

import "strings"

// EnvManagedPersistMode 定义持久化更新遇到环境变量管理字段时的处理策略。
type EnvManagedPersistMode int

const (
	// EnvManagedPersistReject 保持默认安全行为，拒绝覆盖环境变量管理字段。
	EnvManagedPersistReject EnvManagedPersistMode = iota
	// EnvManagedPersistForceFile 允许把更新值强制写入配置文件。
	EnvManagedPersistForceFile
	// EnvManagedPersistRuntimeEnvOnly 只读取真实环境变量更新运行时配置，不写入配置文件。
	EnvManagedPersistRuntimeEnvOnly
)

// UpdateOption 配置 Manager.Update 的可选行为。
type UpdateOption func(*updateOptions)

type updateOptions struct {
	persistPaths          []string
	envManagedPersistMode EnvManagedPersistMode
}

// WithPersistedPaths 要求 Manager.Update 在验证通过后把指定配置路径写回配置文件。
func WithPersistedPaths(paths ...string) UpdateOption {
	return func(options *updateOptions) {
		options.persistPaths = append(options.persistPaths, paths...)
	}
}

// WithEnvManagedPersistMode 设置持久化更新遇到环境变量管理字段时的处理策略。
func WithEnvManagedPersistMode(mode EnvManagedPersistMode) UpdateOption {
	return func(options *updateOptions) {
		options.envManagedPersistMode = mode
	}
}

func collectUpdateOptions(options []UpdateOption) updateOptions {
	var collected updateOptions
	for _, option := range options {
		if option != nil {
			option(&collected)
		}
	}
	collected.persistPaths = normalizeConfigPaths(collected.persistPaths)
	return collected
}

func normalizeConfigPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
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
