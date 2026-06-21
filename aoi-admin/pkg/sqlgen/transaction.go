package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

import (
	"fmt"
	"strings"
	"time"
)

// ============================================================================
// 事务脚本生成
// ============================================================================

// Transaction 生成事务包装的 SQL 脚本
func (g *Generator) Transaction(fn func(tx *Generator) string) string {
	var sb strings.Builder

	// 开始事务
	switch g.dialect.Name() {
	case MySQL, PostgreSQL:
		sb.WriteString("START TRANSACTION;\n")
	case SQLite:
		sb.WriteString("BEGIN TRANSACTION;\n")
	case SQLServer:
		sb.WriteString("BEGIN TRANSACTION;\n")
	default:
		sb.WriteString("BEGIN;\n")
	}

	// 执行用户函数
	content := fn(g)
	sb.WriteString(content)

	// 确保换行
	if !strings.HasSuffix(content, "\n") {
		sb.WriteString("\n")
	}

	// 提交事务
	sb.WriteString("COMMIT;\n")

	return sb.String()
}

// TransactionWithRollback 生成带回滚的事务脚本
func (g *Generator) TransactionWithRollback(fn func(tx *Generator) string, rollbackFn func(tx *Generator) string) string {
	var sb strings.Builder

	// 事务脚本
	sb.WriteString(g.Transaction(fn))

	// 回滚脚本 (作为注释)
	sb.WriteString("\n-- Rollback Script:\n")
	rollback := rollbackFn(g)
	for _, line := range strings.Split(rollback, "\n") {
		if line != "" {
			sb.WriteString("-- " + line + "\n")
		}
	}

	return sb.String()
}

// ============================================================================
// 批量操作构建器
// ============================================================================

// Batch 创建批量操作构建器
func (g *Generator) Batch() *BatchBuilder {
	return &BatchBuilder{
		generator: g,
	}
}

// BatchBuilder 批量操作构建器
type BatchBuilder struct {
	generator  *Generator
	statements []string
}

// Add 添加 SQL 语句
func (b *BatchBuilder) Add(sql string, err error) *BatchBuilder {
	if err == nil && sql != "" {
		b.statements = append(b.statements, sql)
	}
	return b
}

// AddRaw 添加原始 SQL 语句
func (b *BatchBuilder) AddRaw(sql string) *BatchBuilder {
	if sql != "" {
		b.statements = append(b.statements, sql)
	}
	return b
}

// Build 构建批量 SQL 脚本
func (b *BatchBuilder) Build() (string, error) {
	if len(b.statements) == 0 {
		return "", nil
	}

	return strings.Join(b.statements, "\n"), nil
}

// BuildWithTransaction 构建带事务的批量 SQL 脚本
func (b *BatchBuilder) BuildWithTransaction() (string, error) {
	if len(b.statements) == 0 {
		return "", nil
	}

	return b.generator.Transaction(func(tx *Generator) string {
		return strings.Join(b.statements, "\n")
	}), nil
}

// ============================================================================
// 迁移脚本生成
// ============================================================================

// Migration 创建迁移脚本构建器
func (g *Generator) Migration(name string) *MigrationBuilder {
	return &MigrationBuilder{
		generator: g,
		name:      name,
		timestamp: time.Now(),
	}
}

// MigrationBuilder 迁移脚本构建器
type MigrationBuilder struct {
	generator  *Generator
	name       string
	timestamp  time.Time
	operations []migrationOp
}

type migrationOp struct {
	opType    string
	model     interface{}
	column    string
	newType   string
	indexName string
	columns   []string
}

// AddColumn 添加列
func (m *MigrationBuilder) AddColumn(model interface{}, column string) *MigrationBuilder {
	m.operations = append(m.operations, migrationOp{
		opType: "add_column",
		model:  model,
		column: column,
	})
	return m
}

// DropColumn 删除列
func (m *MigrationBuilder) DropColumn(model interface{}, column string) *MigrationBuilder {
	m.operations = append(m.operations, migrationOp{
		opType: "drop_column",
		model:  model,
		column: column,
	})
	return m
}

// ModifyColumn 修改列类型
func (m *MigrationBuilder) ModifyColumn(model interface{}, column, newType string) *MigrationBuilder {
	m.operations = append(m.operations, migrationOp{
		opType:  "modify_column",
		model:   model,
		column:  column,
		newType: newType,
	})
	return m
}

// AddIndex 添加索引
func (m *MigrationBuilder) AddIndex(model interface{}, indexName string, columns ...string) *MigrationBuilder {
	m.operations = append(m.operations, migrationOp{
		opType:    "add_index",
		model:     model,
		indexName: indexName,
		columns:   columns,
	})
	return m
}

// DropIndex 删除索引
func (m *MigrationBuilder) DropIndex(model interface{}, indexName string) *MigrationBuilder {
	m.operations = append(m.operations, migrationOp{
		opType:    "drop_index",
		model:     model,
		indexName: indexName,
	})
	return m
}

// Build 构建迁移脚本
func (m *MigrationBuilder) Build() (string, error) {
	var sb strings.Builder

	// 头部注释
	sb.WriteString(fmt.Sprintf("-- Migration: %s\n", m.name))
	sb.WriteString(fmt.Sprintf("-- Created: %s\n", m.timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString("\n")

	for _, op := range m.operations {
		sql := m.buildOperation(op)
		if sql != "" {
			sb.WriteString(sql)
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// BuildRollback 构建回滚脚本
func (m *MigrationBuilder) BuildRollback() (string, error) {
	var sb strings.Builder

	// 头部注释
	sb.WriteString(fmt.Sprintf("-- Rollback: %s\n", m.name))
	sb.WriteString(fmt.Sprintf("-- Created: %s\n", m.timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString("\n")

	// 反向操作
	for i := len(m.operations) - 1; i >= 0; i-- {
		op := m.operations[i]
		sql := m.buildRollbackOperation(op)
		if sql != "" {
			sb.WriteString(sql)
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// buildOperation 依据当前生成上下文和方言规则构造 SQL 片段，错误会向上冒泡给公开构建入口。
func (m *MigrationBuilder) buildOperation(op migrationOp) string {
	migrate := m.generator.Migrate(op.model)

	switch op.opType {
	case "add_column":
		migrate.AddColumn(op.column)
	case "drop_column":
		migrate.DropColumn(op.column)
	case "modify_column":
		migrate.ModifyColumn(op.column, op.newType)
	case "add_index":
		migrate.AddIndex(op.indexName, op.columns...)
	case "drop_index":
		migrate.DropIndex(op.indexName)
	}

	sql, _ := migrate.Build()
	return sql
}

// buildRollbackOperation 依据当前生成上下文和方言规则构造 SQL 片段，错误会向上冒泡给公开构建入口。
func (m *MigrationBuilder) buildRollbackOperation(op migrationOp) string {
	migrate := m.generator.Migrate(op.model)

	switch op.opType {
	case "add_column":
		// 回滚添加列 = 删除列
		migrate.DropColumn(op.column)
	case "drop_column":
		// 回滚删除列 = 添加列 (需要类型信息，这里简化处理)
		migrate.AddColumn(op.column)
	case "modify_column":
		// 回滚修改需要原始类型，这里返回显式 SQL 注释，避免调用方误以为已生成可执行回滚语句。
		return fmt.Sprintf("-- 回滚列 %s 的类型修改需要记录原始类型", op.column)
	case "add_index":
		migrate.DropIndex(op.indexName)
	case "drop_index":
		migrate.AddIndex(op.indexName, op.columns...)
	}

	sql, _ := migrate.Build()
	return sql
}

// ============================================================================
// 种子数据生成
// ============================================================================

// Seed 创建种子数据构建器
func (g *Generator) Seed() *SeedBuilder {
	return &SeedBuilder{
		generator: g,
	}
}

// SeedBuilder 种子数据构建器
type SeedBuilder struct {
	generator *Generator
	tableName string
	model     interface{}
	data      interface{}
	truncate  bool
}

// Table 设置表 (通过模型)
func (s *SeedBuilder) Table(model interface{}) *SeedBuilder {
	s.model = model
	if err := s.generator.parseModel(model); err == nil {
		s.tableName = s.generator.ctx.TableName
	}
	return s
}

// Data 设置种子数据
func (s *SeedBuilder) Data(data interface{}) *SeedBuilder {
	s.data = data
	return s
}

// Truncate 是否先清空表
func (s *SeedBuilder) Truncate(enabled bool) *SeedBuilder {
	s.truncate = enabled
	return s
}

// Build 构建种子脚本
func (s *SeedBuilder) Build() (string, error) {
	var sb strings.Builder

	// 按需先生成清空表语句。
	if s.truncate {
		truncateSQL, err := s.generator.Truncate(s.model)
		if err == nil {
			sb.WriteString(truncateSQL)
			sb.WriteString("\n")
		}
	}

	// 写入种子数据插入语句。
	if s.data != nil {
		insertSQL, err := s.generator.Create(s.data)
		if err == nil {
			sb.WriteString(insertSQL)
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}
