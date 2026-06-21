package executor

// 本文件属于异步执行器，定义多池隔离、任务提交、panic 恢复、过载反馈和热重载策略。

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// 编译时检查 manager 是否实现了 Manager 接口
// 这是 Go 中的一个常见模式,确保类型安全
var _ Manager = (*manager)(nil)

// manager 实现 Manager 接口
// 管理多个协程池,提供任务执行和动态重载能力
// 线程安全设计:
// - 使用 RWMutex 保护 pools map
// - 使用 atomic.Bool 标记关闭状态
// - 遵循 Copy-On-Write 模式进行热重载
type manager struct {
	// mu 读写锁,保护 pools 的并发访问
	// 读锁: Execute 方法(允许并发)
	// 写锁: Reload、Shutdown 方法(独占访问)
	mu sync.RWMutex

	// pools 存储所有协程池
	// key: 池名称
	// value: 池包装器
	pools map[PoolName]*poolWrapper

	// closed 标记管理器是否已关闭
	// 使用 atomic 实现无锁检查
	closed atomic.Bool
}

// NewManager 创建一个新的执行器管理器
// 这是主要的工厂函数
// 参数:
//
//	configs: 池配置列表
//
// 返回:
//
//	Manager: 管理器接口
//	error: 创建失败时的错误
//
// 使用示例:
//
//	configs := []executor.Config{
//	    {Name: "http", Size: 200, NonBlocking: true},
//	    {Name: "background", Size: 50, NonBlocking: false},
//	}
//	mgr, err := executor.NewManager(configs)
//	if err != nil {
//	    log.Fatal("failed to create executor", "error", err)
//	}
//	defer mgr.Shutdown()
func NewManager(configs []Config) (Manager, error) {
	// 验证配置
	if len(configs) == 0 {
		return nil, fmt.Errorf("%w: no configs provided", ErrInvalidConfig)
	}

	// 创建池 map
	pools := make(map[PoolName]*poolWrapper, len(configs))

	// 创建所有池
	for _, cfg := range configs {
		// 检查重复名称
		if _, exists := pools[cfg.Name]; exists {
			// 清理已创建的池
			releasePools(pools)
			return nil, fmt.Errorf("%w: duplicate pool name: %s", ErrInvalidConfig, cfg.Name)
		}

		// 创建池
		pool, err := newPoolWrapper(cfg)
		if err != nil {
			// 创建失败,清理已创建的池
			releasePools(pools)
			return nil, fmt.Errorf("failed to create pool %s: %w", cfg.Name, err)
		}

		pools[cfg.Name] = pool
	}

	return &manager{
		pools: pools,
	}, nil
}

// Execute 向指定池提交任务
// 实现 Manager 接口
// 参数:
//
//	poolName: 池名称
//	task: 要执行的任务函数
//
// 返回:
//
//	error: 提交失败时的错误
//
// 线程安全:
//
//	使用读锁保护,允许并发调用
func (m *manager) Execute(poolName PoolName, task func()) error {
	// 快速检查管理器是否已关闭
	// 使用 atomic 无锁检查,性能更好
	if m.closed.Load() {
		return ErrManagerClosed
	}

	// 获取读锁
	// 允许多个 goroutine 同时执行
	m.mu.RLock()
	pool, exists := m.pools[poolName]
	m.mu.RUnlock()

	// 检查池是否存在
	if !exists {
		return fmt.Errorf(ErrMsgPoolNotFound, ErrPoolNotFound, poolName)
	}

	// 提交任务到池
	if err := pool.Submit(task); err != nil {
		// 如果是池过载错误,添加池名称信息
		if err == ErrPoolOverload {
			return fmt.Errorf(ErrMsgPoolOverload, ErrPoolOverload, poolName)
		}
		return err
	}

	return nil
}

// Reload 使用新配置重新加载所有池
// 实现 Manager 接口
// 这是一个原子操作,遵循以下步骤:
//  1. 在锁外创建所有新池(避免长时间持有锁)
//  2. 如果任何池创建失败,清理并返回错误
//  3. 获取写锁
//  4. 原子替换池 map
//  5. 释放写锁
//  6. 在锁外优雅关闭旧池
//
// 参数:
//
//	configs: 新的池配置列表
//
// 返回:
//
//	error: 重载失败时的错误
//
// 并发安全:
//
//	使用写锁保护,确保原子性
//	失败时保持原有池不变
func (m *manager) Reload(configs []Config) error {
	// 检查管理器是否已关闭
	if m.closed.Load() {
		return ErrManagerClosed
	}

	// 1. 在锁外创建新池
	// 避免长时间持有锁,提高并发性能
	newPools := make(map[PoolName]*poolWrapper, len(configs))

	for _, cfg := range configs {
		// 检查重复名称
		if _, exists := newPools[cfg.Name]; exists {
			releasePools(newPools)
			return fmt.Errorf("%w: duplicate pool name: %s", ErrInvalidConfig, cfg.Name)
		}

		// 创建新池
		pool, err := newPoolWrapper(cfg)
		if err != nil {
			// 创建失败,清理所有新池
			releasePools(newPools)
			return fmt.Errorf(ErrMsgReloadFailed, err)
		}

		newPools[cfg.Name] = pool
	}

	// 2. 获取写锁,开始原子替换
	m.mu.Lock()

	// 保存旧池的引用,用于后续清理
	oldPools := m.pools

	// 3. 原子地替换池 map
	// 从这一刻起,Execute 会使用新池
	m.pools = newPools

	// 4. 释放写锁
	// 新池已就绪,其他 goroutine 可以使用
	m.mu.Unlock()

	// 5. 在锁外优雅关闭旧池
	// 避免长时间持有锁
	// 旧池可能还有正在执行的任务
	// ReleaseTimeout 会等待任务完成或超时
	releasePools(oldPools)

	return nil
}

// Shutdown 优雅关闭管理器
// 实现 Manager 接口
// 步骤:
//  1. 标记管理器为已关闭
//  2. 等待所有池中的任务完成
//  3. 释放所有资源
//
// 注意:
//
//	调用后管理器不可再使用
//	会阻塞直到所有任务完成或超时
func (m *manager) Shutdown() {
	// 标记为已关闭
	// 使用 atomic 确保线程安全
	m.closed.Store(true)

	// 获取写锁
	m.mu.Lock()
	pools := m.pools
	m.pools = make(map[PoolName]*poolWrapper) // 清空池 map
	m.mu.Unlock()

	// 在锁外释放池
	releasePools(pools)
}

// releasePools 释放池 map 中的所有池
// 辅助函数,用于清理资源
// 参数:
//
//	pools: 要释放的池 map
func releasePools(pools map[PoolName]*poolWrapper) {
	// 使用 WaitGroup 并发释放所有池
	// 提高关闭速度
	var wg sync.WaitGroup
	wg.Add(len(pools))

	for _, pool := range pools {
		pool := pool // 捕获循环变量
		go func() {
			defer wg.Done()
			// 带超时的释放
			// 如果超时,会返回错误但继续执行
			_ = pool.ReleaseTimeout(ShutdownTimeout)
		}()
	}

	// 等待所有池释放完成
	wg.Wait()
}
