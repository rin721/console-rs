package database

// 本测试文件固定数据库连接与事务管理的提交回滚语义，防止注释补全和后续重构改变外部可观察行为。

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/rei0721/go-scaffold/pkg/sqlgen"
)

type txTestUser struct {
	ID    int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Name  string `gorm:"column:name;size:100"`
	Email string `gorm:"column:email;size:100;uniqueIndex"`
}

// setupTxTestDB 准备测试数据库和模型，确保每个事务用例拥有独立可控的初始状态。
func setupTxTestDB(t *testing.T) Database {
	t.Helper()
	db, err := New(&Config{
		Driver: DriverSQLite,
		DBName: filepath.Join(t.TempDir(), "tx.db"),
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	gen := sqlgen.New(&sqlgen.Config{Dialect: sqlgen.SQLite})
	schemaSQL, err := gen.TableIfNotExists(&txTestUser{})
	if err != nil {
		t.Fatalf("failed to generate schema: %v", err)
	}
	if _, err := db.Exec(context.Background(), schemaSQL); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestWithTxCommit 固定数据库连接与事务管理的提交回滚语义，确保后续注释补全或结构调整不改变该场景。
func TestWithTxCommit(t *testing.T) {
	db := setupTxTestDB(t)
	err := db.WithTx(context.Background(), func(ctx context.Context, tx Executor) error {
		return tx.Create(ctx, &txTestUser{Name: "Alice", Email: "alice@example.com"})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count, err := db.Count(context.Background(), &txTestUser{})
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 user, got %d", count)
	}
}

// TestWithTxRollback 固定数据库连接与事务管理的提交回滚语义，确保后续注释补全或结构调整不改变该场景。
func TestWithTxRollback(t *testing.T) {
	db := setupTxTestDB(t)
	expected := errors.New("stop")
	err := db.WithTx(context.Background(), func(ctx context.Context, tx Executor) error {
		if err := tx.Create(ctx, &txTestUser{Name: "Bob", Email: "bob@example.com"}); err != nil {
			return err
		}
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped stop error, got %v", err)
	}

	count, err := db.Count(context.Background(), &txTestUser{})
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rollback, got %d users", count)
	}
}

// TestWithTxNested 固定数据库连接与事务管理的提交回滚语义，确保后续注释补全或结构调整不改变该场景。
func TestWithTxNested(t *testing.T) {
	db := setupTxTestDB(t)
	err := db.WithTx(context.Background(), func(ctx context.Context, tx Executor) error {
		if err := tx.Create(ctx, &txTestUser{Name: "Outer", Email: "outer@example.com"}); err != nil {
			return err
		}
		return db.WithTx(ctx, func(ctx context.Context, tx Executor) error {
			return tx.Create(ctx, &txTestUser{Name: "Inner", Email: "inner@example.com"})
		})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count, err := db.Count(context.Background(), &txTestUser{})
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 users, got %d", count)
	}
}

// TestWithTxNestedRollback 固定数据库连接与事务管理的提交回滚语义，确保后续注释补全或结构调整不改变该场景。
func TestWithTxNestedRollback(t *testing.T) {
	db := setupTxTestDB(t)
	innerErr := errors.New("inner failed")
	err := db.WithTx(context.Background(), func(ctx context.Context, tx Executor) error {
		if err := tx.Create(ctx, &txTestUser{Name: "Outer", Email: "outer@example.com"}); err != nil {
			return err
		}
		err := db.WithTx(ctx, func(ctx context.Context, tx Executor) error {
			if err := tx.Create(ctx, &txTestUser{Name: "Inner", Email: "inner@example.com"}); err != nil {
				return err
			}
			return innerErr
		})
		if !errors.Is(err, innerErr) {
			t.Fatalf("expected inner error, got %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count, err := db.Count(context.Background(), &txTestUser{})
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected only outer user, got %d", count)
	}
}

// TestWithTxDisableNested 固定数据库连接与事务管理的提交回滚语义，确保后续注释补全或结构调整不改变该场景。
func TestWithTxDisableNested(t *testing.T) {
	db := setupTxTestDB(t)
	err := db.WithTx(context.Background(), func(ctx context.Context, tx Executor) error {
		return db.WithTxOptions(ctx, &TxOptions{DisableNestedTransaction: true, Timeout: time.Second}, func(ctx context.Context, tx Executor) error {
			return nil
		})
	})
	if !errors.Is(err, ErrNestedTransactionDisabled) {
		t.Fatalf("expected nested transaction disabled, got %v", err)
	}
}
