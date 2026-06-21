package initcenter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/internal/app/initapp"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/configloader"
	"github.com/rei0721/go-scaffold/types/constants"
)

const bootstrapStatePath = "data/setup/bootstrap_state.json"
const envManagedPersistenceForceFile = "force_file"

type InitConfigStore struct {
	core       initapp.Core
	configPath string
}

func NewInitConfigStore(core initapp.Core, configPath string) InitConfigStore {
	return InitConfigStore{core: core, configPath: fallback(configPath, constants.AppDefaultConfigPath)}
}

func (s InitConfigStore) Save(ctx context.Context, stepKey string, values map[string]any, persist bool) (ConfigSaveResult, error) {
	values = normalizeValues(values)
	inputFingerprint := inputFingerprintFor(stepKey, values)
	paths := configPathsForStep(stepKey, values)
	if len(paths) == 0 {
		return ConfigSaveResult{StepKey: stepKey, InputSummary: "no configuration changes", InputFingerprint: inputFingerprint}, nil
	}
	if !persist {
		return ConfigSaveResult{StepKey: stepKey, InputSummary: summarizeInput(stepKey, values), InputFingerprint: inputFingerprint}, nil
	}
	envManagedPaths, err := s.envManagedPaths(paths)
	if err != nil {
		return ConfigSaveResult{}, err
	}
	if stepKey == "database.configure" {
		target, err := s.candidate(stepKey, values)
		if err != nil {
			return ConfigSaveResult{}, err
		}
		current := s.core.Config
		if s.core.ConfigManager != nil {
			current = s.core.ConfigManager.Get()
		}
		targetFingerprint := databaseFingerprintFor(target)
		currentFingerprint := databaseFingerprintFor(current)
		if err := s.persistDatabase(values); err != nil {
			return ConfigSaveResult{}, err
		}
		if targetFingerprint == currentFingerprint {
			clearBootstrapState()
			return withEnvManagedPersistence(ConfigSaveResult{
				StepKey:          stepKey,
				InputSummary:     summarizeInput(stepKey, values),
				InputFingerprint: inputFingerprint,
			}, envManagedPaths), nil
		}
		if err := writeBootstrapState(bootstrapState{
			CurrentStep:       "database.configure",
			RestartRequired:   true,
			RestartReason:     "数据库配置已保存。请重启服务，让当前进程加载新的数据库配置后继续初始化。",
			TargetFingerprint: targetFingerprint,
			UpdatedAt:         time.Now().UTC(),
		}); err != nil {
			return ConfigSaveResult{}, err
		}
		return withEnvManagedPersistence(ConfigSaveResult{
			StepKey:          stepKey,
			InputSummary:     summarizeInput(stepKey, values),
			InputFingerprint: inputFingerprint,
			RestartRequired:  true,
			RestartReason:    "数据库配置已保存。请重启服务，让当前进程加载新的数据库配置后继续初始化。",
			NextAction:       "restart",
		}, envManagedPaths), nil
	}
	if s.core.ConfigManager == nil {
		return ConfigSaveResult{}, fmt.Errorf("configuration manager unavailable")
	}
	if err := s.core.ConfigManager.Update(func(cfg *appconfig.Config) {
		for _, path := range paths {
			_ = setConfigPath(cfg, path, values[path])
		}
	}, persistOptions(persist, paths)...); err != nil {
		return ConfigSaveResult{}, err
	}
	return withEnvManagedPersistence(ConfigSaveResult{StepKey: stepKey, InputSummary: summarizeInput(stepKey, values), InputFingerprint: inputFingerprint}, envManagedPaths), nil
}

func (s InitConfigStore) candidate(stepKey string, values map[string]any) (*appconfig.Config, error) {
	cfg := s.core.Config
	if s.core.ConfigManager != nil {
		cfg = s.core.ConfigManager.Get()
	}
	if cfg == nil {
		return nil, fmt.Errorf("configuration is unavailable")
	}
	candidate := cloneConfig(cfg)
	for _, path := range configPathsForStep(stepKey, normalizeValues(values)) {
		if err := setConfigPath(candidate, path, values[path]); err != nil {
			return nil, err
		}
	}
	return candidate, nil
}

func persistOptions(persist bool, paths []string) []appconfig.UpdateOption {
	if !persist {
		return nil
	}
	return []appconfig.UpdateOption{
		appconfig.WithPersistedPaths(paths...),
		appconfig.WithEnvManagedPersistMode(appconfig.EnvManagedPersistForceFile),
	}
}

func withEnvManagedPersistence(result ConfigSaveResult, paths []string) ConfigSaveResult {
	if len(paths) == 0 {
		return result
	}
	result.EnvManagedPathsOverwritten = append([]string(nil), paths...)
	result.EnvManagedPersistence = envManagedPersistenceForceFile
	return result
}

func (s InitConfigStore) envManagedPaths(paths []string) ([]string, error) {
	disabled := map[string]struct{}{}
	cfg := s.core.Config
	if s.core.ConfigManager != nil {
		cfg = s.core.ConfigManager.Get()
	}
	if cfg != nil {
		for _, path := range cfg.EnvOverride.DisabledPaths {
			disabled[strings.TrimSpace(path)] = struct{}{}
		}
	}

	seen := map[string]struct{}{}
	managed := make([]string, 0)
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		isManaged := false
		if strings.TrimSpace(s.configPath) != "" {
			contains, err := configloader.YAMLPathContainsEnvPlaceholder(s.configPath, path)
			if err != nil {
				if !strings.Contains(err.Error(), "config key does not exist in file") {
					return nil, err
				}
			} else if contains {
				isManaged = true
			}
		}
		if _, blocked := disabled[path]; !blocked {
			for _, envName := range appconfig.EnvNamesForPath(path) {
				if value, ok := os.LookupEnv(envName); ok && strings.TrimSpace(value) != "" {
					isManaged = true
					break
				}
			}
		}
		if !isManaged {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		managed = append(managed, path)
	}
	sort.Strings(managed)
	return managed, nil
}

func configPathsForStep(stepKey string, values map[string]any) []string {
	allowed := map[string]map[string]struct{}{
		"database.configure": {
			"database.driver": {}, "database.sqlite.path": {},
			"database.mysql.host": {}, "database.mysql.port": {}, "database.mysql.username": {}, "database.mysql.password": {},
			"database.mysql.database": {}, "database.mysql.charset": {},
			"database.postgres.host": {}, "database.postgres.port": {}, "database.postgres.username": {}, "database.postgres.password": {},
			"database.postgres.database": {}, "database.postgres.sslMode": {},
			"database.pool.maxOpenConns": {}, "database.pool.maxIdleConns": {},
		},
		"cache.configure": {
			"cache.driver": {}, "cache.local.maxCost": {}, "cache.local.numCounters": {}, "cache.local.bufferItems": {}, "cache.local.defaultTtlSeconds": {},
			"cache.redis.addr": {}, "cache.redis.username": {}, "cache.redis.password": {}, "cache.redis.db": {},
			"cache.redis.poolSize": {}, "cache.redis.minIdleConns": {}, "cache.redis.maxRetries": {},
			"cache.redis.dialTimeout": {}, "cache.redis.readTimeout": {}, "cache.redis.writeTimeout": {},
		},
		"storage.configure": {
			"storage.driver": {}, "storage.local.fsType": {}, "storage.local.basePath": {}, "storage.local.publicUrl": {},
			"storage.local.enableWatch": {}, "storage.local.watchBufferSize": {},
			"storage.s3.endpoint": {}, "storage.s3.region": {}, "storage.s3.bucket": {}, "storage.s3.accessKeyId": {},
			"storage.s3.secretAccessKey": {}, "storage.s3.usePathStyle": {}, "storage.s3.publicBaseUrl": {},
			"storage.minio.endpoint": {}, "storage.minio.region": {}, "storage.minio.bucket": {}, "storage.minio.accessKeyId": {},
			"storage.minio.secretAccessKey": {}, "storage.minio.usePathStyle": {}, "storage.minio.publicBaseUrl": {},
		},
		"system.configure": {
			"i18n.defaultLocale": {}, "auth.issuer": {}, "auth.password_policy.min_length": {}, "system.seed_defaults_on_start": {},
		},
		"site.configure": {
			"brand.productName": {}, "brand.versionName": {}, "webui.public_base_url": {},
		},
	}
	set := allowed[stepKey]
	paths := make([]string, 0, len(values))
	for key := range values {
		if _, ok := set[key]; ok {
			if isSetupSecretPath(key) && strings.TrimSpace(stringValue(values[key])) == "" {
				continue
			}
			paths = append(paths, key)
		}
	}
	return paths
}

func isSetupSecretPath(path string) bool {
	switch path {
	case "database.mysql.password", "database.postgres.password", "cache.redis.password", "storage.s3.secretAccessKey", "storage.minio.secretAccessKey":
		return true
	default:
		return false
	}
}

func setConfigPath(cfg *appconfig.Config, path string, value any) error {
	switch path {
	case "database.driver":
		cfg.Database.Driver = stringValue(value)
	case "database.sqlite.path":
		cfg.Database.SQLite.Path = stringValue(value)
	case "database.mysql.host":
		cfg.Database.MySQL.Host = stringValue(value)
	case "database.mysql.port":
		cfg.Database.MySQL.Port = intValue(value)
	case "database.mysql.username":
		cfg.Database.MySQL.Username = stringValue(value)
	case "database.mysql.password":
		cfg.Database.MySQL.Password = stringValue(value)
	case "database.mysql.database":
		cfg.Database.MySQL.Database = stringValue(value)
	case "database.mysql.charset":
		cfg.Database.MySQL.Charset = stringValue(value)
	case "database.postgres.host":
		cfg.Database.Postgres.Host = stringValue(value)
	case "database.postgres.port":
		cfg.Database.Postgres.Port = intValue(value)
	case "database.postgres.username":
		cfg.Database.Postgres.Username = stringValue(value)
	case "database.postgres.password":
		cfg.Database.Postgres.Password = stringValue(value)
	case "database.postgres.database":
		cfg.Database.Postgres.Database = stringValue(value)
	case "database.postgres.sslMode":
		cfg.Database.Postgres.SSLMode = stringValue(value)
	case "database.pool.maxOpenConns":
		cfg.Database.Pool.MaxOpenConns = intValue(value)
	case "database.pool.maxIdleConns":
		cfg.Database.Pool.MaxIdleConns = intValue(value)
	case "cache.driver":
		cfg.Cache.Driver = stringValue(value)
	case "cache.local.maxCost":
		cfg.Cache.Local.MaxCost = int64Value(value)
	case "cache.local.numCounters":
		cfg.Cache.Local.NumCounters = int64Value(value)
	case "cache.local.bufferItems":
		cfg.Cache.Local.BufferItems = int64Value(value)
	case "cache.local.defaultTtlSeconds":
		cfg.Cache.Local.DefaultTTLSeconds = intValue(value)
	case "cache.redis.addr":
		cfg.Cache.Redis.Addr = stringValue(value)
	case "cache.redis.username":
		cfg.Cache.Redis.Username = stringValue(value)
	case "cache.redis.password":
		cfg.Cache.Redis.Password = stringValue(value)
	case "cache.redis.db":
		cfg.Cache.Redis.DB = intValue(value)
	case "cache.redis.poolSize":
		cfg.Cache.Redis.PoolSize = intValue(value)
	case "cache.redis.minIdleConns":
		cfg.Cache.Redis.MinIdleConns = intValue(value)
	case "cache.redis.maxRetries":
		cfg.Cache.Redis.MaxRetries = intValue(value)
	case "cache.redis.dialTimeout":
		cfg.Cache.Redis.DialTimeout = intValue(value)
	case "cache.redis.readTimeout":
		cfg.Cache.Redis.ReadTimeout = intValue(value)
	case "cache.redis.writeTimeout":
		cfg.Cache.Redis.WriteTimeout = intValue(value)
	case "storage.driver":
		cfg.Storage.Driver = stringValue(value)
	case "storage.local.fsType":
		cfg.Storage.Local.FSType = stringValue(value)
	case "storage.local.basePath":
		cfg.Storage.Local.BasePath = stringValue(value)
	case "storage.local.publicUrl":
		cfg.Storage.Local.PublicURL = stringValue(value)
	case "storage.local.enableWatch":
		cfg.Storage.Local.EnableWatch = boolValue(value)
	case "storage.local.watchBufferSize":
		cfg.Storage.Local.WatchBufferSize = intValue(value)
	case "storage.s3.endpoint":
		cfg.Storage.S3.Endpoint = stringValue(value)
	case "storage.s3.region":
		cfg.Storage.S3.Region = stringValue(value)
	case "storage.s3.bucket":
		cfg.Storage.S3.Bucket = stringValue(value)
	case "storage.s3.accessKeyId":
		cfg.Storage.S3.AccessKeyID = stringValue(value)
	case "storage.s3.secretAccessKey":
		cfg.Storage.S3.SecretAccessKey = stringValue(value)
	case "storage.s3.usePathStyle":
		cfg.Storage.S3.UsePathStyle = boolValue(value)
	case "storage.s3.publicBaseUrl":
		cfg.Storage.S3.PublicBaseURL = stringValue(value)
	case "storage.minio.endpoint":
		cfg.Storage.MinIO.Endpoint = stringValue(value)
	case "storage.minio.region":
		cfg.Storage.MinIO.Region = stringValue(value)
	case "storage.minio.bucket":
		cfg.Storage.MinIO.Bucket = stringValue(value)
	case "storage.minio.accessKeyId":
		cfg.Storage.MinIO.AccessKeyID = stringValue(value)
	case "storage.minio.secretAccessKey":
		cfg.Storage.MinIO.SecretAccessKey = stringValue(value)
	case "storage.minio.usePathStyle":
		cfg.Storage.MinIO.UsePathStyle = boolValue(value)
	case "storage.minio.publicBaseUrl":
		cfg.Storage.MinIO.PublicBaseURL = stringValue(value)
	case "i18n.defaultLocale":
		cfg.I18n.DefaultLocale = stringValue(value)
	case "brand.productName":
		cfg.Brand.ProductName = stringValue(value)
	case "brand.versionName":
		cfg.Brand.VersionName = stringValue(value)
	case "webui.public_base_url":
		cfg.WebUI.PublicBaseURL = stringValue(value)
	case "auth.issuer":
		cfg.Auth.Issuer = stringValue(value)
	case "auth.password_policy.min_length":
		cfg.Auth.PasswordPolicy.MinLength = intValue(value)
	case "system.seed_defaults_on_start":
		v := boolValue(value)
		cfg.System.SeedDefaultsOnStart = &v
	default:
		return fmt.Errorf("unsupported setup config path %s", path)
	}
	return nil
}

func (s InitConfigStore) persistDatabase(values map[string]any) error {
	updates := []configloader.YAMLScalarUpdate{}
	paths := configPathsForStep("database.configure", values)
	for _, path := range paths {
		updates = append(updates, yamlScalarUpdate(path, values[path]))
	}
	if update, err := disabledPathsUpdate(s.configPath, paths); err != nil {
		return err
	} else if update != nil {
		updates = append(updates, *update)
	}
	return configloader.UpdateYAMLScalars(s.configPath, updates, configloader.WithEnvPlaceholderOverwrite())
}

func disabledPathsUpdate(configPath string, paths []string) (*configloader.YAMLScalarUpdate, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	current, err := configloader.YAMLStringSlice(configPath, "env_override.disabled_paths")
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	next := make([]string, 0, len(current)+len(paths))
	for _, path := range current {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		next = append(next, path)
	}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		next = append(next, path)
	}
	sort.Strings(next)
	return &configloader.YAMLScalarUpdate{
		Kind:          configloader.YAMLScalarStringSlice,
		Path:          "env_override.disabled_paths",
		Values:        next,
		CreateMissing: true,
	}, nil
}

func yamlScalarUpdate(path string, value any) configloader.YAMLScalarUpdate {
	update := configloader.YAMLScalarUpdate{
		CreateMissing: true,
		Kind:          configloader.YAMLScalarString,
		Path:          path,
		Value:         stringValue(value),
	}
	switch typed := value.(type) {
	case bool:
		update.Kind = configloader.YAMLScalarBool
		update.Value = strconv.FormatBool(typed)
	case int:
		update.Kind = configloader.YAMLScalarInt
		update.Value = strconv.Itoa(typed)
	case int64:
		update.Kind = configloader.YAMLScalarInt
		update.Value = strconv.FormatInt(typed, 10)
	case float64:
		if typed == float64(int(typed)) {
			update.Kind = configloader.YAMLScalarInt
			update.Value = strconv.Itoa(int(typed))
		}
	}
	return update
}

type bootstrapState struct {
	CurrentStep       string    `json:"currentStep"`
	RestartRequired   bool      `json:"restartRequired"`
	RestartReason     string    `json:"restartReason"`
	TargetFingerprint string    `json:"targetFingerprint,omitempty"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func writeBootstrapState(state bootstrapState) error {
	if err := os.MkdirAll(filepath.Dir(bootstrapStatePath), 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(bootstrapStatePath, raw, 0600)
}

func readBootstrapState() (bootstrapState, bool) {
	raw, err := os.ReadFile(bootstrapStatePath)
	if err != nil {
		return bootstrapState{}, false
	}
	var state bootstrapState
	if err := json.Unmarshal(raw, &state); err != nil {
		return bootstrapState{}, false
	}
	return state, state.RestartRequired
}

func clearBootstrapState() {
	_ = os.Remove(bootstrapStatePath)
}

func normalizeValues(values map[string]any) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	return values
}

func inputFingerprintFor(stepKey string, values map[string]any) string {
	normalized := normalizeValues(values)
	keys := make([]string, 0, len(normalized))
	for key := range normalized {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		if isSetupSecretPath(key) && strings.TrimSpace(stringValue(normalized[key])) == "" {
			continue
		}
		ordered = append(ordered, map[string]any{"key": key, "value": normalized[key]})
	}
	raw, _ := json.Marshal(map[string]any{
		"stepKey": stepKey,
		"values":  ordered,
	})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])[:16]
}

func summarizeInput(stepKey string, values map[string]any) string {
	count := 0
	for key := range values {
		if strings.Contains(key, "password") || strings.Contains(key, "key") || strings.Contains(key, "pepper") {
			continue
		}
		count++
	}
	if count == 0 {
		return "sensitive or empty input captured"
	}
	return fmt.Sprintf("%s captured %d field(s)", stepKey, count)
}

func cloneConfig(src *appconfig.Config) *appconfig.Config {
	dst := *src
	dst.I18n.Supported = append([]string(nil), src.I18n.Supported...)
	dst.I18n.Resources = cloneStringMap(src.I18n.Resources)
	dst.CORS.AllowOrigins = append([]string(nil), src.CORS.AllowOrigins...)
	dst.CORS.AllowMethods = append([]string(nil), src.CORS.AllowMethods...)
	dst.CORS.AllowHeaders = append([]string(nil), src.CORS.AllowHeaders...)
	dst.CORS.ExposeHeaders = append([]string(nil), src.CORS.ExposeHeaders...)
	dst.Auth.Audience = append([]string(nil), src.Auth.Audience...)
	dst.Executor.Pools = append([]appconfig.ExecutorPoolConfig(nil), src.Executor.Pools...)
	return &dst
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func configFingerprintFor(cfg *appconfig.Config) string {
	if cfg == nil {
		return ""
	}
	raw := fmt.Sprintf("%v|%v|%v|%v|%v|%v", cfg.Database, cfg.Cache, cfg.Storage, cfg.Auth.Enabled, cfg.Migration, cfg.System)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])[:16]
}

func databaseFingerprintFor(cfg *appconfig.Config) string {
	if cfg == nil {
		return ""
	}
	raw := fmt.Sprintf("%v", cfg.Database)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])[:16]
}

func loadConfigFromPath(path string) (*appconfig.Config, error) {
	manager := appconfig.NewManager()
	if err := manager.Load(fallback(path, constants.AppDefaultConfigPath)); err != nil {
		return nil, err
	}
	return manager.Get(), nil
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(value)
	}
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		next, _ := strconv.Atoi(strings.TrimSpace(typed))
		return next
	default:
		return 0
	}
}

func int64Value(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case string:
		next, _ := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return next
	default:
		return 0
	}
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		next, _ := strconv.ParseBool(strings.TrimSpace(typed))
		return next
	default:
		return false
	}
}
