// Package db 提供数据库命令对 SQLGen 的桥接能力。
//
// 本包用于本地验证：它返回生成 SQL 以便测试和审计，但不是生产迁移框架。
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/rei0721/go-scaffold/pkg/database"
	"github.com/rei0721/go-scaffold/pkg/sqlgen"
)

var (
	// ErrUnsupportedDriver 表示当前 database driver 无法映射到 sqlgen 方言。
	ErrUnsupportedDriver = errors.New("unsupported database driver")
	// ErrMissingDatabase 表示调用方未注入数据库实例。
	ErrMissingDatabase = errors.New("database is nil")
)

// DialectForDriver 将应用数据库 driver 映射为 sqlgen 方言。
func DialectForDriver(driver string) (sqlgen.Dialect, error) {
	switch database.Driver(driver) {
	case database.DriverSQLite:
		return sqlgen.SQLite, nil
	case database.DriverMySQL:
		return sqlgen.MySQL, nil
	case database.DriverPostgres:
		return sqlgen.PostgreSQL, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedDriver, driver)
	}
}

// NewGenerator 创建数据库命令使用的 sqlgen 生成器。
func NewGenerator(driver string) (*sqlgen.Generator, error) {
	dialect, err := DialectForDriver(driver)
	if err != nil {
		return nil, err
	}
	return sqlgen.New(&sqlgen.Config{
		Dialect:       dialect,
		SoftDelete:    true,
		SkipZeroValue: true,
	}), nil
}

// DatabaseSQL 生成创建数据库的 SQL。
func DatabaseSQL(driver, name string) (string, error) {
	gen, err := NewGenerator(driver)
	if err != nil {
		return "", err
	}
	return gen.DatabaseIfNotExists(name)
}

// ApplyDatabase 执行创建数据库 SQL，并返回实际生成的语句。
//
// 返回 SQL 是为了让测试和运维日志能确认 sqlgen 生成结果，而不是隐藏在副作用里。
func ApplyDatabase(ctx context.Context, db database.Database, driver, name string) (string, error) {
	if db == nil {
		return "", ErrMissingDatabase
	}
	sql, err := DatabaseSQL(driver, name)
	if err != nil {
		return "", err
	}
	if _, err := db.Exec(ctx, sql); err != nil {
		return sql, fmt.Errorf("apply database create: %w", err)
	}
	return sql, nil
}
