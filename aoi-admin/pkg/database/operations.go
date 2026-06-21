package database

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// gormExecutor 将某个具体 GORM 会话暴露为 Executor。
//
// 事务回调中会使用它包住 tx，确保仓储继续通过统一接口访问同一个事务。
type gormExecutor struct {
	db *gorm.DB
}

// executor 返回当前操作应使用的 GORM 会话。
//
// 如果 context 中已有事务句柄，则复用该事务；否则使用数据库当前连接。这样仓储方法无需显式区分事务内外。
func (d *database) executor(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	if tx := txFromContext(ctx); tx != nil {
		return tx.WithContext(ctx)
	}
	return d.gormDB().WithContext(ctx)
}

func (d *database) Create(ctx context.Context, value any) error {
	return mapError(d.executor(ctx).Create(value).Error)
}

func (d *database) Save(ctx context.Context, value any) error {
	return mapError(d.executor(ctx).Save(value).Error)
}

func (d *database) First(ctx context.Context, dest any, opts ...QueryOption) error {
	return mapError(applyQuery(d.executor(ctx), opts...).First(dest).Error)
}

func (d *database) Find(ctx context.Context, dest any, opts ...QueryOption) error {
	return mapError(applyQuery(d.executor(ctx), opts...).Find(dest).Error)
}

func (d *database) Update(ctx context.Context, model any, values map[string]any, opts ...QueryOption) (Result, error) {
	result := applyQuery(d.executor(ctx).Model(model), opts...).Updates(values)
	return Result{RowsAffected: result.RowsAffected}, mapError(result.Error)
}

func (d *database) Delete(ctx context.Context, model any, opts ...QueryOption) (Result, error) {
	result := applyQuery(d.executor(ctx), opts...).Delete(model)
	return Result{RowsAffected: result.RowsAffected}, mapError(result.Error)
}

func (d *database) Exec(ctx context.Context, sql string, args ...any) (Result, error) {
	result := d.executor(ctx).Exec(sql, args...)
	return Result{RowsAffected: result.RowsAffected}, mapError(result.Error)
}

func (d *database) Raw(ctx context.Context, dest any, sql string, args ...any) (Result, error) {
	result := d.executor(ctx).Raw(sql, args...).Scan(dest)
	return Result{RowsAffected: result.RowsAffected}, mapError(result.Error)
}

func (d *database) Count(ctx context.Context, model any, opts ...QueryOption) (int64, error) {
	var count int64
	result := applyQuery(d.executor(ctx).Model(model), opts...).Count(&count)
	return count, mapError(result.Error)
}

func (d *database) HasTable(ctx context.Context, model any) (bool, error) {
	return d.executor(ctx).Migrator().HasTable(model), nil
}

func (e *gormExecutor) Create(ctx context.Context, value any) error {
	return mapError(e.withContext(ctx).Create(value).Error)
}

func (e *gormExecutor) Save(ctx context.Context, value any) error {
	return mapError(e.withContext(ctx).Save(value).Error)
}

func (e *gormExecutor) First(ctx context.Context, dest any, opts ...QueryOption) error {
	return mapError(applyQuery(e.withContext(ctx), opts...).First(dest).Error)
}

func (e *gormExecutor) Find(ctx context.Context, dest any, opts ...QueryOption) error {
	return mapError(applyQuery(e.withContext(ctx), opts...).Find(dest).Error)
}

func (e *gormExecutor) Update(ctx context.Context, model any, values map[string]any, opts ...QueryOption) (Result, error) {
	result := applyQuery(e.withContext(ctx).Model(model), opts...).Updates(values)
	return Result{RowsAffected: result.RowsAffected}, mapError(result.Error)
}

func (e *gormExecutor) Delete(ctx context.Context, model any, opts ...QueryOption) (Result, error) {
	result := applyQuery(e.withContext(ctx), opts...).Delete(model)
	return Result{RowsAffected: result.RowsAffected}, mapError(result.Error)
}

func (e *gormExecutor) Exec(ctx context.Context, sql string, args ...any) (Result, error) {
	result := e.withContext(ctx).Exec(sql, args...)
	return Result{RowsAffected: result.RowsAffected}, mapError(result.Error)
}

func (e *gormExecutor) Raw(ctx context.Context, dest any, sql string, args ...any) (Result, error) {
	result := e.withContext(ctx).Raw(sql, args...).Scan(dest)
	return Result{RowsAffected: result.RowsAffected}, mapError(result.Error)
}

func (e *gormExecutor) Count(ctx context.Context, model any, opts ...QueryOption) (int64, error) {
	var count int64
	result := applyQuery(e.withContext(ctx).Model(model), opts...).Count(&count)
	return count, mapError(result.Error)
}

func (e *gormExecutor) HasTable(ctx context.Context, model any) (bool, error) {
	return e.withContext(ctx).Migrator().HasTable(model), nil
}

// withContext 为事务执行器补齐 context，避免 nil context 传入 GORM。
func (e *gormExecutor) withContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return e.db.WithContext(ctx)
}

// applyQuery 把通用 QueryOption 应用到 GORM 查询链上。
//
// 选项为空时保持原查询；Limit/Offset 只在正数时生效，避免零值意外改变 GORM 默认行为。
func applyQuery(db *gorm.DB, opts ...QueryOption) *gorm.DB {
	q := Query{}
	for _, opt := range opts {
		if opt != nil {
			opt(&q)
		}
	}
	if q.Table != "" {
		db = db.Table(q.Table)
	}
	for _, condition := range q.Where {
		db = db.Where(condition.Expr, condition.Args...)
	}
	if q.Order != "" {
		db = db.Order(q.Order)
	}
	if q.Limit > 0 {
		db = db.Limit(q.Limit)
	}
	if q.Offset > 0 {
		db = db.Offset(q.Offset)
	}
	return db
}

// mapError 将 GORM 错误转换为本包公开错误。
//
// 调用方只需要匹配 database.ErrNotFound，不必依赖 GORM 的错误哨兵。
func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
