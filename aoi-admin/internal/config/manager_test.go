package config

// 本测试文件固定配置复制、环境变量覆盖和热加载行为，防止注释补全和后续重构改变外部可观察行为。

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestCopyConfigCoversAllFieldsAndDeepCopiesSlices 固定配置复制、环境变量覆盖和热加载行为，确保后续注释补全或结构调整不改变该场景。
func TestCopyConfigCoversAllFieldsAndDeepCopiesSlices(t *testing.T) {
	t.Parallel()

	src := testCompleteConfig()
	m := &manager{}

	got := m.copyConfig(src)
	if !reflect.DeepEqual(got, src) {
		t.Fatalf("copyConfig() did not preserve all fields\nwant: %#v\ngot:  %#v", src, got)
	}

	got.I18n.Supported[0] = "ja-JP"
	got.Executor.Pools[0].Name = "changed"
	*got.System.SeedDefaultsOnStart = false
	got.CORS.AllowOrigins[0] = "https://changed.example.com"
	got.CORS.AllowMethods[0] = "PATCH"
	got.CORS.AllowHeaders[0] = "X-Changed"
	got.CORS.ExposeHeaders[0] = "X-Changed-Expose"
	*got.WebUI.Enabled = false
	got.EnvOverride.DisabledPaths[0] = "auth.refresh_token_pepper"

	if src.I18n.Supported[0] == got.I18n.Supported[0] {
		t.Fatal("copyConfig() shares I18n.Supported slice with source")
	}
	if src.Executor.Pools[0].Name == got.Executor.Pools[0].Name {
		t.Fatal("copyConfig() shares Executor.Pools slice with source")
	}
	if *src.System.SeedDefaultsOnStart == *got.System.SeedDefaultsOnStart {
		t.Fatal("copyConfig() shares System.SeedDefaultsOnStart pointer with source")
	}
	if src.CORS.AllowOrigins[0] == got.CORS.AllowOrigins[0] {
		t.Fatal("copyConfig() shares CORS.AllowOrigins slice with source")
	}
	if src.CORS.AllowMethods[0] == got.CORS.AllowMethods[0] {
		t.Fatal("copyConfig() shares CORS.AllowMethods slice with source")
	}
	if src.CORS.AllowHeaders[0] == got.CORS.AllowHeaders[0] {
		t.Fatal("copyConfig() shares CORS.AllowHeaders slice with source")
	}
	if src.CORS.ExposeHeaders[0] == got.CORS.ExposeHeaders[0] {
		t.Fatal("copyConfig() shares CORS.ExposeHeaders slice with source")
	}
	if *src.WebUI.Enabled == *got.WebUI.Enabled {
		t.Fatal("copyConfig() shares WebUI.Enabled pointer with source")
	}
	if src.EnvOverride.DisabledPaths[0] == got.EnvOverride.DisabledPaths[0] {
		t.Fatal("copyConfig() shares EnvOverride.DisabledPaths slice with source")
	}
}

// TestUpdatePreservesUntouchedFields 固定配置复制、环境变量覆盖和热加载行为，确保后续注释补全或结构调整不改变该场景。
func TestUpdatePreservesUntouchedFields(t *testing.T) {
	t.Parallel()

	src := testCompleteConfig()
	m := &manager{}
	m.config.Store(src)

	if err := m.Update(func(cfg *Config) {
		cfg.Server.Port = 9090
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got := m.Get()
	want := testCompleteConfig()
	want.Server.Port = 9090
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Update() did not preserve untouched fields\nwant: %#v\ngot:  %#v", want, got)
	}

	if src.Server.Port != 8080 {
		t.Fatalf("Update() mutated source config, got source port %d", src.Server.Port)
	}
}

func TestUpdateWithoutPersistDoesNotWriteConfigFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath)
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config before update: %v", err)
	}

	m := NewManager()
	if err := m.Load(configPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := m.Update(func(cfg *Config) {
		cfg.Server.Port = 19091
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config after update: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("expected runtime update to leave file unchanged\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if got := m.Get().Server.Port; got != 19091 {
		t.Fatalf("expected runtime config port 19091, got %d", got)
	}
}

func TestUpdateWithPersistWritesScalarFieldsToConfigFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath)

	m := NewManager()
	if err := m.Load(configPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := m.Update(func(cfg *Config) {
		cfg.Server.Port = 19091
		cfg.Database.MySQL.Password = "persistent-secret"
		cfg.CORS.Enabled = true
		cfg.CORS.AllowOrigins = []string{"https://admin.example.com", "https://app.example.com"}
		cfg.Executor.Pools[0].Size = 42
	}, WithPersistedPaths("server.port", "database.mysql.password", "cors.enabled", "cors.allow_origins", "executor.pools.0.size")); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read persisted config: %v", err)
	}
	text := string(content)
	for _, want := range []string{`port: 19091`, `password: "persistent-secret"`, `enabled: true`, `- "https://admin.example.com"`, `- "https://app.example.com"`, `size: 42`} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected persisted config to contain %q, got:\n%s", want, text)
		}
	}

	reloaded := NewManager()
	if err := reloaded.Load(configPath); err != nil {
		t.Fatalf("reload persisted config: %v", err)
	}
	if got := reloaded.Get().Server.Port; got != 19091 {
		t.Fatalf("reloaded server port = %d, want 19091", got)
	}
	if got := reloaded.Get().Database.MySQL.Password; got != "persistent-secret" {
		t.Fatalf("reloaded database password = %q, want persistent-secret", got)
	}
	if !reloaded.Get().CORS.Enabled {
		t.Fatal("expected reloaded CORS enabled")
	}
	if !reflect.DeepEqual(reloaded.Get().CORS.AllowOrigins, []string{"https://admin.example.com", "https://app.example.com"}) {
		t.Fatalf("reloaded CORS allow origins = %#v", reloaded.Get().CORS.AllowOrigins)
	}
	if got := reloaded.Get().Executor.Pools[0].Size; got != 42 {
		t.Fatalf("reloaded executor pool size = %d, want 42", got)
	}
}

func TestUpdateWithPersistRejectsInvalidConfigWithoutWritingFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath)
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config before update: %v", err)
	}

	m := NewManager()
	if err := m.Load(configPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := m.Update(func(cfg *Config) {
		cfg.Server.Port = 0
	}, WithPersistedPaths("server.port")); err == nil {
		t.Fatal("expected invalid persisted update to fail")
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config after update: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("expected invalid update to leave file unchanged\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if got := m.Get().Server.Port; got != 8080 {
		t.Fatalf("expected current config to remain unchanged, got port %d", got)
	}
}

func TestUpdateWithPersistRejectsMissingFileNode(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath)

	m := NewManager()
	if err := m.Load(configPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := m.Update(func(cfg *Config) {
		cfg.Server.Port = 19091
	}, WithPersistedPaths("server.missing")); err == nil {
		t.Fatal("expected missing persisted path to fail")
	}
	if got := m.Get().Server.Port; got != 8080 {
		t.Fatalf("expected current config to remain unchanged, got port %d", got)
	}
}

func TestUpdateWithPersistRejectsEnvManagedFileNode(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath)
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content = []byte(strings.Replace(string(content), "port: 8080", "port: ${SERVER_PORT:8080}", 1))
	if err := os.WriteFile(configPath, content, 0600); err != nil {
		t.Fatalf("write env managed config: %v", err)
	}

	m := NewManager()
	if err := m.Load(configPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := m.Update(func(cfg *Config) {
		cfg.Server.Port = 19091
	}, WithPersistedPaths("server.port")); err == nil {
		t.Fatal("expected environment placeholder to block persisted update")
	}
	if got := m.Get().Server.Port; got != 8080 {
		t.Fatalf("expected current config to remain unchanged, got port %d", got)
	}
}

func TestUpdateWithPersistRejectsActiveEnvOverride(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath)
	setTaggedEnv(t, ServerConfig{}, "Port", "19090")

	m := NewManager()
	if err := m.Load(configPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := m.Get().Server.Port; got != 19090 {
		t.Fatalf("expected env override port 19090, got %d", got)
	}
	if err := m.Update(func(cfg *Config) {
		cfg.Server.Port = 19091
	}, WithPersistedPaths("server.port")); err == nil {
		t.Fatal("expected active environment override to block persisted update")
	}
	if got := m.Get().Server.Port; got != 19090 {
		t.Fatalf("expected current config to remain env value, got port %d", got)
	}
}

func TestManagerLoadSkipsDisabledEnvOverridePath(t *testing.T) {
	configPath := copyConfigExampleWithEnvOverrideForTest(t, []string{"auth.notification_driver"})
	envNames := EnvNamesForPath("auth.notification_driver")
	unsetEnvForTest(t, envNames...)
	if len(envNames) == 0 {
		t.Fatal("auth.notification_driver should expose environment names")
	}
	t.Setenv(envNames[0], "smtp")

	m := NewManager()
	if err := m.Load(configPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := m.Get().Auth.NotificationDriver; got != "debug" {
		t.Fatalf("notification driver = %q, want config file value debug", got)
	}
}

func TestUpdateWithPersistForceFileOverwritesEnvManagedFileNode(t *testing.T) {
	configPath := copyConfigExampleForTest(t)
	envNames := EnvNamesForPath("auth.signing_key")
	unsetEnvForTest(t, envNames...)
	if len(envNames) == 0 {
		t.Fatal("auth.signing_key should expose environment names")
	}
	t.Setenv(envNames[0], "environment-signing-secret-at-least-32-bytes")

	m := NewManager()
	if err := m.Load(configPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := m.Update(func(cfg *Config) {
		cfg.Auth.SigningKey = "forced-signing-secret-at-least-32-bytes"
	}, WithPersistedPaths("auth.signing_key"), WithEnvManagedPersistMode(EnvManagedPersistForceFile)); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, `signing_key: "forced-signing-secret-at-least-32-bytes"`) {
		t.Fatalf("forced persisted config missing signing key:\n%s", text)
	}
	if strings.Contains(text, "${AUTH_SIGNING_KEY") {
		t.Fatalf("forced update should overwrite signing key placeholder:\n%s", text)
	}
	if !strings.Contains(text, `- "auth.signing_key"`) {
		t.Fatalf("forced update should persist disabled env override path:\n%s", text)
	}

	reloaded := NewManager()
	if err := reloaded.Load(configPath); err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if got := reloaded.Get().Auth.SigningKey; got != "forced-signing-secret-at-least-32-bytes" {
		t.Fatalf("reloaded signing key = %q, want forced file value", got)
	}
}

func TestUpdateWithPersistRuntimeEnvOnlyUsesEnvironmentAndRemovesDisabledPath(t *testing.T) {
	configPath := copyConfigExampleWithEnvOverrideForTest(t, []string{"auth.signing_key"})
	unsetEnvForTest(t, EnvNamesForPath("auth.signing_key")...)
	envNames := EnvNamesForPath("auth.signing_key")
	if len(envNames) == 0 {
		t.Fatal("auth.signing_key should expose environment names")
	}
	envValue := "runtime-env-signing-secret-at-least-32-bytes"
	t.Setenv(envNames[0], envValue)
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config before update: %v", err)
	}

	m := NewManager()
	if err := m.Load(configPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := m.Get().Auth.SigningKey; got != "dev-signing-key-change-me-32-bytes" {
		t.Fatalf("disabled env override should load file/default signing key, got %q", got)
	}
	if err := m.Update(func(cfg *Config) {
		cfg.Auth.SigningKey = "ignored-generated-signing-secret-at-least-32-bytes"
	}, WithPersistedPaths("auth.signing_key"), WithEnvManagedPersistMode(EnvManagedPersistRuntimeEnvOnly)); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config after update: %v", err)
	}
	if string(after) == string(before) {
		t.Fatalf("runtime env-only update should remove disabled path metadata")
	}
	if got := m.Get().Auth.SigningKey; got != envValue {
		t.Fatalf("runtime signing key = %q, want %q", got, envValue)
	}
	text := string(after)
	if strings.Contains(text, `- "auth.signing_key"`) {
		t.Fatalf("runtime env-only update should remove disabled signing key path:\n%s", text)
	}
	if !strings.Contains(text, "${AUTH_SIGNING_KEY:dev-signing-key-change-me-32-bytes}") {
		t.Fatalf("runtime env-only update should not rewrite signing key value:\n%s", text)
	}

	reloaded := NewManager()
	if err := reloaded.Load(configPath); err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if got := reloaded.Get().Auth.SigningKey; got != envValue {
		t.Fatalf("reloaded signing key = %q, want environment value %q", got, envValue)
	}
}

func TestUpdateWithPersistRuntimeEnvOnlyRejectsMissingEnvironment(t *testing.T) {
	configPath := copyConfigExampleForTest(t)
	unsetEnvForTest(t, EnvNamesForPath("auth.signing_key")...)
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config before update: %v", err)
	}

	m := NewManager()
	if err := m.Load(configPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	err = m.Update(func(cfg *Config) {
		cfg.Auth.SigningKey = "ignored-generated-signing-secret-at-least-32-bytes"
	}, WithPersistedPaths("auth.signing_key"), WithEnvManagedPersistMode(EnvManagedPersistRuntimeEnvOnly))
	if err == nil || !strings.Contains(err.Error(), "set one of") {
		t.Fatalf("expected missing environment error, got %v", err)
	}

	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config after update: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("failed runtime env-only update should not write file\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestUpdateWithPersistKeepsWatcherAliveForLaterFileEdits(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath)

	m := NewManager()
	if err := m.Load(configPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := m.Watch(); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	hooks := make(chan *Config, 4)
	m.RegisterHook(func(_, new *Config) {
		hooks <- new
	})

	if err := m.Update(func(cfg *Config) {
		cfg.Server.Port = 19091
	}, WithPersistedPaths("server.port")); err != nil {
		t.Fatalf("persist Update() error = %v", err)
	}
	waitConfigHook(t, hooks, 19091)

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read persisted config: %v", err)
	}
	next := strings.Replace(string(content), "port: 19091", "port: 19092", 1)
	if next == string(content) {
		t.Fatalf("persisted config did not contain expected port:\n%s", content)
	}
	if err := os.WriteFile(configPath, []byte(next), 0600); err != nil {
		t.Fatalf("write manual config change: %v", err)
	}
	waitConfigHook(t, hooks, 19092)
	if got := m.Get().Server.Port; got != 19092 {
		t.Fatalf("expected watcher to load manual config change, got port %d", got)
	}
}

func waitConfigHook(t *testing.T, hooks <-chan *Config, wantPort int) {
	t.Helper()

	deadline := time.After(5 * time.Second)
	for {
		select {
		case cfg := <-hooks:
			if cfg != nil && cfg.Server.Port == wantPort {
				return
			}
		case <-deadline:
			t.Fatalf("timed out waiting for config hook with server.port=%d", wantPort)
		}
	}
}

// taggedEnvName 是当前测试文件的辅助函数，用于复用夹具、断言或输入构造逻辑。
func taggedEnvName(t *testing.T, target any, fieldName string) string {
	t.Helper()

	typ := reflect.TypeOf(target)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	field, ok := typ.FieldByName(fieldName)
	if !ok {
		t.Fatalf("%s has no field %s", typ.Name(), fieldName)
	}
	name := field.Tag.Get("envname")
	if name == "" || name == "-" {
		t.Fatalf("%s.%s has no envname tag", typ.Name(), fieldName)
	}
	return name
}

// setTaggedEnv 是当前测试文件的辅助函数，用于复用夹具、断言或输入构造逻辑。
func setTaggedEnv(t *testing.T, target any, fieldName, value string) {
	t.Helper()

	t.Setenv(EnvPrefixJoin(taggedEnvName(t, target, fieldName)), value)
}

// setUnprefixedTaggedEnv 是当前测试文件的辅助函数，用于复用夹具、断言或输入构造逻辑。
func setUnprefixedTaggedEnv(t *testing.T, target any, fieldName, value string) {
	t.Helper()

	t.Setenv(taggedEnvName(t, target, fieldName), value)
}

// TestEnvNamesUseDynamicAppPrefix 固定配置复制、环境变量覆盖和热加载行为，确保后续注释补全或结构调整不改变该场景。
func TestEnvNamesUseDynamicAppPrefix(t *testing.T) {
	if EnvPrefix() != "RIN_APP" {
		t.Fatalf("EnvPrefix() = %q, want RIN_APP", EnvPrefix())
	}
	dbHostEnvName := taggedEnvName(t, DatabasePostgresConfig{}, "Host")
	if EnvPrefixJoin(dbHostEnvName) != "RIN_APP_DB_POSTGRES_HOST" {
		t.Fatalf("EnvPrefixJoin(%q) = %q, want RIN_APP_DB_POSTGRES_HOST", dbHostEnvName, EnvPrefixJoin(dbHostEnvName))
	}
	if EnvConfigPathName() != "RIN_CONFIG_PATH" {
		t.Fatalf("EnvConfigPathName() = %q, want RIN_CONFIG_PATH", EnvConfigPathName())
	}
}

// TestOverrideWithEnvUsesDynamicPrefixFromAppPrefix 固定配置复制、环境变量覆盖和热加载行为，确保后续注释补全或结构调整不改变该场景。
func TestOverrideWithEnvUsesDynamicPrefixFromAppPrefix(t *testing.T) {
	cfg := testCompleteConfig()

	setUnprefixedTaggedEnv(t, DatabaseConfig{}, "Driver", "mysql")
	setUnprefixedTaggedEnv(t, DatabaseMySQLConfig{}, "Host", "fallback.example.com")
	setUnprefixedTaggedEnv(t, DatabaseMySQLConfig{}, "Port", "3306")
	setUnprefixedTaggedEnv(t, DatabaseMySQLConfig{}, "Username", "fallback")
	setUnprefixedTaggedEnv(t, DatabaseMySQLConfig{}, "Password", "fallback-secret")
	setUnprefixedTaggedEnv(t, DatabaseMySQLConfig{}, "Database", "fallbackdb")
	setUnprefixedTaggedEnv(t, DatabasePoolConfig{}, "MaxOpenConns", "17")
	setUnprefixedTaggedEnv(t, DatabasePoolConfig{}, "MaxIdleConns", "9")
	setTaggedEnv(t, DatabaseConfig{}, "Driver", "postgres")
	setTaggedEnv(t, DatabasePostgresConfig{}, "Host", "db.example.com")
	setTaggedEnv(t, DatabasePostgresConfig{}, "Port", "15432")
	setTaggedEnv(t, DatabasePostgresConfig{}, "Username", "app")
	setTaggedEnv(t, DatabasePostgresConfig{}, "Password", "secret")
	setTaggedEnv(t, DatabasePostgresConfig{}, "Database", "appdb")
	setTaggedEnv(t, DatabasePoolConfig{}, "MaxOpenConns", "42")
	setTaggedEnv(t, DatabasePoolConfig{}, "MaxIdleConns", "21")

	OverrideWithEnv(cfg)

	if cfg.Database.Driver != "postgres" {
		t.Fatalf("Database.Driver = %q, want dynamic prefixed variable to win", cfg.Database.Driver)
	}
	if cfg.Database.Postgres.Host != "db.example.com" {
		t.Fatalf("Database.Postgres.Host = %q, want db.example.com", cfg.Database.Postgres.Host)
	}
	if cfg.Database.Postgres.Port != 15432 {
		t.Fatalf("Database.Postgres.Port = %d, want 15432", cfg.Database.Postgres.Port)
	}
	if cfg.Database.Postgres.Username != "app" {
		t.Fatalf("Database.Postgres.Username = %q, want app", cfg.Database.Postgres.Username)
	}
	if cfg.Database.Postgres.Password != "secret" {
		t.Fatalf("Database.Postgres.Password = %q, want secret", cfg.Database.Postgres.Password)
	}
	if cfg.Database.Postgres.Database != "appdb" {
		t.Fatalf("Database.Postgres.Database = %q, want appdb", cfg.Database.Postgres.Database)
	}
	if cfg.Database.Pool.MaxOpenConns != 42 {
		t.Fatalf("Database.Pool.MaxOpenConns = %d, want 42", cfg.Database.Pool.MaxOpenConns)
	}
	if cfg.Database.Pool.MaxIdleConns != 21 {
		t.Fatalf("Database.Pool.MaxIdleConns = %d, want 21", cfg.Database.Pool.MaxIdleConns)
	}
}

// TestOverrideWithEnvKeepsUnprefixedFallback 固定配置复制、环境变量覆盖和热加载行为，确保后续注释补全或结构调整不改变该场景。
func TestOverrideWithEnvKeepsUnprefixedFallback(t *testing.T) {
	cfg := testCompleteConfig()

	setUnprefixedTaggedEnv(t, DatabaseConfig{}, "Driver", "mysql")
	setUnprefixedTaggedEnv(t, DatabaseMySQLConfig{}, "Host", "fallback.example.com")
	setUnprefixedTaggedEnv(t, DatabaseMySQLConfig{}, "Port", "3306")
	setUnprefixedTaggedEnv(t, DatabaseMySQLConfig{}, "Username", "fallback")
	setUnprefixedTaggedEnv(t, DatabaseMySQLConfig{}, "Password", "fallback-secret")
	setUnprefixedTaggedEnv(t, DatabaseMySQLConfig{}, "Database", "fallbackdb")
	setUnprefixedTaggedEnv(t, DatabasePoolConfig{}, "MaxOpenConns", "17")
	setUnprefixedTaggedEnv(t, DatabasePoolConfig{}, "MaxIdleConns", "9")

	OverrideWithEnv(cfg)

	if cfg.Database.Driver != "mysql" {
		t.Fatalf("Database.Driver = %q, want unprefixed fallback mysql", cfg.Database.Driver)
	}
	if cfg.Database.MySQL.Host != "fallback.example.com" {
		t.Fatalf("Database.MySQL.Host = %q, want fallback.example.com", cfg.Database.MySQL.Host)
	}
	if cfg.Database.MySQL.Port != 3306 {
		t.Fatalf("Database.MySQL.Port = %d, want 3306", cfg.Database.MySQL.Port)
	}
	if cfg.Database.MySQL.Username != "fallback" {
		t.Fatalf("Database.MySQL.Username = %q, want fallback", cfg.Database.MySQL.Username)
	}
	if cfg.Database.MySQL.Password != "fallback-secret" {
		t.Fatalf("Database.MySQL.Password = %q, want fallback-secret", cfg.Database.MySQL.Password)
	}
	if cfg.Database.MySQL.Database != "fallbackdb" {
		t.Fatalf("Database.MySQL.Database = %q, want fallbackdb", cfg.Database.MySQL.Database)
	}
	if cfg.Database.Pool.MaxOpenConns != 17 {
		t.Fatalf("Database.Pool.MaxOpenConns = %d, want 17", cfg.Database.Pool.MaxOpenConns)
	}
	if cfg.Database.Pool.MaxIdleConns != 9 {
		t.Fatalf("Database.Pool.MaxIdleConns = %d, want 9", cfg.Database.Pool.MaxIdleConns)
	}
}

// TestOverrideWithEnvUsesEnvnameTagsForNonDatabaseConfigs 固定配置复制、环境变量覆盖和热加载行为，确保后续注释补全或结构调整不改变该场景。
func TestOverrideWithEnvUsesEnvnameTagsForNonDatabaseConfigs(t *testing.T) {
	cfg := testCompleteConfig()

	setTaggedEnv(t, CacheConfig{}, "Driver", "redis")
	setTaggedEnv(t, RedisCacheConfig{}, "Addr", "redis.example.com:6380")
	setTaggedEnv(t, RedisCacheConfig{}, "Username", "cache-user")
	setTaggedEnv(t, RedisCacheConfig{}, "Password", "redis-secret-2")
	setTaggedEnv(t, RedisCacheConfig{}, "DB", "2")
	setTaggedEnv(t, RedisCacheConfig{}, "PoolSize", "30")
	setTaggedEnv(t, RedisCacheConfig{}, "MinIdleConns", "6")
	setTaggedEnv(t, RedisCacheConfig{}, "MaxRetries", "4")
	setTaggedEnv(t, RedisCacheConfig{}, "DialTimeout", "7")
	setTaggedEnv(t, RedisCacheConfig{}, "ReadTimeout", "8")
	setTaggedEnv(t, RedisCacheConfig{}, "WriteTimeout", "9")
	setTaggedEnv(t, ServerConfig{}, "Host", "0.0.0.0")
	setTaggedEnv(t, ServerConfig{}, "Port", "9090")
	setTaggedEnv(t, ServerConfig{}, "Mode", "release")
	setTaggedEnv(t, ServerConfig{}, "ReadTimeout", "11")
	setTaggedEnv(t, ServerConfig{}, "WriteTimeout", "12")
	setTaggedEnv(t, ServerConfig{}, "IdleTimeout", "13")
	setTaggedEnv(t, LoggerConfig{}, "Level", "warn")
	setTaggedEnv(t, LoggerConfig{}, "Format", "console")
	setTaggedEnv(t, LoggerConfig{}, "ConsoleFormat", "console")
	setTaggedEnv(t, LoggerConfig{}, "FileFormat", "json")
	setTaggedEnv(t, LoggerConfig{}, "Output", "stdout")
	setTaggedEnv(t, LoggerConfig{}, "FilePath", "./logs/env.log")
	setTaggedEnv(t, LoggerConfig{}, "MaxSize", "64")
	setTaggedEnv(t, LoggerConfig{}, "MaxBackups", "3")
	setTaggedEnv(t, LoggerConfig{}, "MaxAge", "14")
	setTaggedEnv(t, I18nConfig{}, "DefaultLocale", "en-US")
	setTaggedEnv(t, I18nConfig{}, "FallbackLocale", "en-US")
	setTaggedEnv(t, I18nConfig{}, "Supported", "zh-CN,en-US")
	setTaggedEnv(t, ExecutorConfig{}, "Enabled", "false")
	setTaggedEnv(t, StorageConfig{}, "Driver", "local")
	setTaggedEnv(t, StorageLocalConfig{}, "FSType", "basepath")
	setTaggedEnv(t, StorageLocalConfig{}, "BasePath", "./env-data")
	setTaggedEnv(t, StorageLocalConfig{}, "EnableWatch", "false")
	setTaggedEnv(t, StorageLocalConfig{}, "WatchBufferSize", "32")
	setTaggedEnv(t, CORSConfig{}, "Enabled", "false")
	setTaggedEnv(t, CORSConfig{}, "AllowOrigins", "https://app.example.com, https://admin.example.com")
	setTaggedEnv(t, CORSConfig{}, "AllowMethods", "GET,POST")
	setTaggedEnv(t, CORSConfig{}, "AllowHeaders", "Origin,X-Request-ID")
	setTaggedEnv(t, CORSConfig{}, "ExposeHeaders", "X-Request-ID,X-Total-Count")
	setTaggedEnv(t, CORSConfig{}, "AllowCredentials", "false")
	setTaggedEnv(t, CORSConfig{}, "MaxAge", "7200")
	setTaggedEnv(t, RPCConfig{}, "Enabled", "true")
	setTaggedEnv(t, RPCConfig{}, "Host", "127.0.0.2")
	setTaggedEnv(t, RPCConfig{}, "Port", "11099")
	setTaggedEnv(t, RPCConfig{}, "ReadTimeout", "14")
	setTaggedEnv(t, RPCConfig{}, "WriteTimeout", "15")
	setTaggedEnv(t, RPCConfig{}, "IdleTimeout", "16")
	setTaggedEnv(t, WebUIConfig{}, "Enabled", "false")
	setTaggedEnv(t, WebUIConfig{}, "MountPath", "/console")
	setTaggedEnv(t, WebUIConfig{}, "DistDir", "./web/dist")
	setTaggedEnv(t, SystemConfig{}, "SeedDefaultsOnStart", "false")

	OverrideWithEnv(cfg)

	if cfg.Cache.Driver != "redis" {
		t.Fatalf("Cache.Driver = %q, want redis", cfg.Cache.Driver)
	}
	if cfg.Cache.Redis.Addr != "redis.example.com:6380" || cfg.Cache.Redis.Username != "cache-user" || cfg.Cache.Redis.Password != "redis-secret-2" {
		t.Fatalf("Cache Redis override mismatch: %#v", cfg.Cache.Redis)
	}
	if cfg.Cache.Redis.DB != 2 || cfg.Cache.Redis.PoolSize != 30 || cfg.Cache.Redis.MinIdleConns != 6 ||
		cfg.Cache.Redis.MaxRetries != 4 || cfg.Cache.Redis.DialTimeout != 7 || cfg.Cache.Redis.ReadTimeout != 8 || cfg.Cache.Redis.WriteTimeout != 9 {
		t.Fatalf("Cache Redis numeric override mismatch: %#v", cfg.Cache.Redis)
	}
	if cfg.Server.Host != "0.0.0.0" || cfg.Server.Port != 9090 || cfg.Server.Mode != "release" ||
		cfg.Server.ReadTimeout != 11 || cfg.Server.WriteTimeout != 12 || cfg.Server.IdleTimeout != 13 {
		t.Fatalf("Server override mismatch: %#v", cfg.Server)
	}
	if cfg.Logger.Level != "warn" || cfg.Logger.Format != "console" || cfg.Logger.ConsoleFormat != "console" ||
		cfg.Logger.FileFormat != "json" || cfg.Logger.Output != "stdout" || cfg.Logger.FilePath != "./logs/env.log" ||
		cfg.Logger.MaxSize != 64 || cfg.Logger.MaxBackups != 3 || cfg.Logger.MaxAge != 14 {
		t.Fatalf("Logger override mismatch: %#v", cfg.Logger)
	}
	if !reflect.DeepEqual(cfg.I18n.Supported, []string{"zh-CN", "en-US"}) ||
		cfg.I18n.DefaultLocale != "en-US" || cfg.I18n.FallbackLocale != "en-US" {
		t.Fatalf("I18n override mismatch: %#v", cfg.I18n)
	}
	if cfg.Executor.Enabled {
		t.Fatalf("Executor.Enabled = true, want false")
	}
	if cfg.Storage.Driver != "local" || cfg.Storage.Local.FSType != "basepath" || cfg.Storage.Local.BasePath != "./env-data" ||
		cfg.Storage.Local.EnableWatch || cfg.Storage.Local.WatchBufferSize != 32 {
		t.Fatalf("Storage override mismatch: %#v", cfg.Storage)
	}
	if cfg.CORS.Enabled || !reflect.DeepEqual(cfg.CORS.AllowOrigins, []string{"https://app.example.com", "https://admin.example.com"}) ||
		!reflect.DeepEqual(cfg.CORS.AllowMethods, []string{"GET", "POST"}) ||
		!reflect.DeepEqual(cfg.CORS.AllowHeaders, []string{"Origin", "X-Request-ID"}) ||
		!reflect.DeepEqual(cfg.CORS.ExposeHeaders, []string{"X-Request-ID", "X-Total-Count"}) ||
		cfg.CORS.AllowCredentials || cfg.CORS.MaxAge != 7200 {
		t.Fatalf("CORS override mismatch: %#v", cfg.CORS)
	}
	if !cfg.RPC.Enabled || cfg.RPC.Host != "127.0.0.2" || cfg.RPC.Port != 11099 ||
		cfg.RPC.ReadTimeout != 14 || cfg.RPC.WriteTimeout != 15 || cfg.RPC.IdleTimeout != 16 {
		t.Fatalf("RPC override mismatch: %#v", cfg.RPC)
	}
	if cfg.WebUI.EnabledValue() || cfg.WebUI.MountPath != "/console" || cfg.WebUI.DistDir != "./web/dist" {
		t.Fatalf("WebUI override mismatch: %#v", cfg.WebUI)
	}
	if cfg.System.SeedDefaultsOnStartValue() {
		t.Fatalf("System.SeedDefaultsOnStart = true, want false")
	}
}

// TestDirectOverrideConfigUsesDynamicPrefix 固定配置复制、环境变量覆盖和热加载行为，确保后续注释补全或结构调整不改变该场景。
func TestDirectOverrideConfigUsesDynamicPrefix(t *testing.T) {
	storageCfg := StorageConfig{}
	setTaggedEnv(t, StorageConfig{}, "Driver", "local")
	setTaggedEnv(t, StorageLocalConfig{}, "FSType", "memory")
	setTaggedEnv(t, StorageLocalConfig{}, "WatchBufferSize", "64")
	storageCfg.OverrideConfig()
	if storageCfg.Driver != "local" || storageCfg.Local.FSType != "memory" || storageCfg.Local.WatchBufferSize != 64 {
		t.Fatalf("Storage OverrideConfig mismatch: %#v", storageCfg)
	}

	corsCfg := CORSConfig{}
	setTaggedEnv(t, CORSConfig{}, "AllowOrigins", "https://app.example.com,https://admin.example.com")
	setTaggedEnv(t, CORSConfig{}, "MaxAge", "1800")
	corsCfg.OverrideConfig()
	if !reflect.DeepEqual(corsCfg.AllowOrigins, []string{"https://app.example.com", "https://admin.example.com"}) ||
		corsCfg.MaxAge != 1800 {
		t.Fatalf("CORS OverrideConfig mismatch: %#v", corsCfg)
	}
}

// TestManagerLoadAutoLoadsDotEnvWithDynamicPrefix 固定配置复制、环境变量覆盖和热加载行为，确保后续注释补全或结构调整不改变该场景。
func TestManagerLoadAutoLoadsDotEnvWithDynamicPrefix(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	dotEnvPath := filepath.Join(tempDir, EnvFilePath)
	serverPortName := taggedEnvName(t, ServerConfig{}, "Port")
	serverPortEnv := EnvPrefixJoin(serverPortName)

	unsetEnvForTest(t, serverPortEnv, serverPortName)
	writeTestConfig(t, configPath)
	if err := os.WriteFile(dotEnvPath, []byte(serverPortEnv+"=19090\n"), 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Errorf("restore wd: %v", err)
		}
	})

	m := NewManager()
	if err := m.Load(configPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got := m.Get()
	if got.Server.Port != 19090 {
		t.Fatalf("Server.Port = %d, want 19090 from .env", got.Server.Port)
	}
}

// testCompleteConfig 是当前测试文件的辅助函数，用于复用夹具、断言或输入构造逻辑。
func testCompleteConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "127.0.0.1",
			Port:         8080,
			Mode:         "test",
			ReadTimeout:  5,
			WriteTimeout: 10,
			IdleTimeout:  60,
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			SQLite: DatabaseSQLiteConfig{
				Path: ":memory:",
			},
			MySQL: DatabaseMySQLConfig{
				Host:     "localhost",
				Port:     3306,
				Username: "user",
				Password: "secret",
				Database: "app",
				Charset:  "utf8mb4",
			},
			Postgres: DatabasePostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Username: "user",
				Password: "secret",
				Database: "app",
				SSLMode:  "disable",
			},
			Pool: DatabasePoolConfig{
				MaxOpenConns: 10,
				MaxIdleConns: 5,
			},
		},
		Cache: CacheConfig{
			Driver: "local",
			Local: LocalCacheConfig{
				MaxCost:           67108864,
				NumCounters:       1000000,
				BufferItems:       64,
				DefaultTTLSeconds: 0,
			},
			Redis: RedisCacheConfig{
				Addr:         "127.0.0.1:6379",
				Password:     "redis-secret",
				DB:           1,
				PoolSize:     20,
				MinIdleConns: 5,
				MaxRetries:   3,
				DialTimeout:  5,
				ReadTimeout:  3,
				WriteTimeout: 3,
			},
		},
		Logger: LoggerConfig{
			Level:         "debug",
			Format:        "json",
			ConsoleFormat: "console",
			FileFormat:    "json",
			Output:        "both",
			FilePath:      "./logs/app.log",
			MaxSize:       100,
			MaxBackups:    7,
			MaxAge:        30,
		},
		I18n: I18nConfig{
			DefaultLocale:  "zh-CN",
			FallbackLocale: "zh-CN",
			Supported:      []string{"zh-CN", "en-US"},
			Resources: map[string]string{
				"ui":         "./configs/locales/ui",
				"api":        "./configs/locales/api",
				"validation": "./configs/locales/validation",
				"system":     "./configs/locales/system",
			},
		},
		Brand: BrandConfig{
			ProductName: "Aoi Admin",
			ProductCode: "aoi-admin",
			VersionName: "Community",
		},
		Executor: ExecutorConfig{
			Enabled: true,
			Pools: []ExecutorPoolConfig{
				{
					Name:        "default",
					Size:        10,
					Expiry:      30,
					NonBlocking: true,
				},
			},
		},
		Storage: StorageConfig{
			Driver: "local",
			Local: StorageLocalConfig{
				FSType:          "memory",
				BasePath:        "./data",
				PublicURL:       "/uploads",
				EnableWatch:     true,
				WatchBufferSize: 16,
			},
		},
		System: SystemConfig{
			SeedDefaultsOnStart: boolPtr(true),
		},
		CORS: CORSConfig{
			Enabled:          true,
			AllowOrigins:     []string{"https://example.com"},
			AllowMethods:     []string{"GET", "POST"},
			AllowHeaders:     []string{"Origin", "Content-Type"},
			ExposeHeaders:    []string{"X-Request-ID"},
			AllowCredentials: true,
			MaxAge:           3600,
		},
		RPC: RPCConfig{
			Enabled:      false,
			Host:         "127.0.0.1",
			Port:         10099,
			ReadTimeout:  10,
			WriteTimeout: 10,
			IdleTimeout:  30,
		},
		WebUI: WebUIConfig{
			Enabled:   boolPtr(true),
			MountPath: "/",
			DistDir:   "./web/app/build/client",
		},
		EnvOverride: EnvOverrideConfig{
			DisabledPaths: []string{"auth.signing_key"},
		},
	}
}

// boolPtr 是当前测试文件的辅助函数，用于复用夹具、断言或输入构造逻辑。
func TestManagerLoadUsesOnlyExplicitConfigFile(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.yaml")
	writeTestConfig(t, configPath)
	pluginConfigDir := filepath.Join(root, "_examples", "remote-plugins", "demo")
	if err := os.MkdirAll(pluginConfigDir, 0o700); err != nil {
		t.Fatalf("mkdir plugin config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginConfigDir, "config.yaml"), []byte("plugins:\n  manifests:\n    - private.yaml\n"), 0o600); err != nil {
		t.Fatalf("write plugin private config: %v", err)
	}

	manager := NewManager()
	if err := manager.Load(configPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg := manager.Get(); cfg == nil || cfg.Database.Driver != "sqlite" {
		t.Fatalf("loaded config = %#v", cfg)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

// writeTestConfig 写入测试夹具文件，并把文件系统准备细节限制在测试辅助层。
func writeTestConfig(t *testing.T, path string) {
	t.Helper()

	const content = `
server:
  host: 127.0.0.1
  port: 8080
  mode: test
  read_timeout: 5
  write_timeout: 10
  idle_timeout: 60
database:
  driver: sqlite
  sqlite:
    path: ":memory:"
  mysql:
    host: localhost
    port: 3306
    username: user
    password: secret
    database: app
    charset: utf8mb4
  postgres:
    host: localhost
    port: 5432
    username: user
    password: secret
    database: app
    sslMode: disable
  pool:
    maxOpenConns: 10
    maxIdleConns: 5
cache:
  driver: local
  local:
    maxCost: 67108864
    numCounters: 1000000
    bufferItems: 64
    defaultTtlSeconds: 0
  redis:
    addr: 127.0.0.1:6379
    username: ""
    password: redis-secret
    db: 1
logger:
  level: debug
  format: json
  output: stdout
i18n:
  defaultLocale: zh-CN
  fallbackLocale: zh-CN
  supportedLocales:
    - zh-CN
  resources:
    ui: ./configs/locales/ui
    api: ./configs/locales/api
    validation: ./configs/locales/validation
    system: ./configs/locales/system
brand:
  productName: Aoi Admin
  productCode: aoi-admin
  versionName: Community
executor:
  enabled: false
  pools:
    - name: default
      size: 10
      expiry: 30
      non_blocking: true
storage:
  driver: local
  local:
    fsType: memory
    basePath: ./data
    publicUrl: /uploads
    enableWatch: false
    watchBufferSize: 16
cors:
  enabled: false
  allow_origins:
    - "*"
  allow_methods:
    - "GET"
    - "POST"
  allow_headers:
    - "Origin"
    - "Content-Type"
  expose_headers:
    - "X-Request-ID"
  allow_credentials: false
  max_age: 3600
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func copyConfigExampleForTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, "configs", "config.example.yaml"))
	if err != nil {
		t.Fatalf("read config example: %v", err)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write config copy: %v", err)
	}
	return path
}

func copyConfigExampleWithEnvOverrideForTest(t *testing.T, disabledPaths []string) string {
	t.Helper()
	path := copyConfigExampleForTest(t)
	var builder strings.Builder
	builder.WriteString("disabled_paths:\n")
	for _, disabledPath := range disabledPaths {
		builder.WriteString("    - ")
		builder.WriteString(disabledPath)
		builder.WriteByte('\n')
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config copy: %v", err)
	}
	next := strings.Replace(string(raw), "disabled_paths: []", builder.String(), 1)
	if next == string(raw) {
		t.Fatalf("config copy did not contain env_override disabled_paths placeholder")
	}
	if err := os.WriteFile(path, []byte(next), 0o600); err != nil {
		t.Fatalf("write env override config copy: %v", err)
	}
	return path
}

// unsetEnvForTest 清理测试期间设置的环境变量或全局状态，避免用例之间互相污染。
func unsetEnvForTest(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		key := key
		oldValue, existed := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		t.Cleanup(func() {
			if existed {
				if err := os.Setenv(key, oldValue); err != nil {
					t.Errorf("restore %s: %v", key, err)
				}
				return
			}
			if err := os.Unsetenv(key); err != nil {
				t.Errorf("restore unset %s: %v", key, err)
			}
		})
	}
}
