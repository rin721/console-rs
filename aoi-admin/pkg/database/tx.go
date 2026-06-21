package database

// 本文件属于数据库基础设施层，定义连接工厂、池参数、事务上下文和热重载资源替换边界。

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const defaultTxTimeout = 30 * time.Second

var (
	// ErrNilTxFunc 表示事务回调为空，调用方没有提供可执行的事务主体。
	ErrNilTxFunc = errors.New("transaction function is nil")
	// ErrInvalidTxOptions 表示事务配置为空或非法，事务包装器无法确定超时与嵌套策略。
	ErrInvalidTxOptions = errors.New("invalid transaction options")
	// ErrNestedTransactionDisabled 表示当前上下文已有事务且配置禁止嵌套事务。
	ErrNestedTransactionDisabled = errors.New("nested transaction is disabled")
)

type txContextKey struct{}

// DefaultTxOptions 返回事务执行的默认边界，包括超时、隔离级别和嵌套事务策略。
func DefaultTxOptions() *TxOptions {
	return &TxOptions{
		Isolation: sql.LevelDefault,
		Timeout:   defaultTxTimeout,
	}
}

// normalize 为事务选项填充默认值并校验超时、隔离级别和嵌套策略。
func (opts *TxOptions) normalize() error {
	if opts == nil {
		return ErrInvalidTxOptions
	}
	if opts.Timeout < 0 {
		return ErrInvalidTxOptions
	}
	if opts.Timeout == 0 {
		opts.Timeout = defaultTxTimeout
	}
	return nil
}

// txFromContext 从 context 中提取当前事务句柄，用于识别服务层的嵌套事务调用。
func txFromContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return nil
	}
	tx, _ := ctx.Value(txContextKey{}).(*gorm.DB)
	return tx
}

// contextWithTx 将 GORM 事务句柄写入 context，使下游仓储可以复用同一个事务。
func contextWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, txContextKey{}, tx)
}

// WithTx 使用默认事务选项执行回调，是服务层最常用的事务入口。
func (d *database) WithTx(ctx context.Context, fn TxFunc) error {
	return d.WithTxOptions(ctx, nil, fn)
}

// WithTxOptions 使用显式事务选项执行回调，并处理嵌套事务、超时和回滚语义。
func (d *database) WithTxOptions(ctx context.Context, opts *TxOptions, fn TxFunc) error {
	if fn == nil {
		return ErrNilTxFunc
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if opts == nil {
		opts = DefaultTxOptions()
	} else if err := opts.normalize(); err != nil {
		return err
	}

	txCtx := ctx
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		txCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	if parentTx := txFromContext(txCtx); parentTx != nil {
		if opts.DisableNestedTransaction {
			return ErrNestedTransactionDisabled
		}
		return runGormTransaction(txCtx, parentTx.WithContext(txCtx), opts, fn)
	}

	return runGormTransaction(txCtx, d.gormDB().WithContext(txCtx), opts, fn)
}

// runGormTransaction 包装 GORM Transaction 调用，负责注入事务 context 并统一错误前缀。
func runGormTransaction(ctx context.Context, db *gorm.DB, opts *TxOptions, fn TxFunc) error {
	txOpts := &sql.TxOptions{
		Isolation: opts.Isolation,
		ReadOnly:  opts.ReadOnly,
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextWithTx(ctx, tx)
		if err := fn(txCtx, &gormExecutor{db: tx}); err != nil {
			return err
		}
		if err := txCtx.Err(); err != nil {
			return err
		}
		return nil
	}, txOpts)
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}
	return nil
}
