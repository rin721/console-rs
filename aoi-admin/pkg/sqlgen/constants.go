package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

// ============================================================================
// 方言类型 (Dialect Types)
// ============================================================================

// Dialect 表示 SQL 数据库方言类型
type Dialect string

const (
	// MySQL MySQL 数据库方言
	MySQL Dialect = "mysql"
	// PostgreSQL PostgreSQL 数据库方言
	PostgreSQL Dialect = "postgres"
	// SQLite SQLite 数据库方言
	SQLite Dialect = "sqlite"
	// SQLServer SQL Server 数据库方言
	SQLServer Dialect = "sqlserver"
)

// ============================================================================
// Tag 类型标志位 (Tag Type Flags)
// ============================================================================

// TagType 表示生成 Struct Tag 的类型标志位
// 使用位运算支持多个 Tag 组合
type TagType uint

const (
	// TagGorm 生成 gorm Tag
	TagGorm TagType = 1 << iota
	// TagJson 生成 json Tag
	TagJson
	// TagXml 生成 xml Tag
	TagXml
	// TagYaml 生成 yaml Tag
	TagYaml
	// TagValidate 生成 validate Tag
	TagValidate
	// TagAll 生成所有 Tag
	TagAll = TagGorm | TagJson | TagXml | TagYaml | TagValidate
	// TagDefault 默认生成 gorm 和 json Tag
	TagDefault = TagGorm | TagJson
)

// ============================================================================
// 命名策略 (Naming Strategies)
// ============================================================================

// NamingStrategy 表示命名风格策略
type NamingStrategy int

const (
	// SnakeCase 蛇形命名 (user_name)
	SnakeCase NamingStrategy = iota
	// CamelCase 小驼峰命名 (userName)
	CamelCase
	// PascalCase 大驼峰命名 (UserName)
	PascalCase
	// KebabCase 短横线命名 (user-name)
	KebabCase
)

// ============================================================================
// SQL 操作类型 (SQL Operation Types)
// ============================================================================

// OperationType 表示 SQL 操作类型
type OperationType int

const (
	// OpSelect SELECT 查询
	OpSelect OperationType = iota
	// OpInsert INSERT 插入
	OpInsert
	// OpUpdate UPDATE 更新
	OpUpdate
	// OpDelete DELETE 删除
	OpDelete
	// OpCreateTable 表示 CREATE TABLE 操作。
	OpCreateTable
	// OpDropTable 表示 DROP TABLE 操作。
	OpDropTable
	// OpAlterTable 表示 ALTER TABLE 操作。
	OpAlterTable
	// OpCreateIndex 表示 CREATE INDEX 操作。
	OpCreateIndex
	// OpDropIndex 表示 DROP INDEX 操作。
	OpDropIndex
)

// ============================================================================
// 默认值常量 (Default Values)
// ============================================================================

const (
	// DefaultMaxBatchSize 默认批量操作最大数量
	DefaultMaxBatchSize = 1000
	// DefaultIndent 默认缩进字符串
	DefaultIndent = "  "
	// DefaultSoftDeleteColumn 默认软删除列名
	DefaultSoftDeleteColumn = "deleted_at"
	// DefaultCreatedAtColumn 默认创建时间列名
	DefaultCreatedAtColumn = "created_at"
	// DefaultUpdatedAtColumn 默认更新时间列名
	DefaultUpdatedAtColumn = "updated_at"
)

// ============================================================================
// GORM Tag 键名 (GORM Tag Keys)
// ============================================================================

const (
	// GormTagColumn 列名
	GormTagColumn = "column"
	// GormTagPrimaryKey 主键
	GormTagPrimaryKey = "primaryKey"
	// GormTagAutoIncrement 自增
	GormTagAutoIncrement = "autoIncrement"
	// GormTagNotNull 非空
	GormTagNotNull = "not null"
	// GormTagDefault 默认值
	GormTagDefault = "default"
	// GormTagSize 大小
	GormTagSize = "size"
	// GormTagType 类型
	GormTagType = "type"
	// GormTagIndex 索引
	GormTagIndex = "index"
	// GormTagUniqueIndex 唯一索引
	GormTagUniqueIndex = "uniqueIndex"
	// GormTagComment 注释
	GormTagComment = "comment"
)
