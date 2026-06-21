package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

import (
	"fmt"
	"reflect"
	"strings"
)

// ============================================================================
// INSERT 生成方法
// ============================================================================

// Create 生成 INSERT 语句
// 支持单个对象或对象切片
func (g *Generator) Create(value interface{}) (string, error) {
	if value == nil {
		return "", ErrEmptyData
	}

	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// 批量插入
	if v.Kind() == reflect.Slice {
		return g.createBatch(value)
	}

	// 单条插入
	return g.createSingle(value)
}

// createSingle 生成单条 INSERT 语句
func (g *Generator) createSingle(value interface{}) (string, error) {
	if err := g.parseModel(value); err != nil {
		return "", err
	}

	fields := parseStructFields(g.ctx.ModelType, g.ctx.ModelValue, g.dialect)
	if len(fields) == 0 {
		return "", ErrInvalidModel
	}

	// 过滤字段
	fields = g.filterFields(fields, true)

	return g.buildInsert(g.ctx.TableName, fields, [][]interface{}{g.getFieldValues(fields)}), nil
}

// createBatch 生成批量 INSERT 语句
func (g *Generator) createBatch(values interface{}) (string, error) {
	v := reflect.ValueOf(values)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Len() == 0 {
		return "", ErrEmptyData
	}

	// 从第一个元素获取结构信息
	first := v.Index(0)
	if first.Kind() == reflect.Ptr {
		first = first.Elem()
	}

	if err := g.parseModel(first.Addr().Interface()); err != nil {
		return "", err
	}

	fields := parseStructFields(g.ctx.ModelType, first, g.dialect)
	fields = g.filterFields(fields, true)

	// 收集所有值
	var allValues [][]interface{}
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}

		itemFields := parseStructFields(g.ctx.ModelType, item, g.dialect)
		itemFields = g.filterFields(itemFields, true)
		allValues = append(allValues, g.getFieldValuesFromFields(itemFields))
	}

	return g.buildInsert(g.ctx.TableName, fields, allValues), nil
}

// ============================================================================
// INSERT 语句构建
// ============================================================================

// buildInsert 依据当前生成上下文和方言规则构造 SQL 片段，错误会向上冒泡给公开构建入口。
func (g *Generator) buildInsert(tableName string, fields []FieldInfo, valuesList [][]interface{}) string {
	if len(fields) == 0 || len(valuesList) == 0 {
		return ""
	}

	var sb strings.Builder
	quotedTable := g.dialect.Quote(tableName)

	sb.WriteString("INSERT INTO ")
	sb.WriteString(quotedTable)
	sb.WriteString(" (")

	// 列名
	columnNames := make([]string, len(fields))
	for i, field := range fields {
		columnNames[i] = g.dialect.Quote(field.ColumnName)
	}
	sb.WriteString(strings.Join(columnNames, ", "))
	sb.WriteString(") VALUES ")

	// 值列表
	var valueSets []string
	for _, values := range valuesList {
		valueStrs := make([]string, len(values))
		for i, v := range values {
			valueStrs[i] = formatValue(v, g.dialect.Quote)
		}
		valueSets = append(valueSets, "("+strings.Join(valueStrs, ", ")+")")
	}

	sb.WriteString(strings.Join(valueSets, ", "))
	sb.WriteString(";")

	return sb.String()
}

// ============================================================================
// 辅助方法
// ============================================================================

// filterFields 根据 Select/Omit 过滤字段
func (g *Generator) filterFields(fields []FieldInfo, skipAutoIncrement bool) []FieldInfo {
	var result []FieldInfo

	selectMap := make(map[string]bool)
	for _, col := range g.ctx.SelectColumns {
		selectMap[strings.ToLower(col)] = true
	}

	omitMap := make(map[string]bool)
	for _, col := range g.ctx.OmitColumns {
		omitMap[strings.ToLower(col)] = true
	}

	for _, field := range fields {
		// 跳过自增字段
		if skipAutoIncrement && field.Tag.AutoIncrement {
			continue
		}

		fieldNameLower := strings.ToLower(field.Name)
		columnNameLower := strings.ToLower(field.ColumnName)

		// 如果指定了 Select，只包含选中的字段
		if len(selectMap) > 0 {
			if !selectMap[fieldNameLower] && !selectMap[columnNameLower] {
				continue
			}
		}

		// 如果在 Omit 列表中，跳过
		if omitMap[fieldNameLower] || omitMap[columnNameLower] {
			continue
		}

		result = append(result, field)
	}

	return result
}

// getFieldValues 获取字段值
func (g *Generator) getFieldValues(fields []FieldInfo) []interface{} {
	values := make([]interface{}, len(fields))
	for i, field := range fields {
		values[i] = field.Value
	}
	return values
}

// getFieldValuesFromFields 从 FieldInfo 获取值
func (g *Generator) getFieldValuesFromFields(fields []FieldInfo) []interface{} {
	values := make([]interface{}, len(fields))
	for i, field := range fields {
		values[i] = field.Value
	}
	return values
}

// ============================================================================
// 插入选项
// ============================================================================

// OnConflict 设置冲突处理 (UPSERT)
func (g *Generator) OnConflict(columns ...string) *ConflictBuilder {
	return &ConflictBuilder{
		generator: g.clone(),
		columns:   columns,
	}
}

// ConflictBuilder 冲突处理构建器
type ConflictBuilder struct {
	generator     *Generator
	columns       []string
	updateColumns []string
	doNothing     bool
}

// DoNothing 忽略冲突
func (c *ConflictBuilder) DoNothing() *ConflictBuilder {
	c.doNothing = true
	return c
}

// DoUpdate 更新指定列
func (c *ConflictBuilder) DoUpdate(columns ...string) *ConflictBuilder {
	c.updateColumns = columns
	return c
}

// Create 生成带冲突处理的 INSERT 语句
func (c *ConflictBuilder) Create(value interface{}) (string, error) {
	sql, err := c.generator.Create(value)
	if err != nil {
		return "", err
	}

	// 移除末尾的分号
	sql = strings.TrimSuffix(sql, ";")

	switch c.generator.dialect.Name() {
	case MySQL:
		if c.doNothing {
			sql += " ON DUPLICATE KEY UPDATE " + c.generator.dialect.Quote(c.columns[0]) + " = " + c.generator.dialect.Quote(c.columns[0])
		} else if len(c.updateColumns) > 0 {
			updates := make([]string, len(c.updateColumns))
			for i, col := range c.updateColumns {
				quotedCol := c.generator.dialect.Quote(toSnakeCase(col))
				updates[i] = fmt.Sprintf("%s = VALUES(%s)", quotedCol, quotedCol)
			}
			sql += " ON DUPLICATE KEY UPDATE " + strings.Join(updates, ", ")
		}
	case PostgreSQL:
		quotedCols := make([]string, len(c.columns))
		for i, col := range c.columns {
			quotedCols[i] = c.generator.dialect.Quote(toSnakeCase(col))
		}
		sql += " ON CONFLICT (" + strings.Join(quotedCols, ", ") + ")"

		if c.doNothing {
			sql += " DO NOTHING"
		} else if len(c.updateColumns) > 0 {
			updates := make([]string, len(c.updateColumns))
			for i, col := range c.updateColumns {
				quotedCol := c.generator.dialect.Quote(toSnakeCase(col))
				updates[i] = fmt.Sprintf("%s = EXCLUDED.%s", quotedCol, quotedCol)
			}
			sql += " DO UPDATE SET " + strings.Join(updates, ", ")
		}
	case SQLite:
		quotedCols := make([]string, len(c.columns))
		for i, col := range c.columns {
			quotedCols[i] = c.generator.dialect.Quote(toSnakeCase(col))
		}
		sql += " ON CONFLICT (" + strings.Join(quotedCols, ", ") + ")"

		if c.doNothing {
			sql += " DO NOTHING"
		} else if len(c.updateColumns) > 0 {
			updates := make([]string, len(c.updateColumns))
			for i, col := range c.updateColumns {
				quotedCol := c.generator.dialect.Quote(toSnakeCase(col))
				updates[i] = fmt.Sprintf("%s = excluded.%s", quotedCol, quotedCol)
			}
			sql += " DO UPDATE SET " + strings.Join(updates, ", ")
		}
	}

	return sql + ";", nil
}
