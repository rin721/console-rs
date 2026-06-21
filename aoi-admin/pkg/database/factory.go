package database

// 本文件属于数据库基础设施层，定义连接工厂、池参数、事务上下文和热重载资源替换边界。

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// database 实现 Database 接口
// 这是数据库抽象层的具体实现
// 设计考虑:
// - 同时持有 GORM 和标准库的 sql.DB
// - GORM 用于 ORM 操作
// - sql.DB 用于连接池管理和健康检查
// - 使用 RWMutex 保证并发安全
type database struct {
	// mu 读写锁,保护 db 和 sqlDB 字段的并发访问
	// 使用 RWMutex 而不是 Mutex:
	// - 读操作(DB, Ping)使用读锁,允许并发读取
	// - 写操作(Reload)使用写锁,独占访问
	mu sync.RWMutex

	// db GORM 数据库实例
	// 用于执行所有 ORM 操作(查询、创建、更新、删除)
	// 必须在持有锁的情况下访问
	db *gorm.DB

	// sqlDB 标准库的 sql.DB 实例
	// 用于:
	// - 配置连接池参数
	// - 执行 Ping 健康检查
	// - 关闭数据库连接
	// 必须在持有锁的情况下访问
	sqlDB *sql.DB
}

func (d *database) gormDB() *gorm.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

// Close 关闭数据库连接
// 实现 Database 接口
// 使用场景:
// - 应用优雅关闭时
// - 测试结束后清理资源
// 注意:
//
//	调用后数据库实例不可再使用
func (d *database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.sqlDB != nil {
		// 关闭底层的 sql.DB
		// 这会关闭所有连接池中的连接
		return d.sqlDB.Close()
	}
	return nil
}

// Ping 验证数据库连接是否存活
// 实现 Database 接口
// 使用场景:
// - 健康检查接口
// - 启动时验证数据库连接
// - 定期检查连接状态
// 返回:
//
//	error: 如果连接失败或超时
func (d *database) Ping(ctx context.Context) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.sqlDB != nil {
		// 执行 ping 操作
		// 会建立一个测试连接并立即关闭
		if ctx == nil {
			ctx = context.Background()
		}
		return d.sqlDB.PingContext(ctx)
	}
	return nil
}

// Reload 使用新配置重新加载数据库连接
// 实现 Reloader 接口
// 这个方法允许在运行时热更新数据库配置,无需重启应用
// 使用场景:
// - 配置文件变更时自动重载
// - 动态调整连接池参数
// - 切换数据库端点
// 参数:
//
//	cfg: 新的数据库配置
//
// 返回:
//
//	error: 重载失败时的错误
//
// 并发安全:
// - 使用写锁保护,确保重载过程原子性
// - 失败时保持原有连接不变
// - 新连接建立成功后才关闭旧连接
// - 新连接验证完成前不替换旧连接；真正的切换窗口由调用方的连接复用和数据库驱动行为决定
func (d *database) SQLDB() (*sql.DB, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.sqlDB == nil {
		return nil, ErrSQLDBUnavailable
	}
	return d.sqlDB, nil
}

func (d *database) Reload(cfg *Config) error {
	// 1. 使用新配置创建新的数据库实例
	// 在锁外创建,避免长时间持有锁
	// 不带 hooks,因为 hooks 在初始化时已注册
	// 如果需要重新注册 hooks,可以扩展此方法接受 hooks 参数
	newDB, err := New(cfg)
	if err != nil {
		// 新连接创建失败,保持原连接不变
		return fmt.Errorf("failed to create new database connection: %w", err)
	}

	// 2. 验证新连接是否可用
	// 执行 Ping 测试,确保新连接确实可用
	if err := newDB.Ping(context.Background()); err != nil {
		// 新连接不可用,关闭它并返回错误
		_ = newDB.Close()
		return fmt.Errorf("new database connection ping failed: %w", err)
	}

	// 3. 获取写锁,开始原子替换操作
	// 写锁确保:
	// - 没有其他 goroutine 正在读取 db/sqlDB
	// - 没有其他 goroutine 正在执行 Reload
	d.mu.Lock()

	// 保存旧连接的引用,用于后续关闭
	oldSQLDB := d.sqlDB

	// 4. 原子地替换数据库实例
	// 将新连接的内部字段复制到当前实例
	// 这样外部持有的 Database 接口引用仍然有效
	newDBImpl := newDB.(*database)
	d.db = newDBImpl.db
	d.sqlDB = newDBImpl.sqlDB

	// 5. 释放写锁
	// 新连接已替换完成,其他 goroutine 可以使用新连接
	d.mu.Unlock()

	// 6. 优雅关闭旧连接
	// 在锁外关闭,避免长时间持有锁
	// 这会关闭所有旧连接池中的连接
	// 注意: 可能仍有进行中的查询使用旧连接,但 sql.DB 会处理这种情况
	if oldSQLDB != nil {
		if err := oldSQLDB.Close(); err != nil {
			// 旧连接关闭失败,只记录错误
			// 不影响新连接的使用
			return fmt.Errorf("warning: failed to close old database connection: %w", err)
		}
	}

	return nil
}

// New 根据提供的配置创建一个新的 Database 实例
// 这是主要的工厂函数,用于创建数据库连接
// 参数:
//
//	cfg: 数据库配置
//
// 返回:
//
//	Database: 数据库接口
//	error: 创建失败时的错误
//
// 使用示例:
//
//	db, err := database.New(config)
func New(cfg *Config) (Database, error) {
	// 委托给 NewWithHooks,不带任何 hooks
	return NewWithHooks(cfg)
}

// NewWithHooks 创建一个带可选 hooks 的 Database 实例
// 这是一个高级工厂函数,支持注册 GORM 回调
// 参数:
//
//	cfg: 数据库配置
//	hooks: 可选的数据库钩子,用于在操作前后执行自定义逻辑
//
// 返回:
//
//	Database: 数据库接口
//	error: 创建失败时的错误
//
// Hooks 使用场景:
//   - 审计日志:记录所有数据变更
//   - 性能监控:统计查询时间
//   - 数据验证:在保存前验证数据
//   - 自动填充:自动设置创建时间等字段
func NewWithHooks(cfg *Config, hooks ...Hook) (Database, error) {
	var dialector gorm.Dialector

	// 1. 根据数据库驱动类型选择对应的 dialector
	// Dialector 是 GORM 的数据库方言,处理特定数据库的 SQL 语法
	switch cfg.Driver {
	case DriverPostgres:
		// PostgreSQL 数据库
		// 构建 PostgreSQL 专用的 DSN(数据源名称)
		dsn := buildPostgresDSN(cfg)
		dialector = postgres.Open(dsn)

	case DriverMySQL:
		// MySQL/MariaDB 数据库
		// 构建 MySQL 专用的 DSN
		dsn := buildMySQLDSN(cfg)
		dialector = mysql.Open(dsn)

	case DriverSQLite:
		// SQLite 嵌入式数据库
		// SQLite 只需要文件路径,不需要复杂的连接字符串
		// cfg.DBName 此时是数据库文件路径(如 ./data/app.db)
		if err := ensureSQLiteDir(cfg.DBName); err != nil {
			return nil, err
		}
		dialector = sqlite.Open(cfg.DBName)

	default:
		// 不支持的数据库驱动
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	// 2. 配置 GORM
	// GORM 配置可以设置:
	// - Logger: 日志记录器
	// - NamingStrategy: 命名策略(表名、列名转换)
	// - NowFunc: 自定义时间函数
	// - DryRun: 模拟运行,不实际执行 SQL
	// 这里使用空配置,采用 GORM 默认值
	gormCfg := &gorm.Config{}
	if cfg.Silent {
		gormCfg.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	}

	// 3. 打开数据库连接
	// gorm.Open 会:
	// - 建立数据库连接
	// - 初始化 GORM 实例
	// - 应用配置
	db, err := gorm.Open(dialector, gormCfg)
	if err != nil {
		// 连接失败,可能原因:
		// - 数据库服务未启动
		// - 凭证错误
		// - 网络问题
		// - 数据库不存在
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 4. 获取底层的 sql.DB
	// GORM 内部使用标准库的 sql.DB
	// 我们需要它来配置连接池和执行 Ping
	sqlDB, err := db.DB()
	if err != nil {
		// 获取失败(极少发生)
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 5. 配置连接池
	// 连接池参数影响性能和资源使用
	configureConnectionPool(sqlDB, cfg)

	// 6. 注册 hooks(如果提供)
	// Hooks 允许在 GORM 操作前后执行自定义逻辑
	// 例如:审计、日志、验证等
	if len(hooks) > 0 {
		registerHooks(db, hooks)
	}

	// 7. 返回数据库实例
	return &database{
		db:    db,    // GORM 实例
		sqlDB: sqlDB, // 标准库 sql.DB
	}, nil
}

// ensureSQLiteDir 在打开 SQLite 前创建父目录，避免本地文件数据库因目录缺失启动失败。
func ensureSQLiteDir(dbName string) error {
	if dbName == "" || dbName == ":memory:" || strings.HasPrefix(dbName, "file:") {
		return nil
	}
	dir := filepath.Dir(dbName)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create sqlite database directory: %w", err)
	}
	return nil
}

// buildPostgresDSN 构建 PostgreSQL 连接字符串
// DSN(Data Source Name)包含所有连接信息
// 参数:
//
//	cfg: 数据库配置
//
// 返回:
//
//	string: PostgreSQL DSN 字符串。
//
// DSN 格式:
//
//	host=localhost port=5432 user=postgres password=secret dbname=mydb sslmode=disable
func buildPostgresDSN(cfg *Config) string {
	// 处理 SSL 模式
	sslMode := cfg.SSLMode
	if sslMode == "" {
		// 默认禁用 SSL
		// 开发环境通常不需要 SSL
		// 生产环境应该启用(require, verify-ca, verify-full)
		sslMode = "disable"
	}

	// 构建连接字符串
	// 使用空格分隔的键值对格式
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,     // 主机地址
		cfg.Port,     // 端口(默认 5432)
		cfg.User,     // 用户名
		cfg.Password, // 密码
		cfg.DBName,   // 数据库名
		sslMode,      // SSL 模式
	)
}

// buildMySQLDSN 构建 MySQL 连接字符串
// 参数:
//
//	cfg: 数据库配置
//
// 返回:
//
//	string: MySQL DSN 字符串。
//
// DSN 格式:
//
//	username:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local
func buildMySQLDSN(cfg *Config) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User,     // 用户名
		cfg.Password, // 密码
		cfg.Host,     // 主机地址
		cfg.Port,     // 端口(默认 3306)
		cfg.DBName,   // 数据库名
		// 查询参数:
		// charset=utf8mb4: 使用 UTF-8 编码,支持 emoji
		// parseTime=True: 自动解析 DATE 和 DATETIME 为 time.Time
		// loc=Local: 使用本地时区
	)
}

// configureConnectionPool 设置连接池参数
// 连接池管理数据库连接的复用,提高性能
// 参数:
//
//	sqlDB: 标准库的 sql.DB 实例
//	cfg: 数据库配置
//
// 连接池参数说明:
//
//	MaxOpenConns: 最大打开连接数
//	MaxIdleConns: 最大空闲连接数
//	ConnMaxLifetime: 连接最大生命周期
func configureConnectionPool(sqlDB *sql.DB, cfg *Config) {
	// 1. 设置最大打开连接数
	if cfg.MaxOpenConns > 0 {
		// MaxOpenConns 限制同时活跃的连接数
		// 设置过大:消耗数据库资源
		// 设置过小:请求会阻塞等待连接
		// 推荐值:10-100,根据并发量调整
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}

	// 2. 设置最大空闲连接数
	if cfg.MaxIdleConns > 0 {
		// MaxIdleConns 控制连接池中的空闲连接数
		// 保持空闲连接可以:
		// - 避免频繁创建/销毁连接
		// - 提高响应速度
		// 建议:设置为 MaxOpenConns 的 50%-100%
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}

	// 3. 设置连接最大生命周期
	if cfg.MaxLifetime > 0 {
		// ConnMaxLifetime 限制连接的使用时间
		// 超过此时间的连接会被关闭并重新创建
		// 好处:
		// - 避免使用过期的连接
		// - 防止数据库端超时
		// - 定期刷新连接
		sqlDB.SetConnMaxLifetime(cfg.MaxLifetime)
	} else {
		// 默认最大生命周期 1 小时
		// 这是一个安全的默认值
		// 防止连接长时间不刷新
		sqlDB.SetConnMaxLifetime(time.Hour)
	}
}

// registerHooks 注册 GORM 回调钩子
// Hooks 允许在数据库操作的特定时机执行自定义代码
// 参数:
//
//	db: GORM 数据库实例
//	hooks: Hook 接口数组
//
// GORM 回调时机:
//
//	Before Create: 创建记录之前
//	After Create: 创建记录之后
//	Before Query: 查询之前
//	After Query: 查询之后
//	(还有 Update, Delete 等,这里未实现)
func registerHooks(db *gorm.DB, hooks []Hook) {
	// 遍历所有 hooks
	for _, hook := range hooks {
		// 捕获循环变量,避免闭包问题
		// 在 Go 1.22 之前,循环变量会被所有闭包共享
		// 这里显式捕获确保每个回调使用正确的 hook
		h := hook

		// 注册 "创建之前" 回调
		// Before("gorm:create"): 在 GORM 的 create 操作之前执行
		// Register: 注册一个命名的回调函数
		db.Callback().Create().Before("gorm:create").Register("hook:before_create", func(tx *gorm.DB) {
			h.BeforeCreate(tx.Statement.Context, &gormExecutor{db: tx})
		})

		// 注册 "创建之后" 回调
		// After("gorm:create"): 在 GORM 的 create 操作之后执行
		db.Callback().Create().After("gorm:create").Register("hook:after_create", func(tx *gorm.DB) {
			h.AfterCreate(tx.Statement.Context, &gormExecutor{db: tx})
		})

		// 注册 "查询之前" 回调
		// 可用于添加通用查询条件(如租户隔离)
		db.Callback().Query().Before("gorm:query").Register("hook:before_query", func(tx *gorm.DB) {
			h.BeforeQuery(tx.Statement.Context, &gormExecutor{db: tx})
		})

		// 注册 "查询之后" 回调
		// 可用于结果处理、缓存更新等
		db.Callback().Query().After("gorm:query").Register("hook:after_query", func(tx *gorm.DB) {
			h.AfterQuery(tx.Statement.Context, &gormExecutor{db: tx})
		})
	}
}
