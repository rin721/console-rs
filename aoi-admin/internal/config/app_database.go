package config

import (
	"errors"
	"fmt"
	"strings"
)

const (
	DatabaseDriverSQLite   = "sqlite"
	DatabaseDriverMySQL    = "mysql"
	DatabaseDriverPostgres = "postgres"
)

type DatabaseConfig struct {
	Driver   string                 `mapstructure:"driver" envname:"DB_DRIVER"`
	SQLite   DatabaseSQLiteConfig   `mapstructure:"sqlite"`
	MySQL    DatabaseMySQLConfig    `mapstructure:"mysql"`
	Postgres DatabasePostgresConfig `mapstructure:"postgres"`
	Pool     DatabasePoolConfig     `mapstructure:"pool"`
}

type DatabaseSQLiteConfig struct {
	Path string `mapstructure:"path" envname:"DB_SQLITE_PATH"`
}

type DatabaseMySQLConfig struct {
	Host     string `mapstructure:"host" envname:"DB_MYSQL_HOST"`
	Port     int    `mapstructure:"port" envname:"DB_MYSQL_PORT"`
	Username string `mapstructure:"username" envname:"DB_MYSQL_USERNAME"`
	Password string `mapstructure:"password" envname:"DB_MYSQL_PASSWORD"`
	Database string `mapstructure:"database" envname:"DB_MYSQL_DATABASE"`
	Charset  string `mapstructure:"charset" envname:"DB_MYSQL_CHARSET"`
}

type DatabasePostgresConfig struct {
	Host     string `mapstructure:"host" envname:"DB_POSTGRES_HOST"`
	Port     int    `mapstructure:"port" envname:"DB_POSTGRES_PORT"`
	Username string `mapstructure:"username" envname:"DB_POSTGRES_USERNAME"`
	Password string `mapstructure:"password" envname:"DB_POSTGRES_PASSWORD"`
	Database string `mapstructure:"database" envname:"DB_POSTGRES_DATABASE"`
	SSLMode  string `mapstructure:"sslMode" envname:"DB_POSTGRES_SSL_MODE"`
}

type DatabasePoolConfig struct {
	MaxOpenConns int `mapstructure:"maxOpenConns" envname:"DB_POOL_MAX_OPEN_CONNS"`
	MaxIdleConns int `mapstructure:"maxIdleConns" envname:"DB_POOL_MAX_IDLE_CONNS"`
}

func (c *DatabaseConfig) ValidateName() string {
	return AppDatabaseName
}

func (c *DatabaseConfig) ValidateRequired() bool {
	return true
}

func (c *DatabaseConfig) Validate() error {
	c.Driver = strings.ToLower(strings.TrimSpace(c.Driver))
	switch c.Driver {
	case DatabaseDriverSQLite:
		if strings.TrimSpace(c.SQLite.Path) == "" {
			return errors.New("sqlite.path is required")
		}
	case DatabaseDriverMySQL:
		if err := validateNetworkDatabase(c.MySQL.Host, c.MySQL.Port, c.MySQL.Username, c.MySQL.Database, "mysql"); err != nil {
			return err
		}
	case DatabaseDriverPostgres:
		if err := validateNetworkDatabase(c.Postgres.Host, c.Postgres.Port, c.Postgres.Username, c.Postgres.Database, "postgres"); err != nil {
			return err
		}
	default:
		return errors.New("driver must be sqlite, mysql, or postgres")
	}
	if c.Pool.MaxOpenConns < 0 {
		return errors.New("pool.maxOpenConns must be non-negative")
	}
	if c.Pool.MaxIdleConns < 0 {
		return errors.New("pool.maxIdleConns must be non-negative")
	}
	return nil
}

func validateNetworkDatabase(host string, port int, username string, databaseName string, driver string) error {
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("%s.host is required", driver)
	}
	if port <= 0 || port > 65535 {
		return fmt.Errorf("%s.port must be between 1 and 65535", driver)
	}
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("%s.username is required", driver)
	}
	if strings.TrimSpace(databaseName) == "" {
		return fmt.Errorf("%s.database is required", driver)
	}
	return nil
}

func (cfg *DatabaseConfig) overrideDatabaseConfig() {
	overrideConfigFromEnv(cfg)
}

func overrideDatabaseConfig(cfg *DatabaseConfig) {
	overrideConfigFromEnv(cfg)
}
