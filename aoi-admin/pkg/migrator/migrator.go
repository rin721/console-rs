// Package migrator 将 goose migration 封装在项目自有 API 后面，统一数据库迁移入口。
package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"sync"

	"github.com/pressly/goose/v3"
	"github.com/rei0721/go-scaffold/pkg/database"
)

// DefaultDir 是未显式配置迁移目录时使用的仓库内默认位置。
const DefaultDir = "internal/migrations"

// SQLProvider 提供底层 *sql.DB，避免迁移包直接依赖具体数据库管理器实现。
type SQLProvider interface {
	SQLDB() (*sql.DB, error)
}

// Config 描述迁移运行所需的数据库驱动和迁移文件目录。
type Config struct {
	Driver string
	Dir    string
}

// Runner 定义数据库迁移的最小操作集合。
type Runner interface {
	// Up 执行所有尚未应用的迁移。
	Up(context.Context) error
	// Down 回滚最近一次已应用迁移。
	Down(context.Context) error
	// Status 将迁移状态写入指定 writer；writer 为空时丢弃 goose 输出。
	Status(context.Context, io.Writer) error
}

type runner struct {
	db  *sql.DB
	cfg Config
}

var gooseMu sync.Mutex

// New 创建迁移 Runner，并在构造阶段解析数据库连接以尽早暴露配置错误。
func New(provider SQLProvider, cfg Config) (Runner, error) {
	if provider == nil {
		return nil, fmt.Errorf("migrator database provider is nil")
	}
	db, err := provider.SQLDB()
	if err != nil {
		return nil, err
	}
	if cfg.Dir == "" {
		cfg.Dir = DefaultDir
	}
	if cfg.Driver == "" {
		return nil, fmt.Errorf("migrator driver is required")
	}
	return &runner{db: db, cfg: cfg}, nil
}

// Up 在互斥保护下执行 goose Up，避免全局 goose 配置被并发调用相互覆盖。
func (r *runner) Up(ctx context.Context) error {
	return r.run(ctx, nil, func(ctx context.Context) error {
		return goose.UpContext(ctx, r.db, r.cfg.Dir, goose.WithNoColor(true))
	})
}

// Down 在互斥保护下回滚最近一次迁移。
func (r *runner) Down(ctx context.Context) error {
	return r.run(ctx, nil, func(ctx context.Context) error {
		return goose.DownContext(ctx, r.db, r.cfg.Dir, goose.WithNoColor(true))
	})
}

// Status 输出当前迁移状态，通常用于 CLI 诊断或部署前检查。
func (r *runner) Status(ctx context.Context, w io.Writer) error {
	return r.run(ctx, w, func(ctx context.Context) error {
		return goose.StatusContext(ctx, r.db, r.cfg.Dir, goose.WithNoColor(true))
	})
}

// run 统一设置 goose 全局方言和 logger；goose 的全局状态要求这里串行执行。
func (r *runner) run(ctx context.Context, w io.Writer, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	gooseMu.Lock()
	defer gooseMu.Unlock()
	if err := goose.SetDialect(gooseDialect(r.cfg.Driver)); err != nil {
		return err
	}
	if w != nil {
		goose.SetLogger(writerLogger{w: w})
	} else {
		goose.SetLogger(writerLogger{w: io.Discard})
	}
	return fn(ctx)
}

// gooseDialect 将项目数据库驱动名映射到 goose 期望的方言名。
func gooseDialect(driver string) string {
	switch driver {
	case string(database.DriverSQLite), "sqlite3":
		return "sqlite3"
	case string(database.DriverPostgres), "postgresql":
		return "postgres"
	default:
		return driver
	}
}

// writerLogger 将 goose 日志重定向到调用方提供的 writer，便于 CLI 捕获和测试断言。
type writerLogger struct {
	w io.Writer
}

func (l writerLogger) Fatalf(format string, v ...interface{}) {
	_, _ = fmt.Fprintf(l.w, format+"\n", v...)
}

func (l writerLogger) Printf(format string, v ...interface{}) {
	_, _ = fmt.Fprintf(l.w, format+"\n", v...)
}
