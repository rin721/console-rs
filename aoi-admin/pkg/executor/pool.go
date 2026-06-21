package executor

// 本文件属于异步执行器，定义多池隔离、任务提交、panic 恢复、过载反馈和热重载策略。

import (
	"fmt"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
)

// poolWrapper 封装 ants.Pool,提供额外功能
// 设计考虑:
// - 包装 ants 池,提供 panic 恢复
// - 统一错误处理
// - 跟踪池元数据
type poolWrapper struct {
	// name 池名称,用于日志和错误消息
	name PoolName

	// pool 底层的 ants 协程池
	// ants 提供高性能的 goroutine 池实现
	pool *ants.Pool

	// config 池配置,用于重建
	config Config
}

// newPoolWrapper 创建新的池包装器
// 参数:
//
//	cfg: 池配置
//
// 返回:
//
//	*poolWrapper: 池包装器实例
//	error: 创建失败时的错误
func newPoolWrapper(cfg Config) (*poolWrapper, error) {
	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf(ErrMsgInvalidConfig, err)
	}

	// 配置 ants 池选项
	options := []ants.Option{
		// 设置 worker 过期时间
		ants.WithExpiryDuration(cfg.Expiry),
		// 设置非阻塞模式
		ants.WithNonblocking(cfg.NonBlocking),
		// 禁用预分配,按需创建 worker
		ants.WithPreAlloc(false),
	}

	// 创建 ants 池
	pool, err := ants.NewPool(cfg.Size, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create ants pool: %w", err)
	}

	return &poolWrapper{
		name:   cfg.Name,
		pool:   pool,
		config: cfg,
	}, nil
}

// Submit 提交任务到池
// 自动包装任务以提供 panic 恢复
// 参数:
//
//	task: 要执行的任务函数
//
// 返回:
//
//	error: 提交失败时的错误
func (p *poolWrapper) Submit(task func()) error {
	// 包装任务,添加 panic 恢复
	wrapped := wrapTaskWithRecover(p.name, task)

	// 提交到 ants 池
	if err := p.pool.Submit(wrapped); err != nil {
		// 转换 ants 错误为项目错误
		if err == ants.ErrPoolOverload {
			return ErrPoolOverload
		}
		if err == ants.ErrPoolClosed {
			return ErrManagerClosed
		}
		return err
	}

	return nil
}

// Release 释放池资源
// 优雅关闭,等待所有任务完成
func (p *poolWrapper) Release() {
	if p.pool != nil {
		p.pool.Release()
	}
}

// ReleaseTimeout 带超时的释放池资源
// 参数:
//
//	timeout: 等待超时时间
//
// 返回:
//
//	error: 超时时返回错误
func (p *poolWrapper) ReleaseTimeout(timeout time.Duration) error {
	if p.pool == nil {
		return nil
	}

	// 创建一个通道用于通知释放完成
	done := make(chan struct{})

	go func() {
		p.pool.Release()
		close(done)
	}()

	// 等待释放完成或超时
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		// 超时,强制释放
		// 注意: ants 的 Release 会阻塞直到所有任务完成
		// 这里我们给它一个超时时间
		return fmt.Errorf(ErrMsgShutdownTimeout)
	}
}

// Running 返回当前运行的 worker 数量
func (p *poolWrapper) Running() int {
	if p.pool == nil {
		return 0
	}
	return p.pool.Running()
}

// Free 返回当前空闲的 worker 数量
func (p *poolWrapper) Free() int {
	if p.pool == nil {
		return 0
	}
	return p.pool.Free()
}

// Cap 返回池容量
func (p *poolWrapper) Cap() int {
	if p.pool == nil {
		return 0
	}
	return p.pool.Cap()
}

// wrapTaskWithRecover 包装任务,添加 panic 恢复
// 这是一个关键的安全机制,确保任何 panic 都不会导致进程崩溃
// 参数:
//
//	poolName: 池名称,用于日志
//	task: 原始任务函数
//
// 返回:
//
//	func(): 包装后的任务函数
func wrapTaskWithRecover(poolName PoolName, task func()) func() {
	return func() {
		// 使用 defer + recover 捕获 panic
		defer func() {
			if r := recover(); r != nil {
				if handler := getPanicHandler(); handler != nil {
					handler.HandlePanic(poolName, r)
					return
				}
				// 捕获到 panic,记录详细信息
				// 注意: 这里我们不能直接使用 logger,因为:
				// 1. pkg 层不应依赖 internal 层
				// 2. logger 可能还未初始化
				// 3. 避免循环依赖
				// 最佳实践是让业务层注入 logger 或使用标准库
				fmt.Printf("[EXECUTOR PANIC] pool=%s panic=%v\n", poolName, r)
				// 在真实场景中,可以考虑:
				// - 通过回调函数记录到日志
				// - 发送到监控系统
				// - 增加 panic 计数器
			}
		}()

		// 执行实际任务
		task()
	}
}

// panicHandler 是一个可选的 panic 处理器接口
// 业务层可以通过此接口自定义 panic 处理逻辑
type panicHandler interface {
	HandlePanic(poolName PoolName, recovered interface{})
}

// 全局 panic 处理器
// 可以通过 SetPanicHandler 设置
var globalPanicHandler panicHandler
var panicHandlerMu sync.RWMutex

// SetPanicHandler 设置全局 panic 处理器
// 用于自定义 panic 处理逻辑
// 参数:
//
//	handler: panic 处理器
func SetPanicHandler(handler panicHandler) {
	panicHandlerMu.Lock()
	defer panicHandlerMu.Unlock()
	globalPanicHandler = handler
}

// getPanicHandler 获取当前的 panic 处理器
func getPanicHandler() panicHandler {
	panicHandlerMu.RLock()
	defer panicHandlerMu.RUnlock()
	return globalPanicHandler
}
