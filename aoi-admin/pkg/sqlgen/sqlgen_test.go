package sqlgen

// 本测试文件固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，防止注释补全和后续重构改变外部可观察行为。

import (
	"strings"
	"testing"
	"time"
)

// ============================================================================
// 测试模型
// ============================================================================

// TestUser 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
type TestUser struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Username  string     `gorm:"column:username;size:64;not null"`
	Email     string     `gorm:"column:email;size:128"`
	Status    int        `gorm:"column:status;default:1"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
}

// TableName 为 SQL 生成测试模型提供稳定表名，避免测试结果受结构体命名策略变化影响。
func (TestUser) TableName() string {
	return "users"
}

// TestUserRole 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
type TestUserRole struct {
	ID     uint `gorm:"column:id;primaryKey;autoIncrement"`
	UserID uint `gorm:"column:user_id;not null;uniqueIndex:uk_user_roles_user_role"`
	RoleID uint `gorm:"column:role_id;not null;uniqueIndex:uk_user_roles_user_role"`
}

// TableName 为 SQL 生成测试模型提供稳定表名，避免测试结果受结构体命名策略变化影响。
func (TestUserRole) TableName() string {
	return "user_roles"
}

// ============================================================================
// Generator 测试
// ============================================================================

// TestNew 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestNew(t *testing.T) {
	gen := New(nil)
	if gen == nil {
		t.Error("New() returned nil")
	}

	gen = New(&Config{Dialect: PostgreSQL})
	if gen.config.Dialect != PostgreSQL {
		t.Error("Dialect not set correctly")
	}
}

// ============================================================================
// DDL 测试
// ============================================================================

// TestTable 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestTable(t *testing.T) {
	gen := New(&Config{Dialect: MySQL})

	sql, err := gen.Table(&TestUser{})
	if err != nil {
		t.Fatalf("Table() failed: %v", err)
	}

	if !strings.Contains(sql, "CREATE TABLE") {
		t.Error("SQL should contain CREATE TABLE")
	}

	if !strings.Contains(sql, "`users`") {
		t.Error("SQL should contain table name")
	}
}

// TestTableIfNotExists 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestTableIfNotExists(t *testing.T) {
	gen := New(&Config{Dialect: SQLite})

	sql, err := gen.TableIfNotExists(&TestUser{})
	if err != nil {
		t.Fatalf("TableIfNotExists() failed: %v", err)
	}

	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS") {
		t.Fatalf("SQL = %q, want IF NOT EXISTS", sql)
	}
	if strings.Contains(sql, "INDEX") {
		t.Fatalf("SQLite CREATE TABLE should not contain inline INDEX clauses: %q", sql)
	}
	if !strings.Contains(sql, `"id" INTEGER PRIMARY KEY AUTOINCREMENT`) {
		t.Fatalf("SQL = %q, want inline SQLite primary key autoincrement", sql)
	}
}

// TestTableGroupsCompositeUniqueIndexes 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestTableGroupsCompositeUniqueIndexes(t *testing.T) {
	gen := New(&Config{Dialect: MySQL})

	sql, err := gen.TableIfNotExists(&TestUserRole{})
	if err != nil {
		t.Fatalf("TableIfNotExists() failed: %v", err)
	}

	want := "UNIQUE INDEX `uk_user_roles_user_role` (`user_id`, `role_id`)"
	if !strings.Contains(sql, want) {
		t.Fatalf("SQL = %q, want composite unique index %q", sql, want)
	}
	if strings.Count(sql, "`uk_user_roles_user_role`") != 1 {
		t.Fatalf("SQL = %q, want one grouped unique index", sql)
	}
}

// TestDatabaseIfNotExists 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestDatabaseIfNotExists(t *testing.T) {
	gen := New(&Config{Dialect: MySQL})

	sql, err := gen.DatabaseIfNotExists("demo_app")
	if err != nil {
		t.Fatalf("DatabaseIfNotExists() failed: %v", err)
	}

	expected := "CREATE DATABASE IF NOT EXISTS `demo_app`;"
	if sql != expected {
		t.Fatalf("SQL = %q, want %q", sql, expected)
	}
}

// TestDrop 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestDrop(t *testing.T) {
	gen := New(&Config{Dialect: MySQL})

	sql, err := gen.Drop(&TestUser{})
	if err != nil {
		t.Fatalf("Drop() failed: %v", err)
	}

	expected := "DROP TABLE IF EXISTS `users`;"
	if sql != expected {
		t.Errorf("Expected %q, got %q", expected, sql)
	}
}

// ============================================================================
// INSERT 测试
// ============================================================================

// TestCreate 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestCreate(t *testing.T) {
	gen := New(&Config{Dialect: MySQL})

	user := TestUser{
		Username: "admin",
		Email:    "admin@test.com",
		Status:   1,
	}

	sql, err := gen.Create(&user)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if !strings.Contains(sql, "INSERT INTO") {
		t.Error("SQL should contain INSERT INTO")
	}

	if !strings.Contains(sql, "`users`") {
		t.Error("SQL should contain table name")
	}

	if !strings.Contains(sql, "'admin'") {
		t.Error("SQL should contain username value")
	}
}

// TestCreateBatch 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestCreateBatch(t *testing.T) {
	gen := New(&Config{Dialect: MySQL})

	users := []TestUser{
		{Username: "user1", Email: "user1@test.com"},
		{Username: "user2", Email: "user2@test.com"},
	}

	sql, err := gen.Create(&users)
	if err != nil {
		t.Fatalf("Create batch failed: %v", err)
	}

	if !strings.Contains(sql, "INSERT INTO") {
		t.Error("SQL should contain INSERT INTO")
	}

	// 应该包含两组值
	if strings.Count(sql, "(") < 3 { // 列 + 2 组值
		t.Error("SQL should contain multiple value sets")
	}
}

// ============================================================================
// SELECT 测试
// ============================================================================

// TestFirst 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestFirst(t *testing.T) {
	gen := New(&Config{Dialect: MySQL, SoftDelete: true})

	var user TestUser
	sql, err := gen.First(&user, 1)
	if err != nil {
		t.Fatalf("First() failed: %v", err)
	}

	if !strings.Contains(sql, "SELECT") {
		t.Error("SQL should contain SELECT")
	}

	if !strings.Contains(sql, "LIMIT 1") {
		t.Error("SQL should contain LIMIT 1")
	}

	if !strings.Contains(sql, "id = 1") {
		t.Error("SQL should contain id condition")
	}
}

// TestFind 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestFind(t *testing.T) {
	gen := New(&Config{Dialect: MySQL, SoftDelete: false})

	var users []TestUser
	sql, err := gen.Where("status = ?", 1).Order("created_at DESC").Limit(10).Offset(20).Find(&users)
	if err != nil {
		t.Fatalf("Find() failed: %v", err)
	}

	if !strings.Contains(sql, "SELECT *") {
		t.Error("SQL should contain SELECT *")
	}

	if !strings.Contains(sql, "status = 1") {
		t.Error("SQL should contain status condition")
	}

	if !strings.Contains(sql, "ORDER BY created_at DESC") {
		t.Error("SQL should contain ORDER BY")
	}

	if !strings.Contains(sql, "LIMIT 10") {
		t.Error("SQL should contain LIMIT")
	}

	if !strings.Contains(sql, "OFFSET 20") {
		t.Error("SQL should contain OFFSET")
	}
}

// TestUnsupportedQueryBuildersReturnErrors 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestUnsupportedQueryBuildersReturnErrors(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		build     func(*Generator) *Generator
	}{
		{
			name:      "or",
			operation: "Or",
			build: func(gen *Generator) *Generator {
				return gen.Or("status = ?", 1)
			},
		},
		{
			name:      "not",
			operation: "Not",
			build: func(gen *Generator) *Generator {
				return gen.Not("status = ?", 1)
			},
		},
		{
			name:      "group",
			operation: "Group",
			build: func(gen *Generator) *Generator {
				return gen.Group("status")
			},
		},
		{
			name:      "having",
			operation: "Having",
			build: func(gen *Generator) *Generator {
				return gen.Having("count(*) > ?", 1)
			},
		},
		{
			name:      "distinct",
			operation: "Distinct",
			build: func(gen *Generator) *Generator {
				return gen.Distinct("status")
			},
		},
		{
			name:      "joins",
			operation: "Joins",
			build: func(gen *Generator) *Generator {
				return gen.Joins("JOIN profiles ON profiles.user_id = users.id")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := New(&Config{Dialect: MySQL})
			var users []TestUser
			_, err := tt.build(gen).Find(&users)
			if !IsError(err, ErrCodeUnsupportedOperation) {
				t.Fatalf("expected unsupported operation error, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.operation) {
				t.Fatalf("expected error to mention %s, got %v", tt.operation, err)
			}
		})
	}
}

// TestUnsupportedQueryBuilderPropagatesToUpdateAndDelete 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestUnsupportedQueryBuilderPropagatesToUpdateAndDelete(t *testing.T) {
	gen := New(&Config{Dialect: MySQL})

	_, err := gen.Model(&TestUser{}).Or("id = ?", 1).Updates(map[string]interface{}{"status": 2})
	if !IsError(err, ErrCodeUnsupportedOperation) {
		t.Fatalf("expected update unsupported operation error, got %v", err)
	}

	_, err = gen.Not("id = ?", 1).Delete(&TestUser{})
	if !IsError(err, ErrCodeUnsupportedOperation) {
		t.Fatalf("expected delete unsupported operation error, got %v", err)
	}
}

// ============================================================================
// UPDATE 测试
// ============================================================================

// TestUpdates 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestUpdates(t *testing.T) {
	gen := New(&Config{Dialect: MySQL})

	sql, err := gen.Model(&TestUser{}).Where("id = ?", 1).Updates(map[string]interface{}{
		"username": "newname",
		"status":   2,
	})
	if err != nil {
		t.Fatalf("Updates() failed: %v", err)
	}

	if !strings.Contains(sql, "UPDATE") {
		t.Error("SQL should contain UPDATE")
	}

	if !strings.Contains(sql, "SET") {
		t.Error("SQL should contain SET")
	}

	if !strings.Contains(sql, "id = 1") {
		t.Error("SQL should contain WHERE condition")
	}
}

// ============================================================================
// DELETE 测试
// ============================================================================

// TestDelete 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestDelete(t *testing.T) {
	gen := New(&Config{Dialect: MySQL, SoftDelete: true})

	sql, err := gen.Delete(&TestUser{}, 1)
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// 软删除应该是 UPDATE
	if !strings.Contains(sql, "UPDATE") {
		t.Error("Soft delete should use UPDATE")
	}

	if !strings.Contains(sql, "deleted_at") {
		t.Error("Soft delete should set deleted_at")
	}
}

// TestHardDelete 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestHardDelete(t *testing.T) {
	gen := New(&Config{Dialect: MySQL, SoftDelete: true})

	sql, err := gen.Unscoped().Delete(&TestUser{}, 1)
	if err != nil {
		t.Fatalf("Unscoped Delete() failed: %v", err)
	}

	if !strings.Contains(sql, "DELETE FROM") {
		t.Error("Hard delete should use DELETE FROM")
	}
}

// TestDeleteInBatchesUnsupported 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestDeleteInBatchesUnsupported(t *testing.T) {
	gen := New(&Config{Dialect: MySQL})

	_, err := gen.DeleteInBatches(&TestUser{ID: 1}, 100)
	if !IsError(err, ErrCodeUnsupportedOperation) {
		t.Fatalf("expected unsupported operation error, got %v", err)
	}
	if !strings.Contains(err.Error(), "DeleteInBatches") {
		t.Fatalf("expected error to mention DeleteInBatches, got %v", err)
	}
}

// ============================================================================
// 逆向生成测试
// ============================================================================

// TestParseSQL 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestParseSQL(t *testing.T) {
	gen := New(&Config{Dialect: MySQL})

	ddl := `
	CREATE TABLE sys_users (
		id bigint unsigned AUTO_INCREMENT PRIMARY KEY,
		username varchar(64) NOT NULL COMMENT '用户名',
		is_active tinyint(1) DEFAULT 1,
		created_at datetime
	);`

	code, err := gen.ParseSQL(ddl).
		Package("models").
		Tags(TagGorm | TagJson).
		WithTableName(true).
		Generate()

	if err != nil {
		t.Fatalf("ParseSQL().Generate() failed: %v", err)
	}

	if !strings.Contains(code, "package models") {
		t.Error("Code should contain package declaration")
	}

	if !strings.Contains(code, "type SysUser struct") {
		t.Error("Code should contain struct definition")
	}

	if !strings.Contains(code, "gorm:") {
		t.Error("Code should contain gorm tags")
	}

	if !strings.Contains(code, "json:") {
		t.Error("Code should contain json tags")
	}

	if !strings.Contains(code, "TableName()") {
		t.Error("Code should contain TableName method")
	}
}

// TestReverseDBUnsupported 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestReverseDBUnsupported(t *testing.T) {
	builder := New(&Config{Dialect: MySQL}).ReverseDB(nil)

	if _, err := builder.Generate(); !IsError(err, ErrCodeUnsupportedOperation) {
		t.Fatalf("expected Generate unsupported operation error, got %v", err)
	}
	if _, err := builder.GenerateAll(); !IsError(err, ErrCodeUnsupportedOperation) {
		t.Fatalf("expected GenerateAll unsupported operation error, got %v", err)
	}
	if err := builder.GenerateToDir(t.TempDir()); !IsError(err, ErrCodeUnsupportedOperation) {
		t.Fatalf("expected GenerateToDir unsupported operation error, got %v", err)
	}
}

// ============================================================================
// 方言测试
// ============================================================================

// TestDialectQuote 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestDialectQuote(t *testing.T) {
	tests := []struct {
		dialect  Dialect
		input    string
		expected string
	}{
		{MySQL, "users", "`users`"},
		{PostgreSQL, "users", "\"users\""},
		{SQLite, "users", "\"users\""},
		{SQLServer, "users", "[users]"},
	}

	for _, tt := range tests {
		d := getDialect(tt.dialect)
		result := d.Quote(tt.input)
		if result != tt.expected {
			t.Errorf("Dialect %s: Quote(%q) = %q, expected %q",
				tt.dialect, tt.input, result, tt.expected)
		}
	}
}

// ============================================================================
// 事务测试
// ============================================================================

// TestTransaction 固定 SQL 生成器的方言、DDL、CRUD 和逆向生成契约，确保后续注释补全或结构调整不改变该场景。
func TestTransaction(t *testing.T) {
	gen := New(&Config{Dialect: MySQL})

	sql := gen.Transaction(func(tx *Generator) string {
		s1, _ := tx.Create(&TestUser{Username: "user1"})
		return s1
	})

	if !strings.Contains(sql, "START TRANSACTION") {
		t.Error("Transaction should start with START TRANSACTION")
	}

	if !strings.Contains(sql, "COMMIT") {
		t.Error("Transaction should end with COMMIT")
	}

	if !strings.Contains(sql, "INSERT INTO") {
		t.Error("Transaction should contain INSERT")
	}
}
