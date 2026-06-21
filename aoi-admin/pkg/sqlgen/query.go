package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

import (
	"fmt"
	"reflect"
	"strings"
)

// ============================================================================
// SELECT 生成方法
// ============================================================================

// First 生成查询第一条记录的 SQL
func (g *Generator) First(dest interface{}, conds ...interface{}) (string, error) {
	ng := g.clone()

	if err := ng.parseModel(dest); err != nil {
		return "", err
	}

	ng.ctx.Operation = OpSelect
	ng.ctx.Limit = 1

	// 处理条件参数
	if len(conds) > 0 {
		ng = ng.processConditions(conds...)
	}

	return ng.buildSelect()
}

// Find 生成查询多条记录的 SQL
func (g *Generator) Find(dest interface{}, conds ...interface{}) (string, error) {
	ng := g.clone()

	if err := ng.parseModel(dest); err != nil {
		return "", err
	}

	ng.ctx.Operation = OpSelect

	// 处理条件参数
	if len(conds) > 0 {
		ng = ng.processConditions(conds...)
	}

	return ng.buildSelect()
}

// Count 生成计数查询的 SQL
func (g *Generator) Count(count *int64) (string, error) {
	ng := g.clone()

	if ng.ctx.TableName == "" {
		return "", ErrNoTableName
	}

	ng.ctx.Operation = OpSelect
	ng.ctx.SelectColumns = []string{"count(*)"}

	return ng.buildSelect()
}

// Pluck 生成单列查询的 SQL
func (g *Generator) Pluck(column string, dest interface{}) (string, error) {
	ng := g.clone()

	if ng.ctx.TableName == "" {
		return "", ErrNoTableName
	}

	ng.ctx.Operation = OpSelect
	ng.ctx.SelectColumns = []string{column}

	return ng.buildSelect()
}

// ============================================================================
// 条件处理
// ============================================================================

// processConditions 处理传入的条件参数
func (g *Generator) processConditions(conds ...interface{}) *Generator {
	if len(conds) == 0 {
		return g
	}

	// 第一个参数决定条件类型
	switch cond := conds[0].(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		// 主键查询: First(&user, 1)
		pk := getPrimaryKeyField(parseStructFields(g.ctx.ModelType, g.ctx.ModelValue, g.dialect))
		if pk != nil {
			g.ctx.WhereConditions = append(g.ctx.WhereConditions, WhereCondition{
				Query: pk.ColumnName + " = ?",
				Args:  []interface{}{cond},
			})
		}
	case string:
		// SQL 条件字符串
		g.ctx.WhereConditions = append(g.ctx.WhereConditions, WhereCondition{
			Query: cond,
			Args:  conds[1:],
		})
	default:
		// Struct 条件
		g = g.processStructCondition(cond)
	}

	return g
}

// processStructCondition 处理结构体条件 (自动忽略零值)
func (g *Generator) processStructCondition(cond interface{}) *Generator {
	v := reflect.ValueOf(cond)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	if t.Kind() != reflect.Struct {
		return g
	}

	fields := parseStructFields(t, v, g.dialect)

	for _, field := range fields {
		if !field.IsZero {
			g.ctx.WhereConditions = append(g.ctx.WhereConditions, WhereCondition{
				Query: g.dialect.Quote(g.ctx.TableName) + "." + g.dialect.Quote(field.ColumnName) + " = ?",
				Args:  []interface{}{field.Value},
			})
		}
	}

	return g
}

// ============================================================================
// SELECT 语句构建
// ============================================================================

// buildSelect 依据当前生成上下文和方言规则构造 SQL 片段，错误会向上冒泡给公开构建入口。
func (g *Generator) buildSelect() (string, error) {
	if err := g.checkUnsupported(); err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString("SELECT ")

	// 列
	if len(g.ctx.SelectColumns) > 0 {
		columns := make([]string, len(g.ctx.SelectColumns))
		for i, col := range g.ctx.SelectColumns {
			if col == "count(*)" || strings.Contains(col, "(") || col == "*" {
				columns[i] = col
			} else {
				columns[i] = g.dialect.Quote(toSnakeCase(col))
			}
		}
		sb.WriteString(strings.Join(columns, ", "))
	} else {
		sb.WriteString("*")
	}

	sb.WriteString(" FROM ")
	sb.WriteString(g.dialect.Quote(g.ctx.TableName))

	// 拼接 WHERE 条件。
	whereClause := g.buildWhereClause()
	if whereClause != "" {
		sb.WriteString(" WHERE ")
		sb.WriteString(whereClause)
	}

	// 拼接 ORDER BY 子句。
	if g.ctx.OrderBy != "" {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(g.ctx.OrderBy)
	}

	// 拼接 LIMIT 子句。
	if g.ctx.Limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", g.ctx.Limit))
	}

	// 拼接 OFFSET 子句。
	if g.ctx.Offset > 0 {
		sb.WriteString(fmt.Sprintf(" OFFSET %d", g.ctx.Offset))
	}

	sb.WriteString(";")

	return sb.String(), nil
}

// buildWhereClause 构建 WHERE 子句
func (g *Generator) buildWhereClause() string {
	var conditions []string

	// 处理软删除
	if !g.ctx.Unscoped && g.config.SoftDelete && g.ctx.ModelType != nil {
		if hasSoftDelete(g.ctx.ModelType) {
			conditions = append(conditions, g.dialect.Quote(g.ctx.TableName)+"."+g.dialect.Quote(DefaultSoftDeleteColumn)+" IS NULL")
		}
	}

	// 处理用户条件
	for _, cond := range g.ctx.WhereConditions {
		condStr := g.buildCondition(cond)
		if condStr != "" {
			conditions = append(conditions, condStr)
		}
	}

	return strings.Join(conditions, " AND ")
}

// buildCondition 构建单个条件
func (g *Generator) buildCondition(cond WhereCondition) string {
	switch query := cond.Query.(type) {
	case string:
		// 字符串条件，替换占位符
		sql, _ := g.dialect.Interpolate(query, cond.Args...)
		return sql
	default:
		return ""
	}
}

// ============================================================================
// 高级查询
// ============================================================================

// Or 添加 OR 条件
func (g *Generator) Or(query interface{}, args ...interface{}) *Generator {
	return g.unsupported("Or")
}

// Not 添加 NOT 条件
func (g *Generator) Not(query interface{}, args ...interface{}) *Generator {
	return g.unsupported("Not")
}

// Group 设置 GROUP BY
func (g *Generator) Group(column string) *Generator {
	return g.unsupported("Group")
}

// Having 设置 HAVING
func (g *Generator) Having(query interface{}, args ...interface{}) *Generator {
	return g.unsupported("Having")
}

// Distinct 设置 DISTINCT
func (g *Generator) Distinct(columns ...string) *Generator {
	return g.unsupported("Distinct")
}

// ============================================================================
// Join 查询
// ============================================================================

// JoinBuilder Join 构建器
type JoinBuilder struct {
	generator *Generator
	joinType  string
	table     string
	condition string
}

// Joins 添加 JOIN
func (g *Generator) Joins(query string, args ...interface{}) *Generator {
	return g.unsupported("Joins")
}
