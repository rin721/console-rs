// Package errors 定义了应用程序的错误码和业务错误类型
// 这个包除了 Go 标准库外没有外部依赖
// 设计思想:
// - 错误码分段管理,便于识别错误类型
// - 使用常量定义,避免魔法数字
// - 前后端通过错误码约定错误处理逻辑
package errors

// 本文件定义跨层业务错误码和错误类型，是 API Result 与业务层之间的稳定错误契约。

// 错误码范围划分:
// - 0: 成功(不是错误)
// - 1000-1999: 参数错误 - 客户端输入有误,应该返回 400 Bad Request
// - 2000-2999: 业务错误 - 业务规则不满足,应该返回 422 Unprocessable Entity
// - 3000-3999: 认证/授权错误 - 身份验证失败,应该返回 401 Unauthorized
// - 4000-4999: 资源错误 - 请求的资源不存在,应该返回 404 Not Found
// - 5000-5999: 系统错误 - 服务端内部错误,应该返回 500 Internal Server Error
//
// 这种分段设计的好处:
// - 通过错误码范围快速判断错误类型
// - 便于自动映射到 HTTP 状态码
// - 为未来扩展预留空间(每个类型 1000 个错误码)

const (
	// CodeSuccess 成功状态码
	// 虽然不是错误,但定义为 0 方便统一处理
	// API 响应中使用此码表示请求成功
	CodeSuccess = 0

	// ==================== 参数错误 (1000-1999) ====================
	// 这类错误通常由客户端输入引起,前端应该做好参数验证

	// ErrInvalidParams 通用的参数错误
	// 当不确定具体是哪个参数错误时使用
	ErrInvalidParams = 1000

	// ErrInvalidUsername 用户名格式错误
	// 例如:长度不符、包含非法字符等
	// 前端应该在用户输入时就进行验证
	ErrInvalidUsername = 1001

	// ErrInvalidEmail 邮箱格式错误
	// 例如:缺少@符号、域名格式不正确等
	// 前端应该使用正则表达式验证
	ErrInvalidEmail = 1002

	// ErrInvalidPassword 密码格式错误
	// 例如:长度不足、复杂度不够等
	// 前端应该显示密码强度提示
	ErrInvalidPassword = 1003

	// ==================== 业务错误 (2000-2999) ====================
	// 这类错误表示业务规则不满足,不是技术问题

	// ErrBusinessLogic 通用的业务逻辑错误
	// 当不确定具体是哪种业务错误时使用
	ErrBusinessLogic = 2000

	// ErrDuplicateUsername 用户名已存在
	// 注册时如果用户名重复会返回此错误
	// 前端应该提示用户更换用户名
	ErrDuplicateUsername = 2001

	// ErrDuplicateEmail 邮箱已被注册
	// 注册时如果邮箱重复会返回此错误
	// 前端可以提示用户使用其他邮箱或直接登录
	ErrDuplicateEmail = 2002

	// ==================== 认证/授权错误 (3000-3999) ====================
	// 这类错误涉及用户身份验证和权限控制

	// ErrUnauthorized 未授权/认证失败
	// 例如:密码错误、用户名不存在等
	// 前端应该提示用户检查登录凭证
	ErrUnauthorized = 3000

	// ErrInvalidToken 令牌无效
	// 例如:Token格式错误、签名验证失败等
	// 前端应该清除本地 Token 并跳转到登录页
	ErrInvalidToken = 3001

	// ErrTokenExpired 令牌已过期
	// 前端可以尝试刷新 Token 或要求用户重新登录
	ErrTokenExpired = 3002

	// ErrPermissionDenied 权限不足
	// 用户已登录但没有执行某操作的权限
	// 前端应该禁用或隐藏无权限的功能
	ErrPermissionDenied = 3003

	// ==================== 资源错误 (4000-4999) ====================
	// 这类错误表示请求的资源不存在

	// ErrResourceNotFound 通用的资源不存在错误
	ErrResourceNotFound = 4000

	// ErrUserNotFound 用户不存在
	// 根据 ID、用户名或邮箱查询用户时,如果不存在返回此错误
	// 前端应该提示用户该用户不存在
	ErrUserNotFound = 4001

	// ==================== 系统错误 (5000-5999) ====================
	// 这类错误是服务端内部问题,不应该暴露技术细节给用户

	// ErrInternalServer 通用的内部服务器错误
	// 用于包装所有未预期的系统错误
	// 前端应该显示友好的错误提示,不要暴露技术细节
	ErrInternalServer = 5000

	// ErrDatabaseError 数据库错误
	// 例如:连接失败、查询超时、死锁等
	// 这些错误应该被记录到日志,但不要返回详情给用户
	ErrDatabaseError = 5001

	// ErrCacheError 缓存错误
	// 例如:Redis 连接失败、缓存写入失败等
	// 一般缓存失败不应该影响主流程,可以降级到直接查数据库
	ErrCacheError = 5002
)
