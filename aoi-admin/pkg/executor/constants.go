package executor

// 本文件属于异步执行器，定义多池隔离、任务提交、panic 恢复、过载反馈和热重载策略。

import (
	"errors"
	"time"
)

// 错误常量定义
// 遵循项目约定,使用常量定义错误消息模板
// 参考: pkg/logger/constants.go
const (
	// ErrMsgPoolNotFound 池不存在的错误消息模板
	ErrMsgPoolNotFound = "%w: %s"

	// ErrMsgPoolOverload 池过载的错误消息模板
	ErrMsgPoolOverload = "%w: %s"

	// ErrMsgInvalidConfig 无效配置的错误消息模板
	ErrMsgInvalidConfig = "invalid config: %w"

	// ErrMsgManagerClosed 管理器已关闭的错误消息
	ErrMsgManagerClosed = "manager is closed"

	// ErrMsgReloadFailed 重载失败的错误消息模板
	ErrMsgReloadFailed = "failed to reload: %w"

	// ErrMsgShutdownTimeout 关闭超时的错误消息
	ErrMsgShutdownTimeout = "shutdown timeout exceeded"
)

// 预定义错误
// 用于快速错误判断和返回
var (
	// ErrPoolNotFound 池不存在错误
	// 当尝试向不存在的池提交任务时返回
	ErrPoolNotFound = errors.New("pool not found")

	// ErrPoolOverload 池过载错误
	// 当池达到容量且配置为 NonBlocking 时返回
	// 调用方应该决定重试、同步执行还是丢弃任务
	ErrPoolOverload = errors.New("pool overloaded")

	// ErrManagerClosed 管理器已关闭错误
	// 当尝试在已关闭的管理器上执行操作时返回
	ErrManagerClosed = errors.New("manager is closed")

	// ErrInvalidConfig 无效配置错误
	// 配置验证失败时返回
	ErrInvalidConfig = errors.New("invalid config")
)

// 默认配置常量
// 提供合理的默认值,适用于大多数场景
const (
	// DefaultPoolSize 默认池大小
	// 推荐值:100 个 worker
	// 可根据应用并发量调整
	DefaultPoolSize = 100

	// DefaultWorkerExpiry 默认 worker 过期时间
	// 闲置 worker 会在此时间后被回收
	// 10 秒是一个平衡值:
	// - 不会太快导致频繁创建/销毁
	// - 不会太慢导致资源浪费
	DefaultWorkerExpiry = 10 * time.Second

	// DefaultNonBlocking 默认非阻塞模式
	// true: 池满时立即返回 ErrPoolOverload
	// false: 池满时阻塞等待
	// 推荐使用 true,让调用方决定降级策略
	DefaultNonBlocking = true

	// ShutdownTimeout 关闭超时时间
	// 等待池中任务完成的最大时间
	// 超过此时间会强制关闭
	// 5 秒适用于大多数场景
	ShutdownTimeout = 5 * time.Second

	// MinPoolSize 最小池大小
	// 确保池至少有一个 worker
	MinPoolSize = 1

	// MaxPoolSize 最大池大小
	// 防止配置过大导致系统资源耗尽
	// 10000 是一个安全的上限
	MaxPoolSize = 10000
)
