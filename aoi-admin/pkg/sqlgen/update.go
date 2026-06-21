package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

import (
	"fmt"
	"reflect"
	"strings"
)

// ============================================================================
// UPDATE 生成方法
// ============================================================================

// Updates 生成 UPDATE 语句 (支持 Map 和 Struct)
func (g *Generator) Updates(values interface{}) (string, error) {
	ng := g.clone()

	if ng.ctx.TableName == "" {
		return "", ErrNoTableName
	}

	ng.ctx.Operation = OpUpdate

	return ng.buildUpdate(values)
}

// Update 生成单列 UPDATE 语句
func (g *Generator) Update(column string, value interface{}) (string, error) {
	ng := g.clone()

	if ng.ctx.TableName == "" {
		return "", ErrNoTableName
	}

	ng.ctx.Operation = OpUpdate

	updates := map[string]interface{}{
		column: value,
	}

	return ng.buildUpdate(updates)
}

// ============================================================================
// UPDATE 语句构建
// ============================================================================

// buildUpdate 依据当前生成上下文和方言规则构造 SQL 片段，错误会向上冒泡给公开构建入口。
func (g *Generator) buildUpdate(values interface{}) (string, error) {
	if err := g.checkUnsupported(); err != nil {
		return "", err
	}

	var setClause string

	switch v := values.(type) {
	case map[string]interface{}:
		setClause = g.buildSetFromMap(v)
	default:
		// Struct 类型
		setClause = g.buildSetFromStruct(v)
	}

	if setClause == "" {
		return "", ErrEmptyData
	}

	var sb strings.Builder
	sb.WriteString("UPDATE ")
	sb.WriteString(g.dialect.Quote(g.ctx.TableName))
	sb.WriteString(" SET ")
	sb.WriteString(setClause)

	// WHERE 条件
	whereClause := g.buildWhereClause()
	if whereClause != "" {
		sb.WriteString(" WHERE ")
		sb.WriteString(whereClause)
	} else if !g.config.AllowEmptyCondition {
		return "", ErrMissingCondition
	}

	sb.WriteString(";")

	return sb.String(), nil
}

// buildSetFromMap 从 Map 构建 SET 子句
func (g *Generator) buildSetFromMap(values map[string]interface{}) string {
	if len(values) == 0 {
		return ""
	}

	var sets []string
	for column, value := range values {
		colName := g.dialect.Quote(toSnakeCase(column))
		valStr := formatValue(value, g.dialect.Quote)
		sets = append(sets, fmt.Sprintf("%s=%s", colName, valStr))
	}

	return strings.Join(sets, ", ")
}

// buildSetFromStruct 从 Struct 构建 SET 子句
func (g *Generator) buildSetFromStruct(value interface{}) string {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	if t.Kind() != reflect.Struct {
		return ""
	}

	fields := parseStructFields(t, v, g.dialect)

	// 过滤 Select 和 Omit
	fields = g.filterUpdateFields(fields)

	var sets []string
	for _, field := range fields {
		// 默认跳过零值，除非在 Select 列表中
		if g.config.SkipZeroValue && field.IsZero && !g.isSelectedColumn(field.Name) {
			continue
		}

		// 跳过主键
		if field.Tag.PrimaryKey {
			continue
		}

		colName := g.dialect.Quote(field.ColumnName)
		valStr := formatValue(field.Value, g.dialect.Quote)
		sets = append(sets, fmt.Sprintf("%s=%s", colName, valStr))
	}

	return strings.Join(sets, ", ")
}

// filterUpdateFields 过滤更新字段
func (g *Generator) filterUpdateFields(fields []FieldInfo) []FieldInfo {
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
		fieldNameLower := strings.ToLower(field.Name)
		columnNameLower := strings.ToLower(field.ColumnName)

		// 如果在 Omit 列表中，跳过
		if omitMap[fieldNameLower] || omitMap[columnNameLower] {
			continue
		}

		// 如果指定了 Select，只包含选中的字段
		if len(selectMap) > 0 {
			if !selectMap[fieldNameLower] && !selectMap[columnNameLower] {
				continue
			}
		}

		result = append(result, field)
	}

	return result
}

// isSelectedColumn 检查列是否在 Select 列表中
func (g *Generator) isSelectedColumn(column string) bool {
	columnLower := strings.ToLower(column)
	for _, col := range g.ctx.SelectColumns {
		if strings.ToLower(col) == columnLower {
			return true
		}
	}
	return false
}

// ============================================================================
// 批量更新
// ============================================================================

// UpdateColumn 使用表达式更新单列
func (g *Generator) UpdateColumn(column string, value interface{}) (string, error) {
	return g.Update(column, value)
}

// UpdateColumns 使用 Map 更新多列
func (g *Generator) UpdateColumns(values map[string]interface{}) (string, error) {
	return g.Updates(values)
}

// ============================================================================
// 增量更新辅助
// ============================================================================

// Increment 生成自增更新 SQL
func (g *Generator) Increment(column string, value interface{}) (string, error) {
	expr := NewExpr(toSnakeCase(column)+" + ?", value)
	return g.Update(column, expr)
}

// Decrement 生成自减更新 SQL
func (g *Generator) Decrement(column string, value interface{}) (string, error) {
	expr := NewExpr(toSnakeCase(column)+" - ?", value)
	return g.Update(column, expr)
}
