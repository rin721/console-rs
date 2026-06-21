package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

import (
	"fmt"
	"strings"
)

// ============================================================================
// DDL 生成方法
// ============================================================================

// Table 生成 CREATE TABLE 语句
func (g *Generator) Table(model interface{}) (string, error) {
	if err := g.parseModel(model); err != nil {
		return "", err
	}

	fields := parseStructFields(g.ctx.ModelType, g.ctx.ModelValue, g.dialect)
	if len(fields) == 0 {
		return "", ErrInvalidModel
	}

	return g.buildCreateTable(g.ctx.TableName, fields, false), nil
}

// TableIfNotExists 生成带 IF NOT EXISTS 保护的 CREATE TABLE 语句。
func (g *Generator) TableIfNotExists(model interface{}) (string, error) {
	if err := g.parseModel(model); err != nil {
		return "", err
	}

	fields := parseStructFields(g.ctx.ModelType, g.ctx.ModelValue, g.dialect)
	if len(fields) == 0 {
		return "", ErrInvalidModel
	}

	return g.buildCreateTable(g.ctx.TableName, fields, true), nil
}

// Database 生成 CREATE DATABASE 语句。
func (g *Generator) Database(name string) (string, error) {
	return g.buildCreateDatabase(name, false)
}

// DatabaseIfNotExists 在方言支持时生成 CREATE DATABASE IF NOT EXISTS 语句。
func (g *Generator) DatabaseIfNotExists(name string) (string, error) {
	return g.buildCreateDatabase(name, true)
}

// Drop 生成 DROP TABLE 语句
func (g *Generator) Drop(model interface{}) (string, error) {
	if err := g.parseModel(model); err != nil {
		return "", err
	}

	return g.buildDropTable(g.ctx.TableName), nil
}

// Migrate 返回迁移构建器
func (g *Generator) Migrate(model interface{}) *MigrateBuilder {
	ng := g.clone()
	_ = ng.parseModel(model)

	return &MigrateBuilder{
		generator: ng,
		tableName: ng.ctx.TableName,
		fields:    parseStructFields(ng.ctx.ModelType, ng.ctx.ModelValue, ng.dialect),
	}
}

// ============================================================================
// CREATE TABLE 构建
// ============================================================================

// buildCreateTable 依据当前生成上下文和方言规则构造 SQL 片段，错误会向上冒泡给公开构建入口。
func (g *Generator) buildCreateTable(tableName string, fields []FieldInfo, ifNotExists bool) string {
	var sb strings.Builder
	quotedTable := g.dialect.Quote(tableName)

	sb.WriteString("CREATE TABLE ")
	if ifNotExists {
		sb.WriteString("IF NOT EXISTS ")
	}
	sb.WriteString(quotedTable)
	sb.WriteString(" (\n")

	var columnDefs []string
	var primaryKeys []string

	for _, field := range fields {
		colDef := g.buildColumnDef(field)
		columnDefs = append(columnDefs, "  "+colDef)

		if field.Tag.PrimaryKey && !g.isInlineSQLitePrimaryKey(field) {
			primaryKeys = append(primaryKeys, g.dialect.Quote(field.ColumnName))
		}

	}
	indexes := g.buildInlineIndexes(tableName, fields)

	sb.WriteString(strings.Join(columnDefs, ",\n"))

	// 添加主键约束
	if len(primaryKeys) > 0 {
		sb.WriteString(",\n  PRIMARY KEY (")
		sb.WriteString(strings.Join(primaryKeys, ", "))
		sb.WriteString(")")
	}

	// 添加索引
	for _, idx := range indexes {
		sb.WriteString(",\n  ")
		sb.WriteString(idx)
	}

	sb.WriteString("\n)")

	// 添加引擎子句 (MySQL)
	if engine := g.dialect.EngineClause(); engine != "" {
		sb.WriteString(" ")
		sb.WriteString(engine)
	}

	sb.WriteString(";")

	return sb.String()
}

// buildColumnDef 构建列定义
type inlineIndexSpec struct {
	name    string
	unique  bool
	columns []string
}

// buildInlineIndexes 依据当前生成上下文和方言规则构造 SQL 片段，错误会向上冒泡给公开构建入口。
func (g *Generator) buildInlineIndexes(tableName string, fields []FieldInfo) []string {
	if !g.supportsInlineIndexes() {
		return nil
	}

	order := make([]string, 0)
	specs := make(map[string]*inlineIndexSpec)
	add := func(rawName string, unique bool, field FieldInfo) {
		if rawName == "" {
			return
		}
		name := rawName
		if name == "true" {
			prefix := "idx"
			if unique {
				prefix = "uk"
			}
			name = prefix + "_" + tableName + "_" + field.ColumnName
		}
		key := fmt.Sprintf("%t:%s", unique, name)
		spec := specs[key]
		if spec == nil {
			spec = &inlineIndexSpec{name: name, unique: unique}
			specs[key] = spec
			order = append(order, key)
		}
		spec.columns = append(spec.columns, field.ColumnName)
	}

	for _, field := range fields {
		add(field.Tag.Index, false, field)
		add(field.Tag.UniqueIndex, true, field)
	}

	indexes := make([]string, 0, len(order))
	for _, key := range order {
		spec := specs[key]
		quotedColumns := make([]string, 0, len(spec.columns))
		for _, column := range spec.columns {
			quotedColumns = append(quotedColumns, g.dialect.Quote(column))
		}
		kind := "INDEX"
		if spec.unique {
			kind = "UNIQUE INDEX"
		}
		indexes = append(indexes, fmt.Sprintf("%s %s (%s)",
			kind, g.dialect.Quote(spec.name), strings.Join(quotedColumns, ", ")))
	}
	return indexes
}

// buildColumnDef 依据当前生成上下文和方言规则构造 SQL 片段，错误会向上冒泡给公开构建入口。
func (g *Generator) buildColumnDef(field FieldInfo) string {
	var parts []string

	// 列名
	parts = append(parts, g.dialect.Quote(field.ColumnName))

	// 类型
	parts = append(parts, field.SQLType)

	if g.isInlineSQLitePrimaryKey(field) {
		parts = append(parts, "PRIMARY KEY", g.dialect.AutoIncrementKeyword())
		return strings.Join(parts, " ")
	}

	// 添加 NOT NULL 约束。
	if field.Tag.NotNull || field.Tag.PrimaryKey {
		parts = append(parts, "NOT NULL")
	}

	// 添加自增关键字；不同方言可能没有对应写法。
	if field.Tag.AutoIncrement {
		autoIncr := g.dialect.AutoIncrementKeyword()
		if autoIncr != "" {
			parts = append(parts, autoIncr)
		}
	}

	// 添加默认值表达式，保持调用方提供的 SQL 片段。
	if field.Tag.Default != "" {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", field.Tag.Default))
	}

	// COMMENT (MySQL 特有)
	if field.Tag.Comment != "" && g.dialect.Name() == MySQL {
		parts = append(parts, fmt.Sprintf("COMMENT '%s'", escapeString(field.Tag.Comment)))
	}

	return strings.Join(parts, " ")
}

// isInlineSQLitePrimaryKey 实现 *Generator 的内部辅助方法，封装该类型的局部控制流和错误边界。
func (g *Generator) isInlineSQLitePrimaryKey(field FieldInfo) bool {
	return g.dialect.Name() == SQLite && field.Tag.PrimaryKey && field.Tag.AutoIncrement
}

// supportsInlineIndexes 实现 *Generator 的内部辅助方法，封装该类型的局部控制流和错误边界。
func (g *Generator) supportsInlineIndexes() bool {
	return g.dialect.Name() == MySQL
}

// buildCreateDatabase 依据当前生成上下文和方言规则构造 SQL 片段，错误会向上冒泡给公开构建入口。
func (g *Generator) buildCreateDatabase(name string, ifNotExists bool) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", NewError(ErrCodeInvalidSQL, "database name is required")
	}
	if g.dialect.Name() == SQLite {
		return "", NewUnsupportedError("create database")
	}
	if ifNotExists && g.dialect.Name() == PostgreSQL {
		return "", NewUnsupportedError("create database if not exists")
	}

	quotedName := g.dialect.Quote(name)
	if ifNotExists && g.dialect.Name() == SQLServer {
		return fmt.Sprintf("IF DB_ID(N'%s') IS NULL CREATE DATABASE %s;", escapeString(name), quotedName), nil
	}

	var sb strings.Builder
	sb.WriteString("CREATE DATABASE ")
	if ifNotExists {
		sb.WriteString("IF NOT EXISTS ")
	}
	sb.WriteString(quotedName)
	sb.WriteString(";")
	return sb.String(), nil
}

// ============================================================================
// DROP TABLE 构建
// ============================================================================

// buildDropTable 依据当前生成上下文和方言规则构造 SQL 片段，错误会向上冒泡给公开构建入口。
func (g *Generator) buildDropTable(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;", g.dialect.Quote(tableName))
}

// ============================================================================
// MigrateBuilder 迁移构建器
// ============================================================================

// MigrateBuilder 迁移操作构建器
type MigrateBuilder struct {
	generator  *Generator
	tableName  string
	fields     []FieldInfo
	operations []string
}

// AddColumn 添加列
func (m *MigrateBuilder) AddColumn(columnName string) *MigrateBuilder {
	for _, field := range m.fields {
		if field.Name == columnName || field.ColumnName == columnName {
			colDef := m.generator.buildColumnDef(field)
			m.operations = append(m.operations, fmt.Sprintf("ADD COLUMN %s", colDef))
			break
		}
	}
	return m
}

// DropColumn 删除列
func (m *MigrateBuilder) DropColumn(columnName string) *MigrateBuilder {
	m.operations = append(m.operations, fmt.Sprintf("DROP COLUMN %s",
		m.generator.dialect.Quote(toSnakeCase(columnName))))
	return m
}

// ModifyColumn 修改列类型
func (m *MigrateBuilder) ModifyColumn(columnName, newType string) *MigrateBuilder {
	keyword := "MODIFY COLUMN"
	if m.generator.dialect.Name() == PostgreSQL {
		keyword = "ALTER COLUMN"
	}
	m.operations = append(m.operations, fmt.Sprintf("%s %s %s",
		keyword, m.generator.dialect.Quote(toSnakeCase(columnName)), newType))
	return m
}

// RenameColumn 重命名列
func (m *MigrateBuilder) RenameColumn(oldName, newName string) *MigrateBuilder {
	m.operations = append(m.operations, fmt.Sprintf("RENAME COLUMN %s TO %s",
		m.generator.dialect.Quote(toSnakeCase(oldName)),
		m.generator.dialect.Quote(toSnakeCase(newName))))
	return m
}

// AddIndex 添加索引
func (m *MigrateBuilder) AddIndex(indexName string, columns ...string) *MigrateBuilder {
	quotedCols := make([]string, len(columns))
	for i, col := range columns {
		quotedCols[i] = m.generator.dialect.Quote(toSnakeCase(col))
	}

	m.operations = append(m.operations, fmt.Sprintf("ADD INDEX %s (%s)",
		m.generator.dialect.Quote(indexName), strings.Join(quotedCols, ", ")))
	return m
}

// DropIndex 删除索引
func (m *MigrateBuilder) DropIndex(indexName string) *MigrateBuilder {
	m.operations = append(m.operations, fmt.Sprintf("DROP INDEX %s",
		m.generator.dialect.Quote(indexName)))
	return m
}

// Build 生成 ALTER TABLE 语句
func (m *MigrateBuilder) Build() (string, error) {
	if len(m.operations) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("ALTER TABLE ")
	sb.WriteString(m.generator.dialect.Quote(m.tableName))
	sb.WriteString("\n  ")
	sb.WriteString(strings.Join(m.operations, ",\n  "))
	sb.WriteString(";")

	return sb.String(), nil
}
