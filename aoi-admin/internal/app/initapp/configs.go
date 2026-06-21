package initapp

import (
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cache"
	"github.com/rei0721/go-scaffold/pkg/database"
	"github.com/rei0721/go-scaffold/pkg/executor"
	"github.com/rei0721/go-scaffold/pkg/httpserver"
	"github.com/rei0721/go-scaffold/pkg/logger"
	mailpkg "github.com/rei0721/go-scaffold/pkg/mail"
	"github.com/rei0721/go-scaffold/pkg/rpcserver"
	"github.com/rei0721/go-scaffold/pkg/storage"
)

func IsCacheConfigChanged(oldCfg, newCfg *config.Config) bool {
	if oldCfg == newCfg {
		return false
	}
	return oldCfg.Cache != newCfg.Cache
}

func IsDatabaseConfigChanged(oldCfg, newCfg *config.Config) bool {
	if oldCfg == newCfg {
		return false
	}
	return oldCfg.Database != newCfg.Database
}

func IsServerConfigChanged(oldCfg, newCfg *config.Config) bool {
	if oldCfg == newCfg {
		return false
	}
	return oldCfg.Server != newCfg.Server
}

func IsRPCConfigChanged(oldCfg, newCfg *config.Config) bool {
	if oldCfg == newCfg {
		return false
	}
	return oldCfg.RPC != newCfg.RPC
}

func IsLoggerConfigChanged(oldCfg, newCfg *config.Config) bool {
	if oldCfg == newCfg {
		return false
	}
	return oldCfg.Logger != newCfg.Logger
}

func IsExecutorConfigChanged(oldCfg, newCfg *config.Config) bool {
	if oldCfg == newCfg {
		return false
	}
	return oldCfg.Executor.Enabled != newCfg.Executor.Enabled ||
		!reflect.DeepEqual(oldCfg.Executor.Pools, newCfg.Executor.Pools)
}

func IsStorageConfigChanged(oldCfg, newCfg *config.Config) bool {
	if oldCfg == newCfg {
		return false
	}
	return oldCfg.Storage != newCfg.Storage
}

func IsIAMRuntimeConfigChanged(oldCfg, newCfg *config.Config) bool {
	if oldCfg == newCfg {
		return false
	}
	return oldCfg.Auth.RegistrationMode != newCfg.Auth.RegistrationMode ||
		oldCfg.Auth.EmailVerificationTTLSeconds != newCfg.Auth.EmailVerificationTTLSeconds ||
		oldCfg.Auth.NotificationDriver != newCfg.Auth.NotificationDriver ||
		oldCfg.Auth.SMTP != newCfg.Auth.SMTP ||
		oldCfg.WebUI.PublicBaseURL != newCfg.WebUI.PublicBaseURL ||
		oldCfg.I18n.DefaultLocale != newCfg.I18n.DefaultLocale ||
		!reflect.DeepEqual(oldCfg.I18n.Resources, newCfg.I18n.Resources) ||
		oldCfg.Brand != newCfg.Brand
}

func CacheRuntimeConfig(cfg *config.Config) cache.FactoryConfig {
	localTTL := time.Duration(cfg.Cache.Local.DefaultTTLSeconds) * time.Second
	return cache.FactoryConfig{
		Driver: cache.Driver(cfg.Cache.Driver),
		Local: cache.LocalConfig{
			MaxCost:     cfg.Cache.Local.MaxCost,
			NumCounters: cfg.Cache.Local.NumCounters,
			BufferItems: cfg.Cache.Local.BufferItems,
			DefaultTTL:  localTTL,
		},
		Redis: RedisConfig(cfg.Cache.Redis),
	}
}

func RedisConfig(cfg config.RedisCacheConfig) *cache.Config {
	defaults := cache.DefaultConfig()
	host, port := splitRedisAddr(cfg.Addr)
	defaults.Host = host
	defaults.Port = port
	defaults.Username = cfg.Username
	defaults.Password = cfg.Password
	defaults.DB = cfg.DB
	if cfg.PoolSize > 0 {
		defaults.PoolSize = cfg.PoolSize
	}
	if cfg.MinIdleConns > 0 {
		defaults.MinIdleConns = cfg.MinIdleConns
	}
	if cfg.MaxRetries > 0 {
		defaults.MaxRetries = cfg.MaxRetries
	}
	if cfg.DialTimeout > 0 {
		defaults.DialTimeout = time.Duration(cfg.DialTimeout) * time.Second
	}
	if cfg.ReadTimeout > 0 {
		defaults.ReadTimeout = time.Duration(cfg.ReadTimeout) * time.Second
	}
	if cfg.WriteTimeout > 0 {
		defaults.WriteTimeout = time.Duration(cfg.WriteTimeout) * time.Second
	}
	return defaults
}

func splitRedisAddr(addr string) (string, int) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", 0
	}
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		parts := strings.Split(addr, ":")
		if len(parts) != 2 {
			return addr, cache.DefaultPort
		}
		host = parts[0]
		portText = parts[1]
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return host, 0
	}
	return host, port
}

func DatabaseConfig(cfg *config.Config) *database.Config {
	db := cfg.Database
	out := &database.Config{
		Driver:       database.Driver(db.Driver),
		MaxOpenConns: db.Pool.MaxOpenConns,
		MaxIdleConns: db.Pool.MaxIdleConns,
	}
	switch db.Driver {
	case config.DatabaseDriverSQLite:
		out.DBName = db.SQLite.Path
	case config.DatabaseDriverMySQL:
		out.Host = db.MySQL.Host
		out.Port = db.MySQL.Port
		out.User = db.MySQL.Username
		out.Password = db.MySQL.Password
		out.DBName = db.MySQL.Database
	case config.DatabaseDriverPostgres:
		out.Host = db.Postgres.Host
		out.Port = db.Postgres.Port
		out.User = db.Postgres.Username
		out.Password = db.Postgres.Password
		out.DBName = db.Postgres.Database
		out.SSLMode = db.Postgres.SSLMode
	}
	return out
}

func DatabaseSummary(cfg *config.Config) string {
	switch cfg.Database.Driver {
	case config.DatabaseDriverSQLite:
		return cfg.Database.SQLite.Path
	case config.DatabaseDriverMySQL:
		return cfg.Database.MySQL.Database
	case config.DatabaseDriverPostgres:
		return cfg.Database.Postgres.Database
	default:
		return ""
	}
}

func DatabaseHostPort(cfg *config.Config) (string, int) {
	switch cfg.Database.Driver {
	case config.DatabaseDriverMySQL:
		return cfg.Database.MySQL.Host, cfg.Database.MySQL.Port
	case config.DatabaseDriverPostgres:
		return cfg.Database.Postgres.Host, cfg.Database.Postgres.Port
	default:
		return "", 0
	}
}

func DatabaseUsername(cfg *config.Config) string {
	switch cfg.Database.Driver {
	case config.DatabaseDriverMySQL:
		return cfg.Database.MySQL.Username
	case config.DatabaseDriverPostgres:
		return cfg.Database.Postgres.Username
	default:
		return ""
	}
}

func DatabasePassword(cfg *config.Config) string {
	switch cfg.Database.Driver {
	case config.DatabaseDriverMySQL:
		return cfg.Database.MySQL.Password
	case config.DatabaseDriverPostgres:
		return cfg.Database.Postgres.Password
	default:
		return ""
	}
}

func LoggerConfig(cfg *config.Config) *logger.Config {
	return &logger.Config{
		Level:         cfg.Logger.Level,
		Format:        cfg.Logger.Format,
		ConsoleFormat: cfg.Logger.ConsoleFormat,
		FileFormat:    cfg.Logger.FileFormat,
		Output:        cfg.Logger.Output,
		FilePath:      cfg.Logger.FilePath,
		MaxSize:       cfg.Logger.MaxSize,
		MaxBackups:    cfg.Logger.MaxBackups,
		MaxAge:        cfg.Logger.MaxAge,
	}
}

func ExecutorConfigs(cfg *config.Config) []executor.Config {
	configs := make([]executor.Config, 0, len(cfg.Executor.Pools))
	for _, poolCfg := range cfg.Executor.Pools {
		configs = append(configs, executor.Config{
			Name:        executor.PoolName(poolCfg.Name),
			Size:        poolCfg.Size,
			Expiry:      time.Duration(poolCfg.Expiry) * time.Second,
			NonBlocking: poolCfg.NonBlocking,
		})
	}
	return configs
}

func HTTPServerConfig(cfg *config.Config) *httpserver.Config {
	return &httpserver.Config{
		Host:         cfg.Server.Host,
		Port:         cfg.Server.Port,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}
}

func RPCServerConfig(cfg *config.Config) *rpcserver.Config {
	rpcCfg := cfg.RPC
	rpcCfg.ApplyDefaults()
	return &rpcserver.Config{
		Enabled:      rpcCfg.Enabled,
		Host:         rpcCfg.Host,
		Port:         rpcCfg.Port,
		ReadTimeout:  time.Duration(rpcCfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(rpcCfg.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(rpcCfg.IdleTimeout) * time.Second,
	}
}

func StorageConfig(cfg *config.Config) *storage.Config {
	return cfg.Storage.ToPkgConfig()
}

func MailConfig(cfg config.SMTPConfig) mailpkg.Config {
	return mailpkg.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Username: cfg.Username,
		Password: cfg.Password,
		From:     cfg.From,
		FromName: cfg.FromName,
		Security: cfg.Security,
	}
}
