package initapp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/rei0721/go-scaffold/internal/config"
	systemmodel "github.com/rei0721/go-scaffold/internal/modules/system/model"
	systemservice "github.com/rei0721/go-scaffold/internal/modules/system/service"
)

type runtimeConfigUpdateOperation func(*config.Config) error

func runtimeConfigUpdater(manager config.Manager) func(context.Context, systemservice.UpdateConfigInput) (systemmodel.ConfigSnapshot, error) {
	if manager == nil {
		return nil
	}
	return func(ctx context.Context, input systemservice.UpdateConfigInput) (systemmodel.ConfigSnapshot, error) {
		if err := ctx.Err(); err != nil {
			return systemmodel.ConfigSnapshot{}, err
		}
		current := manager.Get()
		if current == nil {
			return systemmodel.ConfigSnapshot{}, systemservice.ErrConfigUnavailable
		}
		operations, paths, err := buildRuntimeConfigUpdateOperations(current, input.Items)
		if err != nil {
			return systemmodel.ConfigSnapshot{}, fmt.Errorf("%w: %v", systemservice.ErrInvalidInput, err)
		}
		if len(operations) > 0 {
			var options []config.UpdateOption
			if input.Persist {
				options = append(options, config.WithPersistedPaths(paths...))
			}
			if err := manager.Update(func(cfg *config.Config) {
				for _, operation := range operations {
					_ = operation(cfg)
				}
			}, options...); err != nil {
				return systemmodel.ConfigSnapshot{}, fmt.Errorf("%w: %v", systemservice.ErrInvalidInput, err)
			}
		}
		current = manager.Get()
		if current == nil {
			return systemmodel.ConfigSnapshot{}, systemservice.ErrConfigUnavailable
		}
		return SystemConfigSnapshot(current), nil
	}
}

func buildRuntimeConfigUpdateOperations(current *config.Config, items []systemservice.UpdateConfigItem) ([]runtimeConfigUpdateOperation, []string, error) {
	operations := make([]runtimeConfigUpdateOperation, 0, len(items))
	paths := make([]string, 0, len(items))
	for _, item := range items {
		operation, path, err := buildRuntimeConfigUpdateOperation(current, item.Key, item.Value)
		if err != nil {
			return nil, nil, err
		}
		if operation != nil {
			operations = append(operations, operation)
			paths = append(paths, path)
		}
	}
	return operations, paths, nil
}

func buildRuntimeConfigUpdateOperation(current *config.Config, key string, value any) (runtimeConfigUpdateOperation, string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, "", fmt.Errorf("config key is required")
	}
	if current == nil {
		return nil, "", fmt.Errorf("configuration is unavailable")
	}

	target, err := runtimeConfigField(reflect.ValueOf(current).Elem(), key)
	if err != nil {
		return nil, "", err
	}
	switch target.Kind() {
	case reflect.String:
		next, err := parseRuntimeConfigString(value)
		if err != nil {
			return nil, "", fmt.Errorf("%s must be a string", key)
		}
		if isRuntimeConfigSecretKey(key) && strings.TrimSpace(next) == "" {
			return nil, "", nil
		}
		return func(cfg *config.Config) error {
			field, err := runtimeConfigField(reflect.ValueOf(cfg).Elem(), key)
			if err != nil {
				return err
			}
			field.SetString(next)
			return nil
		}, key, nil
	case reflect.Int:
		next, err := parseRuntimeConfigInt(value)
		if err != nil {
			return nil, "", fmt.Errorf("%s must be an integer", key)
		}
		return func(cfg *config.Config) error {
			field, err := runtimeConfigField(reflect.ValueOf(cfg).Elem(), key)
			if err != nil {
				return err
			}
			field.SetInt(int64(next))
			return nil
		}, key, nil
	case reflect.Bool:
		next, err := parseRuntimeConfigBool(value)
		if err != nil {
			return nil, "", fmt.Errorf("%s must be a boolean", key)
		}
		return func(cfg *config.Config) error {
			field, err := runtimeConfigField(reflect.ValueOf(cfg).Elem(), key)
			if err != nil {
				return err
			}
			field.SetBool(next)
			return nil
		}, key, nil
	case reflect.Slice:
		if target.Type().Elem().Kind() != reflect.String {
			return nil, "", fmt.Errorf("%s is not editable", key)
		}
		next, err := parseRuntimeConfigStringSlice(value)
		if err != nil {
			return nil, "", fmt.Errorf("%s must be a string array", key)
		}
		return func(cfg *config.Config) error {
			field, err := runtimeConfigField(reflect.ValueOf(cfg).Elem(), key)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(next).Convert(field.Type()))
			return nil
		}, key, nil
	case reflect.Ptr:
		if target.Type().Elem().Kind() != reflect.Bool {
			return nil, "", fmt.Errorf("%s is not editable", key)
		}
		next, err := parseRuntimeConfigBool(value)
		if err != nil {
			return nil, "", fmt.Errorf("%s must be a boolean", key)
		}
		return func(cfg *config.Config) error {
			field, err := runtimeConfigField(reflect.ValueOf(cfg).Elem(), key)
			if err != nil {
				return err
			}
			ptr := reflect.New(field.Type().Elem())
			ptr.Elem().SetBool(next)
			field.Set(ptr)
			return nil
		}, key, nil
	default:
		return nil, "", fmt.Errorf("%s is not editable", key)
	}
}

func runtimeConfigField(root reflect.Value, key string) (reflect.Value, error) {
	current := root
	for _, segment := range strings.Split(key, ".") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return reflect.Value{}, fmt.Errorf("invalid config key %q", key)
		}
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				current.Set(reflect.New(current.Type().Elem()))
			}
			current = current.Elem()
		}
		if current.Kind() == reflect.Slice || current.Kind() == reflect.Array {
			index, err := strconv.Atoi(segment)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("%s sequence index must be an integer", key)
			}
			if index < 0 || index >= current.Len() {
				return reflect.Value{}, fmt.Errorf("%s sequence index is out of range", key)
			}
			current = current.Index(index)
			continue
		}
		if current.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("%s is not editable", key)
		}
		field, ok := mapstructureField(current, segment)
		if !ok {
			return reflect.Value{}, fmt.Errorf("unknown config key %s", key)
		}
		current = field
	}
	return current, nil
}

func mapstructureField(value reflect.Value, segment string) (reflect.Value, bool) {
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

func parseRuntimeConfigString(value any) (string, error) {
	next, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("not string")
	}
	return next, nil
}

func parseRuntimeConfigBool(value any) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		return strconv.ParseBool(strings.TrimSpace(typed))
	default:
		return false, fmt.Errorf("not bool")
	}
}

func parseRuntimeConfigStringSlice(value any) ([]string, error) {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), nil
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("not string array")
			}
			items = append(items, text)
		}
		return items, nil
	case string:
		if strings.TrimSpace(typed) == "" {
			return []string{}, nil
		}
		separator := ","
		if strings.Contains(typed, "\n") {
			separator = "\n"
		}
		parts := strings.Split(typed, separator)
		items := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				items = append(items, part)
			}
		}
		return items, nil
	default:
		return nil, fmt.Errorf("not string array")
	}
}

func parseRuntimeConfigInt(value any) (int, error) {
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int64:
		return int(typed), nil
	case float64:
		next := int(typed)
		if typed != float64(next) {
			return 0, fmt.Errorf("not integer")
		}
		return next, nil
	case json.Number:
		next, err := typed.Int64()
		return int(next), err
	case string:
		return strconv.Atoi(strings.TrimSpace(typed))
	default:
		return 0, fmt.Errorf("not integer")
	}
}

var runtimeConfigSecretKeys = map[string]struct{}{
	"auth.mfa_secret_key":           {},
	"auth.refresh_token_pepper":     {},
	"auth.signing_key":              {},
	"auth.smtp.password":            {},
	"cache.redis.password":          {},
	"database.mysql.password":       {},
	"database.postgres.password":    {},
	"storage.s3.secretAccessKey":    {},
	"storage.minio.secretAccessKey": {},
}

func isRuntimeConfigSecretKey(key string) bool {
	_, ok := runtimeConfigSecretKeys[key]
	return ok
}
