package ports

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	// ErrNotFound 统一表示数据记录不存在。
	ErrNotFound = errors.New("record not found")
	// ErrNilTxFunc 表示事务回调为空。
	ErrNilTxFunc = errors.New("transaction function is nil")
	// ErrInvalidTxOptions 表示事务选项不合法。
	ErrInvalidTxOptions = errors.New("invalid transaction options")
	// ErrNestedTransactionDisabled 表示当前上下文不允许开启嵌套事务。
	ErrNestedTransactionDisabled = errors.New("nested transaction is disabled")
)

// Database 是应用层使用的数据库端口，组合通用执行器和生命周期能力。
type Database interface {
	Executor
	Close() error
	Ping(context.Context) error
	SQLDB() (*sql.DB, error)
	WithTx(context.Context, TxFunc) error
	WithTxOptions(context.Context, *TxOptions, TxFunc) error
}

// Executor 描述仓储层所需的最小数据访问操作集合。
type Executor interface {
	Create(context.Context, any) error
	Save(context.Context, any) error
	First(context.Context, any, ...QueryOption) error
	Find(context.Context, any, ...QueryOption) error
	Update(context.Context, any, map[string]any, ...QueryOption) (Result, error)
	Delete(context.Context, any, ...QueryOption) (Result, error)
	Exec(context.Context, string, ...any) (Result, error)
	Raw(context.Context, any, string, ...any) (Result, error)
	Count(context.Context, any, ...QueryOption) (int64, error)
	HasTable(context.Context, any) (bool, error)
}

// Result 表示写操作或原始 SQL 执行后的通用结果。
type Result struct {
	RowsAffected int64
}

// Query 保存跨 ORM 的通用查询条件。
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

// QueryOption 修改 Query，供调用方用函数式选项组合查询条件。
type QueryOption func(*Query)

// Table 指定查询表名。
func Table(name string) QueryOption {
	return func(q *Query) {
		q.Table = name
	}
}

// Where 添加参数化查询条件。
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

// Limit 指定返回记录上限。
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

// TxFunc 是事务回调签名，tx 参数表示同一事务内的数据访问端口。
type TxFunc func(context.Context, Executor) error

// TxOptions 保存事务执行选项。
type TxOptions struct {
	Isolation                sql.IsolationLevel
	ReadOnly                 bool
	Timeout                  time.Duration
	DisableNestedTransaction bool
}
