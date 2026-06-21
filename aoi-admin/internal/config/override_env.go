package config

// 本文件属于配置子系统，处理配置加载、环境变量覆盖、运行时快照或跨分区校验。

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const envNameTag = "envname"

// overrideConfigFromEnv 以反射方式遍历配置结构，把匹配 envname 的环境变量写回配置快照。
func overrideConfigFromEnv(target any) {
	overrideConfigFromEnvExcept(target, nil)
}

// overrideConfigFromEnvExcept 以反射方式遍历配置结构，并跳过显式禁用 env 覆盖的路径。
func overrideConfigFromEnvExcept(target any, disabledPaths []string) {
	value := reflect.ValueOf(target)
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() {
		return
	}
	overrideStructFields(value.Elem(), "", disabledConfigPathSet(disabledPaths))
}

// overrideStructFields 递归处理结构体字段，并在叶子字段上应用环境变量覆盖。
func overrideStructFields(value reflect.Value, basePath string, disabledPaths map[string]struct{}) {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return
	}

	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		structField := valueType.Field(i)
		if structField.PkgPath != "" {
			continue
		}
		fieldPath := mapstructureFieldPath(basePath, structField)

		if envName := strings.TrimSpace(structField.Tag.Get(envNameTag)); envName != "" && envName != "-" {
			if _, disabled := disabledPaths[fieldPath]; !disabled {
				if raw, ok := lookupTaggedEnv(envName); ok {
					setValueFromEnv(field, raw)
				}
			}
		}

		if shouldRecurseEnvField(field) {
			overrideStructFields(field, fieldPath, disabledPaths)
		}
	}
}

// disabledConfigPathSet 将禁用 env 覆盖的配置路径归一化为集合，便于递归遍历时快速判断。
func disabledConfigPathSet(paths []string) map[string]struct{} {
	set := make(map[string]struct{}, len(paths))
	for _, path := range normalizeConfigPaths(paths) {
		set[path] = struct{}{}
	}
	return set
}

// mapstructureFieldPath 生成字段在配置树中的稳定路径。
//
// 优先使用 mapstructure tag，保证运行时持久化、诊断和 env 覆盖使用同一套路径语义。
func mapstructureFieldPath(basePath string, field reflect.StructField) string {
	segment := strings.Split(field.Tag.Get("mapstructure"), ",")[0]
	segment = strings.TrimSpace(segment)
	if segment == "" || segment == "-" {
		segment = field.Name
	}
	if basePath == "" {
		return segment
	}
	return basePath + "." + segment
}

// shouldRecurseEnvField 判断字段是否应继续递归，跳过时间类型和不适合展开的标量字段。
func shouldRecurseEnvField(field reflect.Value) bool {
	switch field.Kind() {
	case reflect.Struct:
		return true
	case reflect.Pointer:
		return !field.IsNil() && field.Elem().Kind() == reflect.Struct
	default:
		return false
	}
}

// lookupTaggedEnv 按动态前缀优先、无前缀兜底的顺序查找 envname 对应环境变量。
func lookupTaggedEnv(envName string) (string, bool) {
	for _, candidate := range envNameCandidates(envName) {
		if value, ok := os.LookupEnv(candidate); ok && value != "" {
			return value, true
		}
	}
	return "", false
}

// envNameCandidates 构造一个字段可接受的环境变量候选名，保持动态 AppPrefix 与历史无前缀变量兼容。
func envNameCandidates(envName string) []string {
	envName = strings.TrimSpace(envName)
	if envName == "" {
		return nil
	}

	prefixed := EnvPrefixJoin(envName)
	if prefixed == envName {
		return []string{envName}
	}
	return []string{prefixed, envName}
}

// setValueFromEnv 根据反射字段类型分派环境变量写入逻辑，并忽略无法设置的字段。
func setValueFromEnv(field reflect.Value, raw string) bool {
	if !field.CanSet() {
		return false
	}

	if field.Kind() == reflect.Pointer {
		return setPointerValueFromEnv(field, raw)
	}

	return setConcreteValueFromEnv(field, raw)
}

// setPointerValueFromEnv 为指针字段分配目标值后写入环境变量解析结果。
func setPointerValueFromEnv(field reflect.Value, raw string) bool {
	elem := reflect.New(field.Type().Elem()).Elem()
	if !setConcreteValueFromEnv(elem, raw) {
		return false
	}
	field.Set(elem.Addr())
	return true
}

// setConcreteValueFromEnv 将字符串环境变量转换为具体标量类型并写入字段。
func setConcreteValueFromEnv(field reflect.Value, raw string) bool {
	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
		return true
	case reflect.Bool:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return false
		}
		field.SetBool(value)
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(raw, 10, field.Type().Bits())
		if err != nil {
			return false
		}
		field.SetInt(value)
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(raw, 10, field.Type().Bits())
		if err != nil {
			return false
		}
		field.SetUint(value)
		return true
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(raw, field.Type().Bits())
		if err != nil {
			return false
		}
		field.SetFloat(value)
		return true
	case reflect.Slice:
		return setSliceValueFromEnv(field, raw)
	default:
		return false
	}
}

// setSliceValueFromEnv 将分隔符表达的环境变量列表转换为切片字段。
func setSliceValueFromEnv(field reflect.Value, raw string) bool {
	elemKind := field.Type().Elem().Kind()
	switch elemKind {
	case reflect.String:
		values := splitEnvList(raw)
		if len(values) == 0 {
			return false
		}
		field.Set(reflect.ValueOf(values).Convert(field.Type()))
		return true
	default:
		value := reflect.New(field.Type()).Interface()
		if err := json.Unmarshal([]byte(raw), value); err != nil {
			return false
		}
		field.Set(reflect.ValueOf(value).Elem())
		return true
	}
}

// splitEnvList 拆分环境变量列表并清理空白项，避免配置切片携带无意义元素。
func splitEnvList(raw string) []string {
	parts := strings.Split(raw, DefaultSeparator)
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
