package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

import (
	"fmt"
	"reflect"
	"strings"
)

// ============================================================================
// 方言接口
// ============================================================================

// DialectHandler 定义数据库方言处理器接口
type DialectHandler interface {
	// Name 返回方言名称
	Name() Dialect

	// Quote 为标识符添加引号 (如 `name` 或 "name")
	Quote(name string) string

	// Placeholder 返回参数占位符 (如 ? 或 $1)
	Placeholder(index int) string

	// TypeMapping 将 Go 类型映射到 SQL 类型
	TypeMapping(goType string, size int) string

	// ReverseTypeMapping 将 SQL 类型映射到 Go 类型
	ReverseTypeMapping(sqlType string) string

	// Interpolate 将参数插入到 SQL 中
	Interpolate(sql string, args ...interface{}) (string, error)

	// AutoIncrementKeyword 返回自增关键字
	AutoIncrementKeyword() string

	// DefaultValueKeyword 返回默认值关键字
	DefaultValueKeyword() string

	// EngineClause 返回引擎子句 (MySQL 专用)
	EngineClause() string
}

// ============================================================================
// MySQL 方言
// ============================================================================

type mysqlDialect struct{}

// Name 返回当前 SQL 方言在生成器配置中的稳定标识。
func (d *mysqlDialect) Name() Dialect { return MySQL }

// Quote 按当前 SQL 方言规则包裹标识符，避免列名或表名与关键字冲突。
func (d *mysqlDialect) Quote(name string) string {
	return "`" + name + "`"
}

// Placeholder 返回当前位置参数的占位符文本，用于后续插值或调试输出。
func (d *mysqlDialect) Placeholder(index int) string {
	return "?"
}

// TypeMapping 将 Go 类型和字段约束映射为当前方言可接受的列类型。
func (d *mysqlDialect) TypeMapping(goType string, size int) string {
	switch goType {
	case "string":
		if size > 0 && size <= 255 {
			return fmt.Sprintf("VARCHAR(%d)", size)
		}
		return "TEXT"
	case "int", "int32":
		return "INT"
	case "int8":
		return "TINYINT"
	case "int16":
		return "SMALLINT"
	case "int64":
		return "BIGINT"
	case "uint", "uint32":
		return "INT UNSIGNED"
	case "uint8":
		return "TINYINT UNSIGNED"
	case "uint16":
		return "SMALLINT UNSIGNED"
	case "uint64":
		return "BIGINT UNSIGNED"
	case "float32":
		return "FLOAT"
	case "float64":
		return "DOUBLE"
	case "bool":
		return "TINYINT(1)"
	case "time.Time", "*time.Time":
		return "DATETIME"
	case "[]byte":
		return "BLOB"
	default:
		return "TEXT"
	}
}

// ReverseTypeMapping 将数据库列类型反向映射为 Go 字段类型，用于 DDL 解析和代码生成。
func (d *mysqlDialect) ReverseTypeMapping(sqlType string) string {
	upper := strings.ToUpper(sqlType)

	// 移除括号内的内容用于匹配
	baseType := upper
	if idx := strings.Index(upper, "("); idx > 0 {
		baseType = upper[:idx]
	}

	switch baseType {
	case "TINYINT":
		if strings.Contains(upper, "(1)") {
			return "bool"
		}
		if strings.Contains(upper, "UNSIGNED") {
			return "uint8"
		}
		return "int8"
	case "SMALLINT":
		if strings.Contains(upper, "UNSIGNED") {
			return "uint16"
		}
		return "int16"
	case "INT", "INTEGER", "MEDIUMINT":
		if strings.Contains(upper, "UNSIGNED") {
			return "uint32"
		}
		return "int32"
	case "BIGINT":
		if strings.Contains(upper, "UNSIGNED") {
			return "uint64"
		}
		return "int64"
	case "FLOAT":
		return "float32"
	case "DOUBLE", "DECIMAL", "NUMERIC":
		return "float64"
	case "CHAR", "VARCHAR", "TEXT", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT":
		return "string"
	case "DATETIME", "TIMESTAMP", "DATE", "TIME":
		return "time.Time"
	case "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB", "BINARY", "VARBINARY":
		return "[]byte"
	case "JSON":
		return "json.RawMessage"
	case "BOOL", "BOOLEAN":
		return "bool"
	default:
		return "string"
	}
}

// Interpolate 将参数值写入 SQL 文本，主要服务生成预览而不是替代数据库驱动参数化执行。
func (d *mysqlDialect) Interpolate(sql string, args ...interface{}) (string, error) {
	return interpolateSQL(sql, args, d.Quote)
}

// AutoIncrementKeyword 返回当前方言声明自增列时使用的关键字。
func (d *mysqlDialect) AutoIncrementKeyword() string {
	return "AUTO_INCREMENT"
}

// DefaultValueKeyword 返回当前方言声明默认值时使用的 SQL 片段。
func (d *mysqlDialect) DefaultValueKeyword() string {
	return "DEFAULT"
}

// EngineClause 返回当前方言的表级存储引擎子句，不需要时返回空字符串。
func (d *mysqlDialect) EngineClause() string {
	return "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"
}

// ============================================================================
// PostgreSQL 方言
// ============================================================================

type postgresDialect struct{}

// Name 返回当前 SQL 方言在生成器配置中的稳定标识。
func (d *postgresDialect) Name() Dialect { return PostgreSQL }

// Quote 按当前 SQL 方言规则包裹标识符，避免列名或表名与关键字冲突。
func (d *postgresDialect) Quote(name string) string {
	return "\"" + name + "\""
}

// Placeholder 返回当前位置参数的占位符文本，用于后续插值或调试输出。
func (d *postgresDialect) Placeholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

// TypeMapping 将 Go 类型和字段约束映射为当前方言可接受的列类型。
func (d *postgresDialect) TypeMapping(goType string, size int) string {
	switch goType {
	case "string":
		if size > 0 && size <= 255 {
			return fmt.Sprintf("VARCHAR(%d)", size)
		}
		return "TEXT"
	case "int", "int32":
		return "INTEGER"
	case "int8", "int16":
		return "SMALLINT"
	case "int64":
		return "BIGINT"
	case "uint", "uint32":
		return "INTEGER"
	case "uint8", "uint16":
		return "SMALLINT"
	case "uint64":
		return "BIGINT"
	case "float32":
		return "REAL"
	case "float64":
		return "DOUBLE PRECISION"
	case "bool":
		return "BOOLEAN"
	case "time.Time", "*time.Time":
		return "TIMESTAMP"
	case "[]byte":
		return "BYTEA"
	default:
		return "TEXT"
	}
}

// ReverseTypeMapping 将数据库列类型反向映射为 Go 字段类型，用于 DDL 解析和代码生成。
func (d *postgresDialect) ReverseTypeMapping(sqlType string) string {
	upper := strings.ToUpper(sqlType)

	baseType := upper
	if idx := strings.Index(upper, "("); idx > 0 {
		baseType = upper[:idx]
	}

	switch baseType {
	case "SMALLINT", "INT2":
		return "int16"
	case "INTEGER", "INT", "INT4":
		return "int32"
	case "BIGINT", "INT8":
		return "int64"
	case "REAL", "FLOAT4":
		return "float32"
	case "DOUBLE PRECISION", "FLOAT8", "NUMERIC", "DECIMAL":
		return "float64"
	case "CHAR", "VARCHAR", "TEXT", "CHARACTER VARYING":
		return "string"
	case "TIMESTAMP", "TIMESTAMPTZ", "DATE", "TIME", "TIMETZ":
		return "time.Time"
	case "BYTEA":
		return "[]byte"
	case "BOOLEAN", "BOOL":
		return "bool"
	case "JSON", "JSONB":
		return "json.RawMessage"
	case "SERIAL":
		return "int32"
	case "BIGSERIAL":
		return "int64"
	case "UUID":
		return "string"
	default:
		return "string"
	}
}

// Interpolate 将参数值写入 SQL 文本，主要服务生成预览而不是替代数据库驱动参数化执行。
func (d *postgresDialect) Interpolate(sql string, args ...interface{}) (string, error) {
	return interpolateSQLPositional(sql, args, d.Quote)
}

// AutoIncrementKeyword 返回当前方言声明自增列时使用的关键字。
func (d *postgresDialect) AutoIncrementKeyword() string {
	return "" // PostgreSQL 使用 SERIAL 类型
}

// DefaultValueKeyword 返回当前方言声明默认值时使用的 SQL 片段。
func (d *postgresDialect) DefaultValueKeyword() string {
	return "DEFAULT"
}

// EngineClause 返回当前方言的表级存储引擎子句，不需要时返回空字符串。
func (d *postgresDialect) EngineClause() string {
	return "" // PostgreSQL 不需要
}

// ============================================================================
// SQLite 方言
// ============================================================================

type sqliteDialect struct{}

// Name 返回当前 SQL 方言在生成器配置中的稳定标识。
func (d *sqliteDialect) Name() Dialect { return SQLite }

// Quote 按当前 SQL 方言规则包裹标识符，避免列名或表名与关键字冲突。
func (d *sqliteDialect) Quote(name string) string {
	return "\"" + name + "\""
}

// Placeholder 返回当前位置参数的占位符文本，用于后续插值或调试输出。
func (d *sqliteDialect) Placeholder(index int) string {
	return "?"
}

// TypeMapping 将 Go 类型和字段约束映射为当前方言可接受的列类型。
func (d *sqliteDialect) TypeMapping(goType string, size int) string {
	switch goType {
	case "string":
		return "TEXT"
	case "int", "int8", "int16", "int32", "int64":
		return "INTEGER"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "INTEGER"
	case "float32", "float64":
		return "REAL"
	case "bool":
		return "INTEGER"
	case "time.Time", "*time.Time":
		return "DATETIME"
	case "[]byte":
		return "BLOB"
	default:
		return "TEXT"
	}
}

// ReverseTypeMapping 将数据库列类型反向映射为 Go 字段类型，用于 DDL 解析和代码生成。
func (d *sqliteDialect) ReverseTypeMapping(sqlType string) string {
	upper := strings.ToUpper(sqlType)

	switch {
	case strings.Contains(upper, "INT"):
		return "int64"
	case strings.Contains(upper, "CHAR"), strings.Contains(upper, "TEXT"), strings.Contains(upper, "CLOB"):
		return "string"
	case strings.Contains(upper, "REAL"), strings.Contains(upper, "FLOA"), strings.Contains(upper, "DOUB"):
		return "float64"
	case strings.Contains(upper, "BLOB"):
		return "[]byte"
	case upper == "DATETIME", upper == "DATE", upper == "TIMESTAMP":
		return "time.Time"
	case upper == "BOOLEAN":
		return "bool"
	default:
		return "string"
	}
}

// Interpolate 将参数值写入 SQL 文本，主要服务生成预览而不是替代数据库驱动参数化执行。
func (d *sqliteDialect) Interpolate(sql string, args ...interface{}) (string, error) {
	return interpolateSQL(sql, args, d.Quote)
}

// AutoIncrementKeyword 返回当前方言声明自增列时使用的关键字。
func (d *sqliteDialect) AutoIncrementKeyword() string {
	return "AUTOINCREMENT"
}

// DefaultValueKeyword 返回当前方言声明默认值时使用的 SQL 片段。
func (d *sqliteDialect) DefaultValueKeyword() string {
	return "DEFAULT"
}

// EngineClause 返回当前方言的表级存储引擎子句，不需要时返回空字符串。
func (d *sqliteDialect) EngineClause() string {
	return "" // SQLite 不需要
}

// ============================================================================
// SQL Server 方言
// ============================================================================

type sqlserverDialect struct{}

// Name 返回当前 SQL 方言在生成器配置中的稳定标识。
func (d *sqlserverDialect) Name() Dialect { return SQLServer }

// Quote 按当前 SQL 方言规则包裹标识符，避免列名或表名与关键字冲突。
func (d *sqlserverDialect) Quote(name string) string {
	return "[" + name + "]"
}

// Placeholder 返回当前位置参数的占位符文本，用于后续插值或调试输出。
func (d *sqlserverDialect) Placeholder(index int) string {
	return fmt.Sprintf("@p%d", index)
}

// TypeMapping 将 Go 类型和字段约束映射为当前方言可接受的列类型。
func (d *sqlserverDialect) TypeMapping(goType string, size int) string {
	switch goType {
	case "string":
		if size > 0 && size <= 4000 {
			return fmt.Sprintf("NVARCHAR(%d)", size)
		}
		return "NVARCHAR(MAX)"
	case "int", "int32":
		return "INT"
	case "int8":
		return "TINYINT"
	case "int16":
		return "SMALLINT"
	case "int64":
		return "BIGINT"
	case "uint", "uint32":
		return "INT"
	case "uint8":
		return "TINYINT"
	case "uint16":
		return "SMALLINT"
	case "uint64":
		return "BIGINT"
	case "float32":
		return "REAL"
	case "float64":
		return "FLOAT"
	case "bool":
		return "BIT"
	case "time.Time", "*time.Time":
		return "DATETIME2"
	case "[]byte":
		return "VARBINARY(MAX)"
	default:
		return "NVARCHAR(MAX)"
	}
}

// ReverseTypeMapping 将数据库列类型反向映射为 Go 字段类型，用于 DDL 解析和代码生成。
func (d *sqlserverDialect) ReverseTypeMapping(sqlType string) string {
	upper := strings.ToUpper(sqlType)

	baseType := upper
	if idx := strings.Index(upper, "("); idx > 0 {
		baseType = upper[:idx]
	}

	switch baseType {
	case "TINYINT":
		return "uint8"
	case "SMALLINT":
		return "int16"
	case "INT":
		return "int32"
	case "BIGINT":
		return "int64"
	case "REAL":
		return "float32"
	case "FLOAT":
		return "float64"
	case "CHAR", "VARCHAR", "NCHAR", "NVARCHAR", "TEXT", "NTEXT":
		return "string"
	case "DATETIME", "DATETIME2", "DATE", "TIME", "SMALLDATETIME":
		return "time.Time"
	case "BINARY", "VARBINARY", "IMAGE":
		return "[]byte"
	case "BIT":
		return "bool"
	case "UNIQUEIDENTIFIER":
		return "string"
	default:
		return "string"
	}
}

// Interpolate 将参数值写入 SQL 文本，主要服务生成预览而不是替代数据库驱动参数化执行。
func (d *sqlserverDialect) Interpolate(sql string, args ...interface{}) (string, error) {
	return interpolateSQLPositional(sql, args, d.Quote)
}

// AutoIncrementKeyword 返回当前方言声明自增列时使用的关键字。
func (d *sqlserverDialect) AutoIncrementKeyword() string {
	return "IDENTITY(1,1)"
}

// DefaultValueKeyword 返回当前方言声明默认值时使用的 SQL 片段。
func (d *sqlserverDialect) DefaultValueKeyword() string {
	return "DEFAULT"
}

// EngineClause 返回当前方言的表级存储引擎子句，不需要时返回空字符串。
func (d *sqlserverDialect) EngineClause() string {
	return "" // SQL Server 不需要
}

// ============================================================================
// 方言注册表
// ============================================================================

var dialects = map[Dialect]DialectHandler{
	MySQL:      &mysqlDialect{},
	PostgreSQL: &postgresDialect{},
	SQLite:     &sqliteDialect{},
	SQLServer:  &sqlserverDialect{},
}

// getDialect 获取方言处理器
func getDialect(d Dialect) DialectHandler {
	if handler, ok := dialects[d]; ok {
		return handler
	}
	return &mysqlDialect{} // 默认返回 MySQL
}

// RegisterDialect 注册自定义方言
func RegisterDialect(d Dialect, handler DialectHandler) {
	dialects[d] = handler
}

// ============================================================================
// SQL 插值辅助函数
// ============================================================================

// interpolateSQL 将 ? 占位符替换为实际值 (MySQL/SQLite 风格)
func interpolateSQL(sql string, args []interface{}, quoter func(string) string) (string, error) {
	if len(args) == 0 {
		return sql, nil
	}

	var result strings.Builder
	argIndex := 0

	for i := 0; i < len(sql); i++ {
		if sql[i] == '?' && argIndex < len(args) {
			result.WriteString(formatValue(args[argIndex], quoter))
			argIndex++
		} else {
			result.WriteByte(sql[i])
		}
	}

	return result.String(), nil
}

// interpolateSQLPositional 将 $1, $2 等占位符替换为实际值 (PostgreSQL 风格)
func interpolateSQLPositional(sql string, args []interface{}, quoter func(string) string) (string, error) {
	if len(args) == 0 {
		return sql, nil
	}

	result := sql
	for i, arg := range args {
		placeholder := fmt.Sprintf("$%d", i+1)
		result = strings.Replace(result, placeholder, formatValue(arg, quoter), 1)
	}

	return result, nil
}

// formatValue 格式化值为 SQL 字符串
func formatValue(v interface{}, quoter func(string) string) string {
	if v == nil {
		return "NULL"
	}

	rv := reflect.ValueOf(v)

	// 处理指针
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return "NULL"
		}
		rv = rv.Elem()
		v = rv.Interface()
	}

	switch val := v.(type) {
	case string:
		return "'" + escapeString(val) + "'"
	case bool:
		if val {
			return "1"
		}
		return "0"
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", val)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%v", val)
	case []byte:
		return fmt.Sprintf("X'%x'", val)
	case *Expr:
		// 表达式，递归插值
		interpolated, _ := interpolateSQL(val.SQL, val.Vars, quoter)
		return interpolated
	default:
		// 时间类型和其他
		return fmt.Sprintf("'%v'", val)
	}
}

// escapeString 转义 SQL 字符串中的特殊字符
func escapeString(s string) string {
	s = strings.ReplaceAll(s, "'", "''")
	s = strings.ReplaceAll(s, "\\", "\\\\")
	return s
}
