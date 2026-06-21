package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rei0721/go-scaffold/pkg/configloader"
)

func TestDocumentedExampleConfigYAMLFilesAreValid(t *testing.T) {
	root := configExamplesRepoRoot(t)
	files := documentedExampleConfigFiles(t, root)

	for _, file := range files {
		file := file
		t.Run(filepath.ToSlash(strings.TrimPrefix(file, root+string(os.PathSeparator))), func(t *testing.T) {
			loader := configloader.New()
			loader.SetConfigFile(file)
			if err := loader.ReadInConfig(); err != nil {
				t.Fatalf("parse example config: %v", err)
			}
			if len(loader.AllSettings()) == 0 {
				t.Fatal("example config must contain a mapping document")
			}
		})
	}
}

func TestDocumentedExampleConfigsLoadWithControlledEnvironment(t *testing.T) {
	root := configExamplesRepoRoot(t)
	files := documentedExampleConfigFiles(t, root)

	for _, file := range files {
		file := file
		t.Run(filepath.ToSlash(strings.TrimPrefix(file, root+string(os.PathSeparator))), func(t *testing.T) {
			setControlledExampleEnv(t, filepath.Base(file))
			mgr := NewManager()
			if err := mgr.Load(file); err != nil {
				t.Fatalf("load example config: %v", err)
			}
			if cfg := mgr.Get(); cfg == nil {
				t.Fatal("loaded config is nil")
			}
		})
	}
}

func documentedExampleConfigFiles(t *testing.T, root string) []string {
	t.Helper()

	files := []string{
		filepath.Join(root, "configs", "config.example.yaml"),
		filepath.Join(root, "deploy", "config.production.example.yaml"),
	}
	matches, err := filepath.Glob(filepath.Join(root, "configs", "examples", "*.example.yaml"))
	if err != nil {
		t.Fatalf("glob scenario examples: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no scenario example configs found")
	}
	files = append(files, matches...)
	return files
}

func configExamplesRepoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func setControlledExampleEnv(t *testing.T, fileName string) {
	t.Helper()

	env := map[string]string{
		"RIN_APP_SERVER_MODE":                         "debug",
		"RIN_APP_DB_DRIVER":                           "sqlite",
		"RIN_APP_DB_SQLITE_PATH":                      "./data/test-example.db",
		"RIN_APP_DB_MYSQL_HOST":                       "127.0.0.1",
		"RIN_APP_DB_MYSQL_PORT":                       "3306",
		"RIN_APP_DB_MYSQL_USERNAME":                   "go_scaffold",
		"RIN_APP_DB_MYSQL_PASSWORD":                   "example-db-password",
		"RIN_APP_DB_MYSQL_DATABASE":                   "go_scaffold",
		"RIN_APP_DB_MYSQL_CHARSET":                    "utf8mb4",
		"RIN_APP_DB_POSTGRES_HOST":                    "127.0.0.1",
		"RIN_APP_DB_POSTGRES_PORT":                    "5432",
		"RIN_APP_DB_POSTGRES_USERNAME":                "go_scaffold",
		"RIN_APP_DB_POSTGRES_PASSWORD":                "example-db-password",
		"RIN_APP_DB_POSTGRES_DATABASE":                "go_scaffold",
		"RIN_APP_DB_POSTGRES_SSL_MODE":                "disable",
		"RIN_APP_DB_POOL_MAX_OPEN_CONNS":              "10",
		"RIN_APP_DB_POOL_MAX_IDLE_CONNS":              "5",
		"RIN_APP_CACHE_DRIVER":                        "local",
		"RIN_APP_CACHE_LOCAL_MAX_COST":                "67108864",
		"RIN_APP_CACHE_LOCAL_NUM_COUNTERS":            "1000000",
		"RIN_APP_CACHE_LOCAL_BUFFER_ITEMS":            "64",
		"RIN_APP_CACHE_LOCAL_DEFAULT_TTL_SECONDS":     "0",
		"RIN_APP_CACHE_REDIS_ADDR":                    "127.0.0.1:6379",
		"RIN_APP_CACHE_REDIS_USERNAME":                "",
		"RIN_APP_CACHE_REDIS_PASSWORD":                "",
		"RIN_APP_CACHE_REDIS_DB":                      "0",
		"RIN_APP_CACHE_REDIS_POOL_SIZE":               "10",
		"RIN_APP_CACHE_REDIS_MIN_IDLE_CONNS":          "2",
		"RIN_APP_CACHE_REDIS_MAX_RETRIES":             "2",
		"RIN_APP_CACHE_REDIS_DIAL_TIMEOUT":            "5",
		"RIN_APP_CACHE_REDIS_READ_TIMEOUT":            "3",
		"RIN_APP_CACHE_REDIS_WRITE_TIMEOUT":           "3",
		"RIN_APP_LOG_LEVEL":                           "info",
		"RIN_APP_LOG_FORMAT":                          "console",
		"RIN_APP_LOG_CONSOLE_FORMAT":                  "console",
		"RIN_APP_LOG_FILE_FORMAT":                     "json",
		"RIN_APP_LOG_OUTPUT":                          "stdout",
		"RIN_APP_LOG_FILE_PATH":                       "./logs/app.log",
		"RIN_APP_LOG_MAX_SIZE":                        "100",
		"RIN_APP_LOG_MAX_BACKUPS":                     "7",
		"RIN_APP_LOG_MAX_AGE":                         "30",
		"RIN_APP_I18N_DEFAULT_LOCALE":                 "zh-CN",
		"RIN_APP_I18N_FALLBACK_LOCALE":                "zh-CN",
		"RIN_APP_I18N_SUPPORTED_LOCALES":              "zh-CN,en-US",
		"RIN_APP_EXECUTOR_ENABLED":                    "true",
		"RIN_APP_STORAGE_DRIVER":                      "local",
		"RIN_APP_STORAGE_LOCAL_FS_TYPE":               "basepath",
		"RIN_APP_STORAGE_LOCAL_BASE_PATH":             "./data/uploads",
		"RIN_APP_STORAGE_LOCAL_PUBLIC_URL":            "/uploads",
		"RIN_APP_STORAGE_LOCAL_ENABLE_WATCH":          "false",
		"RIN_APP_STORAGE_LOCAL_WATCH_BUFFER_SIZE":     "100",
		"RIN_APP_STORAGE_S3_ENDPOINT":                 "https://s3.example.com",
		"RIN_APP_STORAGE_S3_REGION":                   "us-east-1",
		"RIN_APP_STORAGE_S3_BUCKET":                   "aoi-admin",
		"RIN_APP_STORAGE_S3_ACCESS_KEY_ID":            "example",
		"RIN_APP_STORAGE_S3_SECRET_ACCESS_KEY":        "example",
		"RIN_APP_STORAGE_S3_USE_PATH_STYLE":           "true",
		"RIN_APP_STORAGE_S3_PUBLIC_BASE_URL":          "",
		"RIN_APP_STORAGE_MINIO_ENDPOINT":              "http://127.0.0.1:9000",
		"RIN_APP_STORAGE_MINIO_REGION":                "us-east-1",
		"RIN_APP_STORAGE_MINIO_BUCKET":                "aoi-admin",
		"RIN_APP_STORAGE_MINIO_ACCESS_KEY_ID":         "example",
		"RIN_APP_STORAGE_MINIO_SECRET_ACCESS_KEY":     "example",
		"RIN_APP_STORAGE_MINIO_USE_PATH_STYLE":        "true",
		"RIN_APP_STORAGE_MINIO_PUBLIC_BASE_URL":       "",
		"RIN_APP_SYSTEM_SEED_DEFAULTS_ON_START":       "true",
		"RIN_APP_WEBUI_ENABLED":                       "true",
		"RIN_APP_WEBUI_MOUNT_PATH":                    "/",
		"RIN_APP_WEBUI_DIST_DIR":                      "./web/app/build/client",
		"RIN_APP_WEBUI_PUBLIC_BASE_URL":               "/",
		"RIN_APP_RPC_ENABLED":                         "false",
		"RIN_APP_RPC_HOST":                            "127.0.0.1",
		"RIN_APP_RPC_PORT":                            "10099",
		"RIN_APP_RPC_READ_TIMEOUT":                    "10",
		"RIN_APP_RPC_WRITE_TIMEOUT":                   "10",
		"RIN_APP_RPC_IDLE_TIMEOUT":                    "30",
		"RIN_APP_PLUGINS_ENABLED":                     "false",
		"RIN_APP_PLUGINS_HEARTBEAT_TIMEOUT_SECONDS":   "30",
		"RIN_APP_PLUGINS_RPC_ENABLED":                 "false",
		"RIN_APP_AUTH_ENABLED":                        "true",
		"RIN_APP_AUTH_REGISTRATION_MODE":              "direct",
		"RIN_APP_AUTH_ISSUER":                         "aoi-admin",
		"RIN_APP_AUTH_AUDIENCE":                       "aoi-admin-api",
		"RIN_APP_AUTH_SIGNING_KEY":                    "example-signing-key-at-least-32-bytes",
		"RIN_APP_AUTH_ACCESS_TOKEN_TTL_SECONDS":       "900",
		"RIN_APP_AUTH_REFRESH_TOKEN_TTL_SECONDS":      "604800",
		"RIN_APP_AUTH_REFRESH_TOKEN_PEPPER":           "example-refresh-pepper-at-least-32",
		"RIN_APP_AUTH_MFA_ISSUER":                     "aoi-admin",
		"RIN_APP_AUTH_MFA_SECRET_KEY":                 "example-mfa-secret-key-at-least-32",
		"RIN_APP_AUTH_LOGIN_MAX_FAILURES":             "5",
		"RIN_APP_AUTH_LOGIN_LOCK_MINUTES":             "15",
		"RIN_APP_AUTH_LOGIN_CAPTCHA_ENABLED":          "false",
		"RIN_APP_AUTH_CAPTCHA_TTL_SECONDS":            "120",
		"RIN_APP_AUTH_INVITATION_TTL_SECONDS":         "86400",
		"RIN_APP_AUTH_EMAIL_VERIFICATION_TTL_SECONDS": "86400",
		"RIN_APP_AUTH_PASSWORD_RESET_TTL_SECONDS":     "1800",
		"RIN_APP_AUTH_NOTIFICATION_DRIVER":            "debug",
		"RIN_APP_AUTH_SMTP_HOST":                      "127.0.0.1",
		"RIN_APP_AUTH_SMTP_PORT":                      "1025",
		"RIN_APP_AUTH_SMTP_USERNAME":                  "mailer",
		"RIN_APP_AUTH_SMTP_PASSWORD":                  "example-smtp-password",
		"RIN_APP_AUTH_SMTP_FROM":                      "no-reply@example.invalid",
		"RIN_APP_AUTH_SMTP_FROM_NAME":                 "${BRAND_PRODUCT_NAME:Aoi Admin}",
		"RIN_APP_AUTH_SMTP_SECURITY":                  "none",
		"RIN_APP_AUTH_PASSWORD_MIN_LENGTH":            "8",
		"RIN_APP_AUTH_PASSWORD_REQUIRE_LOWER":         "false",
		"RIN_APP_AUTH_PASSWORD_REQUIRE_UPPER":         "false",
		"RIN_APP_AUTH_PASSWORD_REQUIRE_NUMBER":        "false",
		"RIN_APP_AUTH_PASSWORD_REQUIRE_SYMBOL":        "false",
		"RIN_APP_AUTH_CASBIN_RELOAD_INTERVAL_SECONDS": "300",
		"RIN_APP_MIGRATION_AUTO_APPLY":                "true",
		"RIN_APP_MIGRATION_DIR":                       "./internal/migrations",
		"RIN_APP_CORS_ENABLED":                        "true",
		"RIN_APP_CORS_ALLOW_ORIGINS":                  "*",
		"RIN_APP_CORS_ALLOW_METHODS":                  "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		"RIN_APP_CORS_ALLOW_HEADERS":                  "Origin,Content-Type,X-Request-ID,Authorization",
		"RIN_APP_CORS_EXPOSE_HEADERS":                 "X-Request-ID",
		"RIN_APP_CORS_ALLOW_CREDENTIALS":              "false",
		"RIN_APP_CORS_MAX_AGE":                        "3600",
		"AUTH_SIGNING_KEY":                            "example-signing-key-at-least-32-bytes",
		"AUTH_REFRESH_TOKEN_PEPPER":                   "example-refresh-pepper-at-least-32",
		"AUTH_MFA_SECRET_KEY":                         "example-mfa-secret-key-at-least-32",
		"AUTH_SMTP_HOST":                              "127.0.0.1",
		"AUTH_SMTP_PORT":                              "1025",
		"AUTH_SMTP_USERNAME":                          "mailer",
		"AUTH_SMTP_PASSWORD":                          "example-smtp-password",
		"AUTH_SMTP_FROM":                              "no-reply@example.invalid",
		"AUTH_SMTP_FROM_NAME":                         "${BRAND_PRODUCT_NAME:Aoi Admin}",
		"AUTH_SMTP_SECURITY":                          "none",
	}

	switch fileName {
	case "config.production.example.yaml", "postgres-production.example.yaml":
		env["RIN_APP_SERVER_MODE"] = "release"
		env["RIN_APP_DB_DRIVER"] = "postgres"
		env["RIN_APP_AUTH_REGISTRATION_MODE"] = "disabled"
		env["RIN_APP_AUTH_NOTIFICATION_DRIVER"] = "smtp"
		env["RIN_APP_AUTH_SMTP_SECURITY"] = "starttls"
		env["RIN_APP_MIGRATION_AUTO_APPLY"] = "false"
		env["RIN_APP_CORS_ALLOW_ORIGINS"] = "https://admin.example.invalid"
	case "mysql-redis.example.yaml":
		env["RIN_APP_DB_DRIVER"] = "mysql"
		env["RIN_APP_CACHE_DRIVER"] = "redis"
	case "smtp-auth.example.yaml":
		env["RIN_APP_AUTH_REGISTRATION_MODE"] = "disabled"
		env["RIN_APP_AUTH_NOTIFICATION_DRIVER"] = "smtp"
	case "storage-media.example.yaml":
		env["RIN_APP_STORAGE_DRIVER"] = "local"
	case "plugins-remote-rpc.example.yaml":
		env["RIN_APP_PLUGINS_ENABLED"] = "true"
		env["RIN_APP_RPC_ENABLED"] = "true"
		env["RIN_APP_PLUGINS_RPC_ENABLED"] = "true"
	}

	for key, value := range env {
		t.Setenv(key, value)
	}
}
