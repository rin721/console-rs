package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

import (
	"reflect"
	"strings"
	"sync"
)

// ============================================================================
// Generator 主结构
// ============================================================================

// Generator SQL 生成器主结构
// 提供类似 GORM 的链式 API，但返回 SQL 字符串而非执行 SQL
type Generator struct {
	config  *Config
	dialect DialectHandler
	ctx     *QueryContext
	mu      sync.RWMutex
}

// New 创建新的 SQL 生成器
func New(cfg *Config) *Generator {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	g := &Generator{
		config: cfg,
		ctx:    &QueryContext{},
	}

	// 设置方言处理器
	g.dialect = getDialect(cfg.Dialect)

	return g
}

// clone 克隆生成器 (用于链式调用)
func (g *Generator) clone() *Generator {
	g.mu.RLock()
	defer g.mu.RUnlock()

	newCtx := *g.ctx
	return &Generator{
		config:  g.config,
		dialect: g.dialect,
		ctx:     &newCtx,
	}
}

// ============================================================================
// 模型设置方法
// ============================================================================

// Model 设置模型 (用于不需要数据的操作，如 COUNT)
func (g *Generator) Model(model interface{}) *Generator {
	ng := g.clone()
	ng.ctx.Model = model

	// 解析模型信息
	if err := ng.parseModel(model); err != nil {
		// 错误会在后续操作中处理
		return ng
	}

	return ng
}

// parseModel 解析模型信息
func (g *Generator) parseModel(model interface{}) error {
	if model == nil {
		return ErrInvalidModel
	}

	t := reflect.TypeOf(model)
	v := reflect.ValueOf(model)

	// 处理指针
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	// 处理切片
	if t.Kind() == reflect.Slice {
		t = t.Elem()
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
	}

	// 必须是结构体
	if t.Kind() != reflect.Struct {
		return ErrInvalidModel
	}

	g.ctx.ModelType = t
	g.ctx.ModelValue = v
	g.ctx.TableName = g.getTableName(model, t)

	return nil
}

// getTableName 获取表名
func (g *Generator) getTableName(model interface{}, t reflect.Type) string {
	// 1. 检查是否实现了 Tabler 接口
	if tabler, ok := model.(Tabler); ok {
		return tabler.TableName()
	}

	// 2. 从类型名推断
	name := t.Name()
	return toSnakeCase(name) + "s" // 简单的复数化
}

// ============================================================================
// 查询条件方法 (链式调用)
// ============================================================================

// Select 选择特定列
func (g *Generator) Select(columns ...string) *Generator {
	ng := g.clone()
	ng.ctx.SelectColumns = columns
	return ng
}

// Omit 忽略特定列
func (g *Generator) Omit(columns ...string) *Generator {
	ng := g.clone()
	ng.ctx.OmitColumns = columns
	return ng
}

// Where 添加 WHERE 条件
func (g *Generator) Where(query interface{}, args ...interface{}) *Generator {
	ng := g.clone()
	ng.ctx.WhereConditions = append(ng.ctx.WhereConditions, WhereCondition{
		Query: query,
		Args:  args,
	})
	return ng
}

// Order 设置排序
func (g *Generator) Order(value string) *Generator {
	ng := g.clone()
	ng.ctx.OrderBy = value
	return ng
}

// Limit 设置 LIMIT
func (g *Generator) Limit(n int) *Generator {
	ng := g.clone()
	ng.ctx.Limit = n
	return ng
}

// Offset 设置 OFFSET
func (g *Generator) Offset(n int) *Generator {
	ng := g.clone()
	ng.ctx.Offset = n
	return ng
}

// Unscoped 忽略软删除条件
func (g *Generator) Unscoped() *Generator {
	ng := g.clone()
	ng.ctx.Unscoped = true
	return ng
}

// unsupported 标记当前链路包含尚未支持的操作。
func (g *Generator) unsupported(operation string) *Generator {
	ng := g.clone()
	ng.ctx.UnsupportedOperation = operation
	return ng
}

// checkUnsupported 返回当前链路中的 unsupported 操作错误。
func (g *Generator) checkUnsupported() error {
	if g.ctx.UnsupportedOperation == "" {
		return nil
	}
	return NewUnsupportedError(g.ctx.UnsupportedOperation)
}

// ============================================================================
// 原生 SQL
// ============================================================================

// Raw 使用原生 SQL
func (g *Generator) Raw(sql string, args ...interface{}) *RawBuilder {
	return &RawBuilder{
		generator: g,
		sql:       sql,
		args:      args,
	}
}

// RawBuilder 原生 SQL 构建器
type RawBuilder struct {
	generator *Generator
	sql       string
	args      []interface{}
}

// Build 构建 SQL
func (r *RawBuilder) Build() (string, error) {
	return r.generator.dialect.Interpolate(r.sql, r.args...)
}

// ============================================================================
// 接口定义
// ============================================================================

// Tabler 表名接口 (与 GORM 兼容)
type Tabler interface {
	TableName() string
}

// ============================================================================
// 辅助函数
// ============================================================================

// toSnakeCase 将字符串转换为蛇形命名
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// toCamelCase 将字符串转换为小驼峰命名
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// toPascalCase 将字符串转换为大驼峰命名
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 0; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// toKebabCase 将字符串转换为短横线命名
func toKebabCase(s string) string {
	return strings.ReplaceAll(toSnakeCase(s), "_", "-")
}

// convertNaming 根据策略转换命名
func convertNaming(s string, strategy NamingStrategy) string {
	switch strategy {
	case SnakeCase:
		return toSnakeCase(s)
	case CamelCase:
		return toCamelCase(s)
	case PascalCase:
		return toPascalCase(s)
	case KebabCase:
		return toKebabCase(s)
	default:
		return s
	}
}
