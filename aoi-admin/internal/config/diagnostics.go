package config

import (
	"fmt"
	"strings"

	"github.com/rei0721/go-scaffold/pkg/configloader"
)

// ConfigDiagnosticSeverity 表示配置诊断的严重级别。
type ConfigDiagnosticSeverity string

const (
	ConfigDiagnosticError ConfigDiagnosticSeverity = "error"
)

// ConfigDiagnostic 描述一个可由 CLI 或 WebUI 展示的配置问题。
//
// EnvNames 会列出该配置路径可用的环境变量名，帮助用户选择是修复文件还是补齐运行时环境。
type ConfigDiagnostic struct {
	Section  string
	Path     string
	Message  string
	EnvNames []string
	Severity ConfigDiagnosticSeverity
}

// LoadDiagnostics 加载配置文件并返回不阻断解析的配置诊断列表。
//
// 与 Manager.Load 不同，该函数不直接执行 Validate，而是保留可读取的配置快照，让启动向导能一次性展示
// 多个可修复问题。
func LoadDiagnostics(configPath string) (*Config, []ConfigDiagnostic, error) {
	m := &manager{
		v:     configloader.New(),
		hooks: make([]HookHandler, 0),
	}
	m.configPath = configPath

	LoadEnv()
	m.v.SetConfigFile(configPath)
	if err := m.v.ReadInConfig(); err != nil {
		return nil, nil, fmt.Errorf("failed to read config file: %w", err)
	}
	if err := m.processEnvSubstitution(); err != nil {
		return nil, nil, fmt.Errorf("failed to process env substitution: %w", err)
	}

	cfg := &Config{}
	if err := m.v.Unmarshal(cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	OverrideWithEnvExcept(cfg, cfg.EnvOverride.DisabledPaths)
	return cfg, cfg.Diagnostics(), nil
}

// Diagnostics 对当前配置快照执行跨区段诊断。
//
// 返回值只表达需要用户处理的问题，不修改业务配置；唯一的归一化副作用是清理 env_override.disabled_paths。
func (c *Config) Diagnostics() []ConfigDiagnostic {
	if c == nil {
		return []ConfigDiagnostic{newDiagnostic("", "", "config is required")}
	}
	var diagnostics []ConfigDiagnostic
	add := func(section string, path string, message string) {
		diagnostics = append(diagnostics, newDiagnostic(section, path, message))
	}

	c.diagnoseServer(add)
	c.diagnoseDatabase(add)
	c.diagnoseCache(add)
	c.diagnoseLogger(add)
	c.diagnoseI18n(add)
	c.diagnoseExecutor(add)
	c.diagnoseStorage(add)
	c.diagnoseCORS(add)
	c.diagnoseRPC(add)
	c.diagnoseAuth(add)
	c.diagnoseWebUI(add)
	c.diagnosePlugins(add)
	if c.Plugins.Enabled && !c.Auth.Enabled {
		add(AppPluginsName, "plugins.enabled", "auth must be enabled when plugins are enabled")
	}
	c.EnvOverride.DisabledPaths = normalizeConfigPaths(c.EnvOverride.DisabledPaths)
	return diagnostics
}

// newDiagnostic 创建带环境变量候选名的错误级诊断项。
func newDiagnostic(section string, path string, message string) ConfigDiagnostic {
	return ConfigDiagnostic{
		Section:  section,
		Path:     path,
		Message:  message,
		EnvNames: EnvNamesForPath(path),
		Severity: ConfigDiagnosticError,
	}
}

func (c *Config) diagnoseServer(add func(string, string, string)) {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		add(AppServerName, "server.port", "port must be between 1 and 65535")
	}
	if c.Server.Mode != "debug" && c.Server.Mode != "release" && c.Server.Mode != "test" {
		add(AppServerName, "server.mode", "mode must be debug, release, or test")
	}
	if c.Server.ReadTimeout <= 0 {
		add(AppServerName, "server.read_timeout", "read_timeout must be positive")
	}
	if c.Server.WriteTimeout <= 0 {
		add(AppServerName, "server.write_timeout", "write_timeout must be positive")
	}
}

func (c *Config) diagnoseDatabase(add func(string, string, string)) {
	driver := strings.ToLower(strings.TrimSpace(c.Database.Driver))
	validDrivers := map[string]bool{"postgres": true, "mysql": true, "sqlite": true}
	if !validDrivers[driver] {
		add(AppDatabaseName, "database.driver", "driver must be postgres, mysql, or sqlite")
		return
	}
	switch driver {
	case DatabaseDriverSQLite:
		if strings.TrimSpace(c.Database.SQLite.Path) == "" {
			add(AppDatabaseName, "database.sqlite.path", "sqlite.path is required")
		}
	case DatabaseDriverMySQL:
		diagnoseNetworkDatabase(add, AppDatabaseName, "database.mysql", c.Database.MySQL.Host, c.Database.MySQL.Port, c.Database.MySQL.Username, c.Database.MySQL.Database)
	case DatabaseDriverPostgres:
		diagnoseNetworkDatabase(add, AppDatabaseName, "database.postgres", c.Database.Postgres.Host, c.Database.Postgres.Port, c.Database.Postgres.Username, c.Database.Postgres.Database)
	}
	if c.Database.Pool.MaxOpenConns < 0 {
		add(AppDatabaseName, "database.pool.maxOpenConns", "pool.maxOpenConns must be non-negative")
	}
	if c.Database.Pool.MaxIdleConns < 0 {
		add(AppDatabaseName, "database.pool.maxIdleConns", "pool.maxIdleConns must be non-negative")
	}
}

func diagnoseNetworkDatabase(add func(string, string, string), section, basePath, host string, port int, username, databaseName string) {
	if strings.TrimSpace(host) == "" {
		add(section, basePath+".host", "host is required")
	}
	if port <= 0 || port > 65535 {
		add(section, basePath+".port", "port must be between 1 and 65535")
	}
	if strings.TrimSpace(username) == "" {
		add(section, basePath+".username", "username is required")
	}
	if strings.TrimSpace(databaseName) == "" {
		add(section, basePath+".database", "database is required")
	}
}

func (c *Config) diagnoseCache(add func(string, string, string)) {
	driver := strings.ToLower(strings.TrimSpace(c.Cache.Driver))
	switch driver {
	case CacheDriverDisabled, CacheDriverLocal:
	case CacheDriverRedis, CacheDriverHybrid:
		if strings.TrimSpace(c.Cache.Redis.Addr) == "" {
			add(AppCacheName, "cache.redis.addr", "redis addr is required")
		}
		if c.Cache.Redis.DB < 0 || c.Cache.Redis.DB > 15 {
			add(AppCacheName, "cache.redis.db", "db must be between 0 and 15")
		}
		if c.Cache.Redis.PoolSize < 0 {
			add(AppCacheName, "cache.redis.poolSize", "poolSize must be non-negative")
		}
	case "":
		add(AppCacheName, "cache.driver", "driver is required")
	default:
		add(AppCacheName, "cache.driver", "driver must be disabled, local, redis, or hybrid")
	}
}

func (c *Config) diagnoseLogger(add func(string, string, string)) {
	if !stringInSet(c.Logger.Level, "debug", "info", "warn", "error") {
		add(AppLoggerName, "logger.level", "level must be debug, info, warn, or error")
	}
	if !stringInSet(c.Logger.Format, "json", "console") {
		add(AppLoggerName, "logger.format", "format must be json or console")
	}
	if !stringInSet(c.Logger.Output, "stdout", "file", "both") {
		add(AppLoggerName, "logger.output", "output must be stdout, file, or both")
	}
}

func (c *Config) diagnoseI18n(add func(string, string, string)) {
	if strings.TrimSpace(c.I18n.DefaultLocale) == "" {
		add(AppI18nName, "i18n.defaultLocale", "defaultLocale is required")
	}
	if len(c.I18n.Supported) == 0 {
		add(AppI18nName, "i18n.supportedLocales", "at least one supported locale is required")
		return
	}
	found := false
	for _, supported := range c.I18n.Supported {
		if supported == c.I18n.DefaultLocale {
			found = true
			break
		}
	}
	if !found {
		add(AppI18nName, "i18n.defaultLocale", "defaultLocale must be in supportedLocales")
	}
	if strings.TrimSpace(c.I18n.FallbackLocale) == "" {
		add(AppI18nName, "i18n.fallbackLocale", "fallbackLocale is required")
	}
	for _, namespace := range []string{"ui", "api", "validation", "system"} {
		if strings.TrimSpace(c.I18n.Resources[namespace]) == "" {
			add(AppI18nName, "i18n.resources."+namespace, "resource directory is required")
		}
	}
}

func (c *Config) diagnoseExecutor(add func(string, string, string)) {
	if !c.Executor.Enabled {
		return
	}
	if len(c.Executor.Pools) == 0 {
		add(AppExecutorName, "executor.pools", "at least one pool is required when executor is enabled")
		return
	}
	seen := map[string]struct{}{}
	for i, pool := range c.Executor.Pools {
		basePath := fmt.Sprintf("executor.pools.%d", i)
		name := strings.TrimSpace(pool.Name)
		if name == "" {
			add(AppExecutorName, basePath+".name", "pool name is required")
		} else if _, ok := seen[name]; ok {
			add(AppExecutorName, basePath+".name", "duplicate pool name: "+name)
		} else {
			seen[name] = struct{}{}
		}
		if pool.Size <= 0 {
			add(AppExecutorName, basePath+".size", "pool size must be positive")
		}
		if pool.Size > 10000 {
			add(AppExecutorName, basePath+".size", "pool size must not exceed 10000")
		}
		if pool.Expiry < 0 {
			add(AppExecutorName, basePath+".expiry", "pool expiry must be non-negative")
		}
	}
}

func (c *Config) diagnoseStorage(add func(string, string, string)) {
	driver := strings.ToLower(strings.TrimSpace(c.Storage.Driver))
	if driver == StorageDriverDisabled {
		return
	}
	if !stringInSet(driver, StorageDriverLocal, StorageDriverS3, StorageDriverMinIO, StorageDriverLocalS3, StorageDriverLocalMinIO) {
		add(AppStorageName, "storage.driver", "driver must be one of: disabled, local, s3, minio, local+s3, local+minio")
	}
	if driver == StorageDriverLocal || driver == StorageDriverLocalS3 || driver == StorageDriverLocalMinIO {
		fsType := firstNonEmpty(c.Storage.Local.FSType, "os")
		if !stringInSet(fsType, "os", "memory", "readonly", "basepath") {
			add(AppStorageName, "storage.local.fsType", "local fsType must be one of: os, memory, readonly, basepath")
		}
		if fsType == "basepath" && strings.TrimSpace(c.Storage.Local.BasePath) == "" {
			add(AppStorageName, "storage.local.basePath", "local basePath is required when fsType is basepath")
		}
	}
	if driver == StorageDriverS3 || driver == StorageDriverLocalS3 {
		diagnoseObjectStorage(add, "storage.s3", c.Storage.S3.Endpoint, c.Storage.S3.Bucket, c.Storage.S3.AccessKeyID, c.Storage.S3.SecretAccessKey)
	}
	if driver == StorageDriverMinIO || driver == StorageDriverLocalMinIO {
		diagnoseObjectStorage(add, "storage.minio", c.Storage.MinIO.Endpoint, c.Storage.MinIO.Bucket, c.Storage.MinIO.AccessKeyID, c.Storage.MinIO.SecretAccessKey)
	}
	if c.Storage.Local.WatchBufferSize < 0 {
		add(AppStorageName, "storage.local.watchBufferSize", "watchBufferSize must be non-negative")
	}
}

func diagnoseObjectStorage(add func(string, string, string), basePath string, endpoint, bucket, accessKeyID, secretAccessKey string) {
	if strings.TrimSpace(endpoint) == "" {
		add(AppStorageName, basePath+".endpoint", "endpoint is required")
	}
	if strings.TrimSpace(bucket) == "" {
		add(AppStorageName, basePath+".bucket", "bucket is required")
	}
	if strings.TrimSpace(accessKeyID) == "" {
		add(AppStorageName, basePath+".accessKeyId", "accessKeyId is required")
	}
	if strings.TrimSpace(secretAccessKey) == "" {
		add(AppStorageName, basePath+".secretAccessKey", "secretAccessKey is required")
	}
}

func (c *Config) diagnoseCORS(add func(string, string, string)) {
	if !c.CORS.Enabled {
		return
	}
	if c.CORS.AllowCredentials {
		for _, origin := range c.CORS.AllowOrigins {
			if origin == "*" {
				add(AppCORSName, "cors.allow_origins", "allow_origins cannot contain wildcard \"*\" when allow_credentials is true")
				break
			}
		}
	}
	if c.CORS.MaxAge < 0 {
		add(AppCORSName, "cors.max_age", "max_age must be non-negative")
	}
}

func (c *Config) diagnoseRPC(add func(string, string, string)) {
	if !c.RPC.Enabled {
		return
	}
	c.RPC.ApplyDefaults()
	if strings.TrimSpace(c.RPC.Host) == "" {
		add(AppRPCName, "rpc.host", "host is required")
	}
	if c.RPC.Port <= 0 || c.RPC.Port > 65535 {
		add(AppRPCName, "rpc.port", "port must be between 1 and 65535")
	}
	if c.RPC.ReadTimeout <= 0 {
		add(AppRPCName, "rpc.read_timeout", "read_timeout must be positive")
	}
	if c.RPC.WriteTimeout <= 0 {
		add(AppRPCName, "rpc.write_timeout", "write_timeout must be positive")
	}
	if c.RPC.IdleTimeout < 0 {
		add(AppRPCName, "rpc.idle_timeout", "idle_timeout must be non-negative")
	}
}

func (c *Config) diagnoseAuth(add func(string, string, string)) {
	if !c.Auth.Enabled {
		return
	}
	c.Auth.ApplyDefaults()
	if strings.TrimSpace(c.Auth.Issuer) == "" {
		add(AppAuthName, "auth.issuer", "issuer is required")
	}
	if len(c.Auth.SigningKey) < 32 {
		add(AppAuthName, "auth.signing_key", "signing_key must be at least 32 bytes")
	}
	if strings.TrimSpace(c.Auth.RefreshTokenPepper) == "" {
		add(AppAuthName, "auth.refresh_token_pepper", "refresh_token_pepper is required")
	}
	if len(c.Auth.MFASecretKey) < 32 {
		add(AppAuthName, "auth.mfa_secret_key", "mfa_secret_key must be at least 32 bytes")
	}
	if c.Auth.AccessTokenTTLSeconds <= 0 {
		add(AppAuthName, "auth.access_token_ttl_seconds", "access_token_ttl_seconds must be positive")
	}
	if c.Auth.RefreshTokenTTLSeconds <= 0 {
		add(AppAuthName, "auth.refresh_token_ttl_seconds", "refresh_token_ttl_seconds must be positive")
	}
	if c.Auth.InvitationTTLSeconds <= 0 {
		add(AppAuthName, "auth.invitation_ttl_seconds", "invitation_ttl_seconds must be positive")
	}
	if !validRegistrationMode(c.Auth.RegistrationMode) {
		add(AppAuthName, "auth.registration_mode", "registration_mode must be one of disabled, direct, email_verification, invite_only")
	}
	if c.Auth.EmailVerificationTTLSeconds <= 0 {
		add(AppAuthName, "auth.email_verification_ttl_seconds", "email_verification_ttl_seconds must be positive")
	}
	if c.Auth.PasswordResetTTLSeconds <= 0 {
		add(AppAuthName, "auth.password_reset_ttl_seconds", "password_reset_ttl_seconds must be positive")
	}
	if c.Auth.LoginMaxFailures <= 0 {
		add(AppAuthName, "auth.login_max_failures", "login_max_failures must be positive")
	}
	if c.Auth.LoginLockMinutes <= 0 {
		add(AppAuthName, "auth.login_lock_minutes", "login_lock_minutes must be positive")
	}
	if c.Auth.LoginCaptchaEnabled && c.Auth.CaptchaTTLSeconds <= 0 {
		add(AppAuthName, "auth.captcha_ttl_seconds", "captcha_ttl_seconds must be positive when login captcha is enabled")
	}
	if c.Auth.PasswordPolicy.MinLength <= 0 {
		add(AppAuthName, "auth.password_policy.min_length", "password policy min_length must be positive")
	}
	if strings.EqualFold(c.Auth.NotificationDriver, "smtp") {
		if strings.TrimSpace(c.Auth.SMTP.Host) == "" {
			add(AppAuthName, "auth.smtp.host", "smtp host is required when notification_driver is smtp")
		}
		if c.Auth.SMTP.Port <= 0 {
			add(AppAuthName, "auth.smtp.port", "smtp port is required when notification_driver is smtp")
		}
		if strings.TrimSpace(c.Auth.SMTP.From) == "" {
			add(AppAuthName, "auth.smtp.from", "smtp from is required when notification_driver is smtp")
		}
		if strings.TrimSpace(c.Auth.SMTP.Security) == "" {
			add(AppAuthName, "auth.smtp.security", "smtp security is required when notification_driver is smtp")
		} else if !validSMTPSecurity(c.Auth.SMTP.Security) {
			add(AppAuthName, "auth.smtp.security", "smtp security must be one of none, starttls, tls")
		}
	}
}

func (c *Config) diagnoseWebUI(add func(string, string, string)) {
	c.WebUI.ApplyDefaults()
	if !c.WebUI.EnabledValue() {
		return
	}
	if c.WebUI.MountPath == "" || !strings.HasPrefix(c.WebUI.MountPath, "/") {
		add(AppWebUIName, "webui.mount_path", "mount_path must start with /")
	}
	if webUIReservedPath(c.WebUI.MountPath) {
		add(AppWebUIName, "webui.mount_path", "mount_path conflicts with reserved API or probe path")
	}
	if strings.TrimSpace(c.WebUI.DistDir) == "" {
		add(AppWebUIName, "webui.dist_dir", "dist_dir is required")
	}
	c.WebUI.PublicBaseURL = strings.TrimRight(strings.TrimSpace(c.WebUI.PublicBaseURL), "/")
}

func (c *Config) diagnosePlugins(add func(string, string, string)) {
	if !c.Plugins.Enabled {
		return
	}
	c.Plugins.ApplyDefaults()
	if c.Plugins.BasePath == "" || !strings.HasPrefix(c.Plugins.BasePath, "/") || c.Plugins.BasePath == "/" {
		add(AppPluginsName, "plugins.base_path", "base_path must be an absolute non-root path")
	}
	if strings.TrimSpace(c.Plugins.DefaultProtocolVersion) == "" {
		add(AppPluginsName, "plugins.default_protocol_version", "default_protocol_version is required")
	}
	if len(c.Plugins.AllowedTransports) == 0 {
		add(AppPluginsName, "plugins.allowed_transports", "allowed_transports must not be empty")
	}
	for _, transport := range c.Plugins.AllowedTransports {
		if !stringInSet(strings.ToLower(strings.TrimSpace(transport)), "http", "websocket", "rpc") {
			add(AppPluginsName, "plugins.allowed_transports", "allowed_transports must contain only http, websocket, or rpc")
			break
		}
	}
	if c.Plugins.RequestTimeoutSeconds <= 0 {
		add(AppPluginsName, "plugins.request_timeout_seconds", "request_timeout_seconds must be positive")
	}
	if c.Plugins.HeartbeatTimeoutSeconds <= 0 {
		add(AppPluginsName, "plugins.heartbeat_timeout_seconds", "heartbeat_timeout_seconds must be positive")
	}
	if c.Plugins.LeaseTTLSeconds <= 0 {
		add(AppPluginsName, "plugins.lease_ttl_seconds", "lease_ttl_seconds must be positive")
	}
	if c.Plugins.LeaseScanIntervalSeconds <= 0 {
		add(AppPluginsName, "plugins.lease_scan_interval_seconds", "lease_scan_interval_seconds must be positive")
	}
	if c.Plugins.RetryCount < 0 {
		add(AppPluginsName, "plugins.retry_count", "retry_count must not be negative")
	}
	if !stringInSet(strings.ToLower(strings.TrimSpace(c.Plugins.RegistryBackend)), "db", "memory") {
		add(AppPluginsName, "plugins.registry_backend", "registry_backend must be db or memory")
	}
	if !stringInSet(strings.ToLower(strings.TrimSpace(c.Plugins.RouterStrategy)), "round_robin") {
		add(AppPluginsName, "plugins.router_strategy", "router_strategy must be round_robin")
	}
	if !stringInSet(strings.ToLower(strings.TrimSpace(c.Plugins.RegistrationAuthMode)), "none", "shared_secret", "signature") {
		add(AppPluginsName, "plugins.registration_auth_mode", "registration_auth_mode must be none, shared_secret, or signature")
	}
	if strings.EqualFold(strings.TrimSpace(c.Plugins.RegistrationAuthMode), "shared_secret") && strings.TrimSpace(c.Plugins.SharedSecretEnv) == "" {
		add(AppPluginsName, "plugins.shared_secret_env", "shared_secret_env is required when registration_auth_mode is shared_secret")
	}
	if !c.Plugins.HTTPEnabled && !c.Plugins.WSEnabled && !c.Plugins.RPCEnabled {
		add(AppPluginsName, "plugins.allowed_transports", "at least one plugin transport endpoint must be enabled")
	}
}

// stringInSet 用于配置枚举值校验，保持各诊断函数的条件表达式简洁。
func stringInSet(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}
