package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/rei0721/go-scaffold/pkg/configloader"
)

const envOverrideDisabledPathsConfigPath = "env_override.disabled_paths"

// persistConfigUpdate 将 Manager.Update 中指定的配置路径写回 YAML 文件。
//
// paths 使用 mapstructure 路径；函数只持久化可表示为 YAML 标量或字符串数组的字段。遇到环境变量管理的
// 字段时会按 options.envManagedPersistMode 决定拒绝、强制覆盖占位符或仅更新运行时配置。
func (m *manager) persistConfigUpdate(newCfg *Config, paths []string, options updateOptions) error {
	if strings.TrimSpace(m.configPath) == "" {
		return fmt.Errorf("configuration file path is not available")
	}
	if options.envManagedPersistMode < EnvManagedPersistReject || options.envManagedPersistMode > EnvManagedPersistRuntimeEnvOnly {
		return fmt.Errorf("unsupported env managed persist mode %d", options.envManagedPersistMode)
	}

	updates := make([]configloader.YAMLScalarUpdate, 0, len(paths))
	for _, path := range paths {
		if options.envManagedPersistMode != EnvManagedPersistForceFile {
			if envName, ok := activeConfigPathEnvName(path); ok {
				return fmt.Errorf("%s is managed by environment variable %s", path, envName)
			}
		}
		if options.envManagedPersistMode == EnvManagedPersistRuntimeEnvOnly {
			if managed, err := m.configPathHasEnvPlaceholder(path); err != nil {
				return err
			} else if managed {
				return missingRuntimeEnvError(path)
			}
		}

		value, err := configValueByMapstructurePath(reflect.ValueOf(newCfg).Elem(), path)
		if err != nil {
			return err
		}
		update, err := yamlScalarUpdateFromConfigValue(path, value)
		if err != nil {
			return err
		}
		updates = append(updates, update)
	}

	var yamlOptions []configloader.YAMLUpdateOption
	if options.envManagedPersistMode == EnvManagedPersistForceFile {
		yamlOptions = append(yamlOptions, configloader.WithEnvPlaceholderOverwrite())
	}
	return configloader.UpdateYAMLScalars(m.configPath, updates, yamlOptions...)
}

// applyRuntimeEnvOnlyPersistPaths 处理“只更新运行时环境值”的持久化策略。
//
// 对仍由环境变量或占位符管理的字段，函数从真实环境变量读值写入 newCfg，并从文件持久化路径中移除；
// 返回的 bool 表示 env_override.disabled_paths 是否需要同步更新。
func (m *manager) applyRuntimeEnvOnlyPersistPaths(newCfg *Config, paths []string) (map[string]struct{}, bool, error) {
	runtimeOnlyPaths := make(map[string]struct{}, len(paths))
	disabledPaths := disabledConfigPathSet(newCfg.EnvOverride.DisabledPaths)
	root := reflect.ValueOf(newCfg).Elem()
	for _, path := range paths {
		_, alreadyDisabled := disabledPaths[path]
		managed, err := m.configPathIsEnvManaged(path)
		if err != nil {
			return nil, false, err
		}
		if !managed && !alreadyDisabled {
			continue
		}

		envName, raw, ok := activeConfigPathEnv(path)
		if !ok {
			return nil, false, missingRuntimeEnvError(path)
		}
		field, err := configValueByMapstructurePath(root, path)
		if err != nil {
			return nil, false, err
		}
		if !setValueFromEnv(field, raw) {
			return nil, false, fmt.Errorf("%s cannot be set from environment variable %s", path, envName)
		}
		runtimeOnlyPaths[path] = struct{}{}
	}
	metadataChanged := removeDisabledEnvOverridePaths(&newCfg.EnvOverride, runtimeOnlyPaths)
	return runtimeOnlyPaths, metadataChanged, nil
}

// configPathIsEnvManaged 判断配置路径是否正由环境变量或 YAML 占位符接管。
func (m *manager) configPathIsEnvManaged(path string) (bool, error) {
	if _, _, ok := activeConfigPathEnv(path); ok {
		return true, nil
	}
	return m.configPathHasEnvPlaceholder(path)
}

// configPathHasEnvPlaceholder 检查配置文件中的路径值是否包含 ${VAR:default} 语法。
func (m *manager) configPathHasEnvPlaceholder(path string) (bool, error) {
	if strings.TrimSpace(m.configPath) == "" {
		return false, fmt.Errorf("configuration file path is not available")
	}
	return configloader.YAMLPathContainsEnvPlaceholder(m.configPath, path)
}

// missingRuntimeEnvError 构造运行时环境缺失时的可操作错误。
//
// 错误消息会列出候选环境变量名，方便 CLI 引导用户补齐 env 或选择强制写文件。
func missingRuntimeEnvError(path string) error {
	candidates := EnvNamesForPath(path)
	if len(candidates) == 0 {
		return fmt.Errorf("%s is managed by environment placeholder but has no environment variable mapping", path)
	}
	return fmt.Errorf("%s is managed by environment placeholder; set one of %s or choose force file persistence", path, strings.Join(candidates, ", "))
}

// removeConfigPaths 从持久化路径列表中移除已由运行时环境接管的路径。
func removeConfigPaths(paths []string, remove map[string]struct{}) []string {
	if len(remove) == 0 {
		return paths
	}
	kept := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, ok := remove[path]; ok {
			continue
		}
		kept = append(kept, path)
	}
	return kept
}

// addDisabledEnvOverridePaths 将路径加入 env_override.disabled_paths。
//
// 强制写文件时需要记录这些路径不再接受 env override，否则下一次加载仍会被环境变量覆盖。
func addDisabledEnvOverridePaths(cfg *Config, paths []string) bool {
	if cfg == nil {
		return false
	}
	changed := false
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(cfg.EnvOverride.DisabledPaths)+len(paths))
	for _, path := range normalizeConfigPaths(cfg.EnvOverride.DisabledPaths) {
		seen[path] = struct{}{}
		normalized = append(normalized, path)
	}
	for _, path := range normalizeConfigPaths(paths) {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		normalized = append(normalized, path)
		changed = true
	}
	cfg.EnvOverride.DisabledPaths = normalized
	return changed
}

// removeDisabledEnvOverridePaths 从 env override 禁用列表中移除已恢复由运行时环境管理的路径。
func removeDisabledEnvOverridePaths(cfg *EnvOverrideConfig, paths map[string]struct{}) bool {
	if cfg == nil || len(paths) == 0 {
		return false
	}
	next := make([]string, 0, len(cfg.DisabledPaths))
	changed := false
	for _, path := range normalizeConfigPaths(cfg.DisabledPaths) {
		if _, ok := paths[path]; ok {
			changed = true
			continue
		}
		next = append(next, path)
	}
	cfg.DisabledPaths = next
	return changed
}

// configValueByMapstructurePath 按 mapstructure 路径在配置结构中定位可写字段。
//
// 支持结构体字段和切片/数组索引，例如 executor.pools.0.size；返回值用于运行时赋值和 YAML 标量持久化。
func configValueByMapstructurePath(root reflect.Value, path string) (reflect.Value, error) {
	current := root
	for _, segment := range strings.Split(path, ".") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return reflect.Value{}, fmt.Errorf("invalid config key %q", path)
		}
		if current.Kind() == reflect.Pointer {
			if current.IsNil() {
				return reflect.Value{}, fmt.Errorf("%s is nil", path)
			}
			current = current.Elem()
		}
		if current.Kind() == reflect.Slice || current.Kind() == reflect.Array {
			index, err := strconv.Atoi(segment)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("%s sequence index must be an integer", path)
			}
			if index < 0 || index >= current.Len() {
				return reflect.Value{}, fmt.Errorf("%s sequence index is out of range", path)
			}
			current = current.Index(index)
			continue
		}
		if current.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("%s is not editable", path)
		}
		field, ok := mapstructureValueField(current, segment)
		if !ok {
			return reflect.Value{}, fmt.Errorf("unknown config key %s", path)
		}
		current = field
	}
	return current, nil
}

// mapstructureValueField 根据 mapstructure tag 在结构值中查找字段。
func mapstructureValueField(value reflect.Value, segment string) (reflect.Value, bool) {
	valueType := value.Type()
	for index := 0; index < value.NumField(); index++ {
		fieldType := valueType.Field(index)
		tag := strings.Split(fieldType.Tag.Get("mapstructure"), ",")[0]
		if tag == segment {
			return value.Field(index), true
		}
	}
	return reflect.Value{}, false
}

// yamlScalarUpdateFromConfigValue 将反射字段转换为 YAML 标量更新描述。
//
// 只允许字符串、布尔、整数和字符串切片，避免把复杂对象局部写回后破坏配置文件结构。
func yamlScalarUpdateFromConfigValue(path string, value reflect.Value) (configloader.YAMLScalarUpdate, error) {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return configloader.YAMLScalarUpdate{}, fmt.Errorf("%s is nil", path)
		}
		value = value.Elem()
	}

	update := configloader.YAMLScalarUpdate{Path: path}
	switch value.Kind() {
	case reflect.String:
		update.Kind = configloader.YAMLScalarString
		update.Value = value.String()
	case reflect.Bool:
		update.Kind = configloader.YAMLScalarBool
		update.Value = strconv.FormatBool(value.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		update.Kind = configloader.YAMLScalarInt
		update.Value = strconv.FormatInt(value.Int(), 10)
	case reflect.Slice:
		if value.Type().Elem().Kind() != reflect.String {
			return configloader.YAMLScalarUpdate{}, fmt.Errorf("%s is not a persistable scalar value", path)
		}
		update.Kind = configloader.YAMLScalarStringSlice
		update.CreateMissing = path == envOverrideDisabledPathsConfigPath
		update.Values = make([]string, 0, value.Len())
		for index := 0; index < value.Len(); index++ {
			update.Values = append(update.Values, value.Index(index).String())
		}
	default:
		return configloader.YAMLScalarUpdate{}, fmt.Errorf("%s is not a persistable scalar value", path)
	}
	return update, nil
}

// activeConfigPathEnvName 返回当前实际接管该配置路径的环境变量名。
func activeConfigPathEnvName(path string) (string, bool) {
	envName, _, ok := activeConfigPathEnv(path)
	return envName, ok
}

// activeConfigPathEnv 查找配置路径对应且当前有值的环境变量。
//
// 候选名按带应用前缀、无前缀的顺序检查，保持部署时的优先级与加载逻辑一致。
func activeConfigPathEnv(path string) (string, string, bool) {
	envName, ok := configPathEnvName(path)
	if !ok {
		return "", "", false
	}
	for _, candidate := range envNameCandidates(envName) {
		if value, ok := os.LookupEnv(candidate); ok && value != "" {
			return candidate, value, true
		}
	}
	return "", "", false
}

// EnvNamesForPath 返回配置路径可使用的环境变量名，按实际覆盖优先级排序。
func EnvNamesForPath(path string) []string {
	envName, ok := configPathEnvName(path)
	if !ok {
		return nil
	}
	return envNameCandidates(envName)
}

// configPathEnvName 从配置结构 tag 中解析指定路径绑定的 envname。
func configPathEnvName(path string) (string, bool) {
	field, ok := configStructFieldByMapstructurePath(reflect.TypeOf(Config{}), path)
	if !ok {
		return "", false
	}
	envName := strings.TrimSpace(field.Tag.Get(envNameTag))
	return envName, envName != "" && envName != "-"
}

// configStructFieldByMapstructurePath 按 mapstructure 路径定位配置结构字段元数据。
//
// 与 configValueByMapstructurePath 不同，这里只读取类型信息，用于解析 envname tag。
func configStructFieldByMapstructurePath(root reflect.Type, path string) (reflect.StructField, bool) {
	current := root
	var field reflect.StructField
	for _, segment := range strings.Split(path, ".") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return reflect.StructField{}, false
		}
		if current.Kind() == reflect.Pointer {
			current = current.Elem()
		}
		if current.Kind() == reflect.Slice || current.Kind() == reflect.Array {
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 {
				return reflect.StructField{}, false
			}
			current = current.Elem()
			continue
		}
		if current.Kind() != reflect.Struct {
			return reflect.StructField{}, false
		}
		next, ok := mapstructureTypeField(current, segment)
		if !ok {
			return reflect.StructField{}, false
		}
		field = next
		current = next.Type
	}
	return field, true
}

// mapstructureTypeField 根据 mapstructure tag 在结构类型中查找字段。
func mapstructureTypeField(valueType reflect.Type, segment string) (reflect.StructField, bool) {
	for index := 0; index < valueType.NumField(); index++ {
		field := valueType.Field(index)
		tag := strings.Split(field.Tag.Get("mapstructure"), ",")[0]
		if tag == segment {
			return field, true
		}
	}
	return reflect.StructField{}, false
}
