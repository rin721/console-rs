package adapters

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rei0721/go-scaffold/internal/ports"
	"github.com/rei0721/go-scaffold/pkg/database"
)

// Database 将 pkg/database.Database 适配为 internal/ports.Database。
//
// 该类型是模块服务与基础设施数据库包之间的边界，负责类型转换和错误归一化。
type Database struct {
	inner database.Database
}

// Executor 将 pkg/database.Executor 适配为 internal/ports.Executor。
type Executor struct {
	inner database.Executor
}

// NewDatabase 创建数据库端口适配器；nil 输入表示数据库能力不可用。
func NewDatabase(db database.Database) ports.Database {
	if db == nil {
		return nil
	}
	return Database{inner: db}
}

// NewExecutor 创建事务或普通执行器的端口适配器。
func NewExecutor(executor database.Executor) ports.Executor {
	if executor == nil {
		return nil
	}
	return Executor{inner: executor}
}

// UnwrapDatabase 在需要调用基础设施特有能力时解包数据库适配器。
func UnwrapDatabase(db ports.Database) database.Database {
	if wrapped, ok := db.(Database); ok {
		return wrapped.inner
	}
	if wrapped, ok := db.(*Database); ok && wrapped != nil {
		return wrapped.inner
	}
	return nil
}

// UnwrapExecutor 在需要调用基础设施特有能力时解包执行器适配器。
func UnwrapExecutor(executor ports.Executor) database.Executor {
	if wrapped, ok := executor.(Executor); ok {
		return wrapped.inner
	}
	if wrapped, ok := executor.(*Executor); ok && wrapped != nil {
		return wrapped.inner
	}
	return nil
}

func (d Database) Close() error {
	return d.inner.Close()
}

func (d Database) Ping(ctx context.Context) error {
	return d.inner.Ping(ctx)
}

func (d Database) SQLDB() (*sql.DB, error) {
	return d.inner.SQLDB()
}

// WithTx 在基础设施事务中注入端口层 Executor，保持服务层不依赖 pkg/database。
func (d Database) WithTx(ctx context.Context, fn ports.TxFunc) error {
	return mapDatabaseError(d.inner.WithTx(ctx, func(txCtx context.Context, tx database.Executor) error {
		return fn(txCtx, NewExecutor(tx))
	}))
}

// WithTxOptions 转换事务选项后执行事务函数。
func (d Database) WithTxOptions(ctx context.Context, opts *ports.TxOptions, fn ports.TxFunc) error {
	return mapDatabaseError(d.inner.WithTxOptions(ctx, databaseTxOptions(opts), func(txCtx context.Context, tx database.Executor) error {
		return fn(txCtx, NewExecutor(tx))
	}))
}

func (d Database) Create(ctx context.Context, value any) error {
	return mapDatabaseError(d.inner.Create(ctx, value))
}

func (d Database) Save(ctx context.Context, value any) error {
	return mapDatabaseError(d.inner.Save(ctx, value))
}

func (d Database) First(ctx context.Context, dest any, opts ...ports.QueryOption) error {
	return mapDatabaseError(d.inner.First(ctx, dest, databaseQueryOptions(opts)...))
}

func (d Database) Find(ctx context.Context, dest any, opts ...ports.QueryOption) error {
	return mapDatabaseError(d.inner.Find(ctx, dest, databaseQueryOptions(opts)...))
}

func (d Database) Update(ctx context.Context, model any, values map[string]any, opts ...ports.QueryOption) (ports.Result, error) {
	result, err := d.inner.Update(ctx, model, values, databaseQueryOptions(opts)...)
	return ports.Result{RowsAffected: result.RowsAffected}, mapDatabaseError(err)
}

func (d Database) Delete(ctx context.Context, model any, opts ...ports.QueryOption) (ports.Result, error) {
	result, err := d.inner.Delete(ctx, model, databaseQueryOptions(opts)...)
	return ports.Result{RowsAffected: result.RowsAffected}, mapDatabaseError(err)
}

func (d Database) Exec(ctx context.Context, sql string, args ...any) (ports.Result, error) {
	result, err := d.inner.Exec(ctx, sql, args...)
	return ports.Result{RowsAffected: result.RowsAffected}, mapDatabaseError(err)
}

func (d Database) Raw(ctx context.Context, dest any, sql string, args ...any) (ports.Result, error) {
	result, err := d.inner.Raw(ctx, dest, sql, args...)
	return ports.Result{RowsAffected: result.RowsAffected}, mapDatabaseError(err)
}

func (d Database) Count(ctx context.Context, model any, opts ...ports.QueryOption) (int64, error) {
	count, err := d.inner.Count(ctx, model, databaseQueryOptions(opts)...)
	return count, mapDatabaseError(err)
}

func (d Database) HasTable(ctx context.Context, model any) (bool, error) {
	return d.inner.HasTable(ctx, model)
}

func (e Executor) Create(ctx context.Context, value any) error {
	return mapDatabaseError(e.inner.Create(ctx, value))
}

func (e Executor) Save(ctx context.Context, value any) error {
	return mapDatabaseError(e.inner.Save(ctx, value))
}

func (e Executor) First(ctx context.Context, dest any, opts ...ports.QueryOption) error {
	return mapDatabaseError(e.inner.First(ctx, dest, databaseQueryOptions(opts)...))
}

func (e Executor) Find(ctx context.Context, dest any, opts ...ports.QueryOption) error {
	return mapDatabaseError(e.inner.Find(ctx, dest, databaseQueryOptions(opts)...))
}

func (e Executor) Update(ctx context.Context, model any, values map[string]any, opts ...ports.QueryOption) (ports.Result, error) {
	result, err := e.inner.Update(ctx, model, values, databaseQueryOptions(opts)...)
	return ports.Result{RowsAffected: result.RowsAffected}, mapDatabaseError(err)
}

func (e Executor) Delete(ctx context.Context, model any, opts ...ports.QueryOption) (ports.Result, error) {
	result, err := e.inner.Delete(ctx, model, databaseQueryOptions(opts)...)
	return ports.Result{RowsAffected: result.RowsAffected}, mapDatabaseError(err)
}

func (e Executor) Exec(ctx context.Context, sql string, args ...any) (ports.Result, error) {
	result, err := e.inner.Exec(ctx, sql, args...)
	return ports.Result{RowsAffected: result.RowsAffected}, mapDatabaseError(err)
}

func (e Executor) Raw(ctx context.Context, dest any, sql string, args ...any) (ports.Result, error) {
	result, err := e.inner.Raw(ctx, dest, sql, args...)
	return ports.Result{RowsAffected: result.RowsAffected}, mapDatabaseError(err)
}

func (e Executor) Count(ctx context.Context, model any, opts ...ports.QueryOption) (int64, error) {
	count, err := e.inner.Count(ctx, model, databaseQueryOptions(opts)...)
	return count, mapDatabaseError(err)
}

func (e Executor) HasTable(ctx context.Context, model any) (bool, error) {
	return e.inner.HasTable(ctx, model)
}

// databaseQueryOptions 将端口层查询配置转换为 pkg/database 查询配置。
//
// 每个 QueryOption 都会先作用到端口层 Query，再复制到基础设施 Query，避免跨包共享可变结构。
func databaseQueryOptions(opts []ports.QueryOption) []database.QueryOption {
	out := make([]database.QueryOption, 0, len(opts))
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		q := ports.Query{}
		opt(&q)
		out = append(out, func(target *database.Query) {
			target.Table = q.Table
			target.Order = q.Order
			target.Limit = q.Limit
			target.Offset = q.Offset
			target.Unscoped = q.Unscoped
			target.WithDeleted = q.WithDeleted
			for _, condition := range q.Where {
				target.Where = append(target.Where, database.Condition{
					Expr: condition.Expr,
					Args: condition.Args,
				})
			}
		})
	}
	return out
}

// databaseTxOptions 将端口层事务选项转换为基础设施事务选项。
func databaseTxOptions(opts *ports.TxOptions) *database.TxOptions {
	if opts == nil {
		return nil
	}
	return &database.TxOptions{
		Isolation:                opts.Isolation,
		ReadOnly:                 opts.ReadOnly,
		Timeout:                  opts.Timeout,
		DisableNestedTransaction: opts.DisableNestedTransaction,
	}
}

// mapDatabaseError 将基础设施数据库错误归一化为端口层错误。
//
// 服务层只匹配 ports 错误，避免直接依赖 pkg/database 的错误哨兵。
func mapDatabaseError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, database.ErrNotFound):
		return ports.ErrNotFound
	case errors.Is(err, database.ErrNilTxFunc):
		return ports.ErrNilTxFunc
	case errors.Is(err, database.ErrInvalidTxOptions):
		return ports.ErrInvalidTxOptions
	case errors.Is(err, database.ErrNestedTransactionDisabled):
		return ports.ErrNestedTransactionDisabled
	default:
		return err
	}
}
