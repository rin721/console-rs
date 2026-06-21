// Package database 提供统一的数据库抽象层
// 支持 PostgreSQL、MySQL 和 SQLite,并提供连接池管理
// 设计目标:
// - 提供统一的接口,屏蔽不同数据库的差异
// - 支持连接池,提高性能和资源利用率
// - 便于切换数据库类型,无需修改业务代码
package database

// 本文件属于数据库基础设施层，定义连接工厂、池参数、事务上下文和热重载资源替换边界。

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ErrSQLDBUnavailable 表示底层标准库 SQL 连接不可用。
var ErrSQLDBUnavailable = errors.New("sql db unavailable")

var (
	// ErrNotFound 表示查询没有命中任何记录。
	ErrNotFound = errors.New("record not found")
)

// Driver 表示数据库驱动类型
// 使用字符串类型定义,便于配置文件指定
// 这是一个类型别名,提供类型安全
type Driver string

const (
	// DriverPostgres PostgreSQL 数据库驱动
	// PostgreSQL 的优点:
	// - 功能强大,支持复杂查询
	// - 完全符合 ACID
	// - 适合大型应用和复杂业务
	DriverPostgres Driver = "postgres"

	// DriverMySQL MySQL 数据库驱动
	// MySQL 的优点:
	// - 使用广泛,生态成熟
	// - 性能优秀
	// - 部署简单
	DriverMySQL Driver = "mysql"

	// DriverSQLite SQLite 数据库驱动
	// SQLite 的优点:
	// - 零配置,嵌入式数据库
	// - 适合开发和测试
	// - 单文件存储,便于备份
	// 注意:SQLite 不适合高并发生产环境
	DriverSQLite Driver = "sqlite"
)

// Config 保存数据库连接配置
// 包含了连接数据库所需的所有参数
// mapstructure tag 用于从配置文件(YAML/JSON)加载
type Config struct {
	// Driver 数据库驱动类型
	// 必须是 postgres、mysql 或 sqlite 之一
	Driver Driver `mapstructure:"driver"`

	// Host 数据库服务器地址
	// 例如: localhost, 127.0.0.1, db.example.com
	// SQLite 不需要此字段
	Host string `mapstructure:"host"`

	// Port 数据库端口
	// PostgreSQL 默认 5432
	// MySQL 默认 3306
	// SQLite 不需要此字段
	Port int `mapstructure:"port"`

	// User 数据库用户名
	// SQLite 不需要此字段
	User string `mapstructure:"user"`

	// Password 数据库密码
	// 生产环境应该从环境变量或密钥管理服务读取,不要硬编码
	// SQLite 不需要此字段
	Password string `mapstructure:"password"`

	// DBName 数据库名称
	// PostgreSQL/MySQL: 数据库名
	// SQLite: 文件路径(例如: ./data/app.db)
	DBName string `mapstructure:"dbname"`

	// SSLMode SSL 连接模式
	// PostgreSQL 使用: disable, require, verify-ca, verify-full
	// MySQL 使用: true, false, skip-verify, preferred
	// 生产环境建议启用 SSL 保护数据传输
	SSLMode string `mapstructure:"sslMode"`

	// MaxOpenConns 最大打开连接数
	// 控制同时活跃的数据库连接数量
	// 设置过大:消耗数据库资源
	// 设置过小:并发请求会等待连接
	// 推荐值:根据应用并发量调整,通常 10-100
	MaxOpenConns int `mapstructure:"maxOpenConns"`

	// MaxIdleConns 最大空闲连接数
	// 保持在连接池中的空闲连接数
	// 好处:避免频繁创建/销毁连接的开销
	// 建议:设置为 MaxOpenConns 的 50%-100%
	MaxIdleConns int `mapstructure:"maxIdleConns"`

	// MaxLifetime 连接最大生命周期
	// 连接使用超过此时间后会被关闭并重新创建
	// 用途:
	// - 避免使用过期的连接
	// - 定期刷新连接,防止数据库端超时
	// 推荐值: 5-30 分钟
	MaxLifetime time.Duration `mapstructure:"maxLifetime"`

	// Silent 控制 GORM 是否静默输出。
	// CLI/TUI bootstrap 阶段会启用该选项，避免 SQL 日志穿透交互界面。
	Silent bool `mapstructure:"-"`
}

// Reloader 定义数据库配置重载接口
// 允许在运行时动态更新数据库配置,无需重启应用
// 使用场景:
// - 配置文件热更新
// - 动态调整连接池参数
// - 切换数据库端点
// - 更新 SSL/TLS 配置
type Reloader interface {
	// Reload 使用新配置重新加载数据库连接
	// 步骤:
	// 1. 验证新配置的有效性
	// 2. 优雅关闭现有连接
	// 3. 使用新配置建立连接
	// 4. 重新配置连接池
	// 参数:
	//   cfg: 新的数据库配置
	// 返回:
	//   error: 重载失败时的错误(失败时保持原连接)
	Reload(cfg *Config) error
}

// Database 定义数据库操作的接口
// 这是一个抽象接口,隐藏了具体的数据库实现
// 好处:
// - 业务代码依赖接口而不是实现,符合依赖倒置原则
// - 便于 mock,易于单元测试
// - 可以轻松切换不同的数据库实现
type Database interface {
	Executor

	// Close 关闭数据库连接
	// 应该在应用关闭时调用,释放资源
	// 用途:
	// - 优雅关闭时释放数据库连接
	// - 防止连接泄漏
	// 返回:
	//   error: 关闭失败时的错误
	Close() error

	// Ping 验证数据库连接是否存活
	// 用途:
	// - 健康检查接口
	// - 初始化时验证配置是否正确
	// - 定期检查连接状态
	// 返回:
	//   error: 如果连接失败或不可用
	Ping(ctx context.Context) error

	// SQLDB 返回标准库数据库连接，供基础设施代码使用。
	//
	// 应用模块的常规数据访问仍应使用 Executor，避免绕过统一错误映射和事务入口。
	SQLDB() (*sql.DB, error)

	// WithTx 使用默认事务选项执行函数。
	WithTx(ctx context.Context, fn TxFunc) error

	// WithTxOptions 使用自定义事务选项执行函数。
	WithTxOptions(ctx context.Context, opts *TxOptions, fn TxFunc) error

	// Reloader 嵌入重载接口
	// 支持数据库配置的热更新
	Reloader
}

// Executor 定义内部层可使用的数据访问能力。
type Executor interface {
	Create(ctx context.Context, value any) error
	Save(ctx context.Context, value any) error
	First(ctx context.Context, dest any, opts ...QueryOption) error
	Find(ctx context.Context, dest any, opts ...QueryOption) error
	Update(ctx context.Context, model any, values map[string]any, opts ...QueryOption) (Result, error)
	Delete(ctx context.Context, model any, opts ...QueryOption) (Result, error)
	Exec(ctx context.Context, sql string, args ...any) (Result, error)
	Raw(ctx context.Context, dest any, sql string, args ...any) (Result, error)
	Count(ctx context.Context, model any, opts ...QueryOption) (int64, error)
	HasTable(ctx context.Context, model any) (bool, error)
}

// Query 保存通用查询条件，不暴露任何 ORM 特定类型。
type Query struct {
	Table       string
	Where       []Condition
	Order       string
	Limit       int
	Offset      int
	Unscoped    bool
	WithDeleted bool
}

// Condition 表示一条参数化查询条件。
type Condition struct {
	Expr string
	Args []any
}

// Result 表示写入或查询执行后的通用结果。
type Result struct {
	RowsAffected int64
}

// QueryOption 修改 Query。
type QueryOption func(*Query)

// Table 指定查询表名。
func Table(name string) QueryOption {
	return func(q *Query) {
		q.Table = name
	}
}

// Where 增加参数化查询条件。
func Where(expr string, args ...any) QueryOption {
	return func(q *Query) {
		q.Where = append(q.Where, Condition{Expr: expr, Args: args})
	}
}

// Order 指定排序表达式。
func Order(expr string) QueryOption {
	return func(q *Query) {
		q.Order = expr
	}
}

// Limit 指定最大返回条数。
func Limit(n int) QueryOption {
	return func(q *Query) {
		q.Limit = n
	}
}

// Offset 指定查询偏移量。
func Offset(n int) QueryOption {
	return func(q *Query) {
		q.Offset = n
	}
}

// TxFunc 是事务回调函数签名。
type TxFunc func(ctx context.Context, tx Executor) error

// TxOptions 保存事务执行选项。
type TxOptions struct {
	Isolation                sql.IsolationLevel
	ReadOnly                 bool
	Timeout                  time.Duration
	DisableNestedTransaction bool
}

// Hook 定义数据库操作的回调接口
// 这是一个扩展点,允许在数据库操作前后插入自定义逻辑
// 使用场景:
// - 记录审计日志
// - 性能监控
// - 数据验证
// - 自动填充字段
// 注意:
//
//	目前是接口定义,实际使用时需要实现具体的 Hook
type Hook interface {
	// BeforeCreate 在创建记录之前调用
	// 可以用于:
	// - 设置默认值
	// - 生成 ID
	// - 验证数据
	BeforeCreate(ctx context.Context, tx Executor)

	// AfterCreate 在创建记录之后调用
	// 可以用于:
	// - 记录审计日志
	// - 发送通知
	// - 更新缓存
	AfterCreate(ctx context.Context, tx Executor)

	// BeforeQuery 在查询之前调用
	// 可以用于:
	// - 添加通用查询条件(如租户隔离)
	// - 记录查询日志
	// - 性能监控开始
	BeforeQuery(ctx context.Context, tx Executor)

	// AfterQuery 在查询之后调用
	// 可以用于:
	// - 性能监控结束
	// - 结果处理
	// - 缓存更新
	AfterQuery(ctx context.Context, tx Executor)
}
