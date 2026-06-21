package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

import "fmt"

// ============================================================================
// 错误类型 (Error Types)
// ============================================================================

// Error 表示 sqlgen 包的错误类型
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

// Error 实现 error 接口
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("sqlgen: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("sqlgen: %s", e.Message)
}

// Unwrap 返回底层错误
func (e *Error) Unwrap() error {
	return e.Cause
}

// ============================================================================
// 错误码 (Error Codes)
// ============================================================================

// ErrorCode 表示错误码类型
type ErrorCode int

const (
	// ErrCodeUnknown 未知错误
	ErrCodeUnknown ErrorCode = iota
	// ErrCodeInvalidModel 无效的模型
	ErrCodeInvalidModel
	// ErrCodeInvalidDialect 无效的方言
	ErrCodeInvalidDialect
	// ErrCodeInvalidSQL 无效的 SQL
	ErrCodeInvalidSQL
	// ErrCodeParseFailed 解析失败
	ErrCodeParseFailed
	// ErrCodeReflectFailed 反射失败
	ErrCodeReflectFailed
	// ErrCodeGenerateFailed 生成失败
	ErrCodeGenerateFailed
	// ErrCodeFileIO 文件 I/O 错误
	ErrCodeFileIO
	// ErrCodeMissingCondition 缺少条件
	ErrCodeMissingCondition
	// ErrCodeEmptyData 空数据
	ErrCodeEmptyData
	// ErrCodeUnsupportedOperation 不支持的操作
	ErrCodeUnsupportedOperation
)

// ============================================================================
// 预定义错误 (Predefined Errors)
// ============================================================================

var (
	// ErrInvalidModel 无效的模型 (必须是结构体指针)
	ErrInvalidModel = &Error{
		Code:    ErrCodeInvalidModel,
		Message: "model must be a pointer to struct",
	}

	// ErrInvalidDialect 不支持的数据库方言
	ErrInvalidDialect = &Error{
		Code:    ErrCodeInvalidDialect,
		Message: "unsupported dialect",
	}

	// ErrInvalidSQL 无效的 SQL 语句
	ErrInvalidSQL = &Error{
		Code:    ErrCodeInvalidSQL,
		Message: "invalid SQL statement",
	}

	// ErrParseFailed SQL 解析失败
	ErrParseFailed = &Error{
		Code:    ErrCodeParseFailed,
		Message: "failed to parse SQL",
	}

	// ErrMissingCondition 缺少 WHERE 条件 (UPDATE/DELETE 危险操作)
	ErrMissingCondition = &Error{
		Code:    ErrCodeMissingCondition,
		Message: "missing WHERE condition for UPDATE/DELETE",
	}

	// ErrEmptyData 数据为空
	ErrEmptyData = &Error{
		Code:    ErrCodeEmptyData,
		Message: "data is empty",
	}

	// ErrNoTableName 无法获取表名
	ErrNoTableName = &Error{
		Code:    ErrCodeReflectFailed,
		Message: "cannot determine table name",
	}
)

// ============================================================================
// 错误构造函数 (Error Constructors)
// ============================================================================

// NewError 创建新的错误
func NewError(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// WrapError 包装错误
func WrapError(code ErrorCode, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewUnsupportedError 创建不支持操作错误
func NewUnsupportedError(operation string) *Error {
	return NewError(ErrCodeUnsupportedOperation, fmt.Sprintf("unsupported operation: %s", operation))
}

// IsError 判断是否为 sqlgen 错误
func IsError(err error, code ErrorCode) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == code
	}
	return false
}
