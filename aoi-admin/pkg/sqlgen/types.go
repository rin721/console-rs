package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

import (
	"reflect"
	"time"
)

// ============================================================================
// 配置类型 (Configuration Types)
// ============================================================================

// Config 保存 SQL 生成器的配置
type Config struct {
	// Dialect 数据库方言类型 (MySQL, PostgreSQL, SQLite, SQLServer)
	Dialect Dialect

	// Pretty 是否格式化输出 (添加缩进和换行)
	Pretty bool

	// SkipZeroValue 是否跳过零值字段 (用于 UPDATE)
	SkipZeroValue bool

	// SoftDelete 是否启用软删除 (自动检测 deleted_at 字段)
	SoftDelete bool

	// AllowEmptyCondition 是否允许无条件的 UPDATE/DELETE
	AllowEmptyCondition bool
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Dialect:             MySQL,
		Pretty:              false,
		SkipZeroValue:       true,
		SoftDelete:          true,
		AllowEmptyCondition: false,
	}
}

// ============================================================================
// Schema 类型 (DDL 解析结果)
// ============================================================================

// Schema 表示解析后的表结构
type Schema struct {
	// Name 结构体名称 (PascalCase)
	Name string

	// TableName 表名 (snake_case)
	TableName string

	// Fields 字段列表
	Fields []Field

	// Comment 表注释
	Comment string

	// Indexes 索引列表
	Indexes []Index

	// Package 包名 (用于代码生成)
	Package string

	// Imports 需要导入的包
	Imports []string
}

// Field 表示结构体字段
type Field struct {
	// Name Go 字段名 (PascalCase)
	Name string

	// Type Go 类型 (如 string, int64, *time.Time)
	Type string

	// Tags 完整的 struct tag 字符串
	Tags string

	// Column 对应的数据库列信息
	Column Column

	// Comment 字段注释
	Comment string
}

// Column 表示数据库列定义
type Column struct {
	// Name 列名
	Name string

	// Type SQL 数据类型 (如 VARCHAR(64), BIGINT)
	Type string

	// GoType 对应的 Go 类型
	GoType string

	// PrimaryKey 是否为主键
	PrimaryKey bool

	// AutoIncrement 是否自增
	AutoIncrement bool

	// NotNull 是否非空
	NotNull bool

	// Default 默认值
	Default string

	// Comment 列注释
	Comment string

	// Size 大小限制 (用于 VARCHAR 等)
	Size int

	// Precision 精度 (用于 DECIMAL 等)
	Precision int

	// Scale 小数位数 (用于 DECIMAL 等)
	Scale int
}

// Index 表示数据库索引定义
type Index struct {
	// Name 索引名
	Name string

	// Columns 索引包含的列
	Columns []string

	// Unique 是否为唯一索引
	Unique bool

	// Type 索引类型 (BTREE, HASH 等)
	Type string
}

// ============================================================================
// 查询上下文 (Query Context)
// ============================================================================

// QueryContext 保存链式调用的上下文状态
type QueryContext struct {
	// Model 模型实例
	Model interface{}

	// ModelType 模型的反射类型
	ModelType reflect.Type

	// ModelValue 模型的反射值
	ModelValue reflect.Value

	// TableName 表名
	TableName string

	// Operation 操作类型
	Operation OperationType

	// SelectColumns 选择的列
	SelectColumns []string

	// OmitColumns 忽略的列
	OmitColumns []string

	// WhereConditions WHERE 条件
	WhereConditions []WhereCondition

	// OrderBy ORDER BY 子句
	OrderBy string

	// Limit LIMIT 子句
	Limit int

	// Offset OFFSET 子句
	Offset int

	// Unscoped 是否忽略软删除
	Unscoped bool

	// Updates 更新的值 (用于 UPDATE)
	Updates interface{}

	// Returning RETURNING 子句 (PostgreSQL)
	Returning []string

	// UnsupportedOperation 记录当前链式调用中尚未支持的操作。
	UnsupportedOperation string
}

// WhereCondition 表示 WHERE 条件
type WhereCondition struct {
	// Query 条件表达式 (可以是字符串或结构体)
	Query interface{}

	// Args 参数列表
	Args []interface{}
}

// ============================================================================
// 逆向生成配置 (Reverse Builder Options)
// ============================================================================

// ReverseOptions 保存逆向生成的配置选项
type ReverseOptions struct {
	// StructName 自定义结构体名称
	StructName string

	// Package 包名
	Package string

	// Tags 要生成的 Tag 类型
	Tags TagType

	// JSONNaming JSON Tag 命名策略
	JSONNaming NamingStrategy

	// FieldNaming 字段命名策略
	FieldNaming NamingStrategy

	// TypeMappings 自定义类型映射 (SQL type -> Go type)
	TypeMappings map[string]string

	// WithComments 是否生成注释
	WithComments bool

	// WithTableName 是否生成 TableName() 方法
	WithTableName bool

	// WithSoftDelete 是否识别软删除字段
	WithSoftDelete bool

	// Imports 额外导入的包
	Imports []string

	// Template 自定义模板
	Template string

	// FieldConverter 自定义字段转换器
	FieldConverter func(col Column) Field

	// BeforeGenerate 生成前钩子
	BeforeGenerate func(schema *Schema)

	// AfterGenerate 生成后钩子
	AfterGenerate func(code string) string

	// FileNaming 文件命名策略
	FileNaming NamingStrategy

	// Overwrite 是否覆盖已存在的文件
	Overwrite bool
}

// DefaultReverseOptions 返回默认逆向生成选项
func DefaultReverseOptions() *ReverseOptions {
	return &ReverseOptions{
		Package:        "models",
		Tags:           TagDefault,
		JSONNaming:     SnakeCase,
		FieldNaming:    PascalCase,
		TypeMappings:   make(map[string]string),
		WithComments:   true,
		WithTableName:  true,
		WithSoftDelete: true,
		FileNaming:     SnakeCase,
		Overwrite:      false,
	}
}

// ============================================================================
// 辅助类型 (Helper Types)
// ============================================================================

// Expr 表示 SQL 表达式 (类似 gorm.Expr)
type Expr struct {
	// SQL 表达式字符串
	SQL string

	// Vars 表达式参数
	Vars []interface{}
}

// NewExpr 创建新的 SQL 表达式
func NewExpr(sql string, vars ...interface{}) *Expr {
	return &Expr{
		SQL:  sql,
		Vars: vars,
	}
}

// ============================================================================
// 时间类型 (Time Types)
// ============================================================================

// DeletedAt 软删除时间类型 (可为 nil)
type DeletedAt *time.Time

// IsDeleted 判断是否已删除
func IsDeleted(t DeletedAt) bool {
	return t != nil
}
