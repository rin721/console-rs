// Package executor 提供通用的并发任务调度器
// 基于 ants 协程池实现,支持多池隔离、原子热重载、panic 恢复
// 设计目标:
// - 提供多维资源隔离,防止单一任务类型耗尽所有资源
// - 全链路 panic 捕获,确保任何异步任务的异常都不会导致主进程崩溃
// - 支持动态热重载,无需重启进程即可更新配置
// - 接口化设计,便于依赖注入和单元测试
package executor

// 本文件属于异步执行器，定义多池隔离、任务提交、panic 恢复、过载反馈和热重载策略。

import "time"

// PoolName 定义池的名称类型
// 使用类型别名提供类型安全,防止字符串拼写错误
// 业务层应该定义常量:
//
//	const (
//	    PoolNameHTTP     executor.PoolName = "http"
//	    PoolNameDatabase executor.PoolName = "database"
//	)
type PoolName string

// Config 保存单个池的配置
// 定义了协程池的行为参数
type Config struct {
	// Name 池的唯一标识符
	// 必须在所有池中唯一
	// 例如: "http", "database", "background"
	Name PoolName `json:"name" yaml:"name" mapstructure:"name"`

	// Size 池的容量,即最大并发 worker 数量
	// 取值范围: [MinPoolSize, MaxPoolSize]
	// 设置建议:
	// - CPU 密集型任务: runtime.NumCPU() * 2
	// - IO 密集型任务: 可以设置更大,如 100-500
	// - 根据实际负载测试调整
	Size int `json:"size" yaml:"size" mapstructure:"size"`

	// Expiry worker 的过期时间
	// 闲置超过此时间的 worker 会被回收
	// 好处:
	// - 释放不再需要的资源
	// - 避免长期持有过多 goroutine
	// 推荐值: 10s - 60s
	Expiry time.Duration `json:"expiry" yaml:"expiry" mapstructure:"expiry"`

	// NonBlocking 是否使用非阻塞模式
	// true:  池满时立即返回 ErrPoolOverload
	// false: 池满时阻塞等待,直到有可用 worker
	// 推荐使用 true,让业务层决定降级策略:
	// - HTTP 服务: 返回 503
	// - CLI 工具: 阻塞等待
	// - 后台任务: 重试或丢弃
	NonBlocking bool `json:"nonBlocking" yaml:"nonBlocking" mapstructure:"nonBlocking"`
}

// Validate 验证配置有效性
// 确保所有字段都在合理范围内
// 返回:
//
//	error: 验证失败时的错误
func (c *Config) Validate() error {
	// 验证池名称
	if c.Name == "" {
		return ErrInvalidConfig
	}

	// 验证池大小
	if c.Size < MinPoolSize {
		c.Size = MinPoolSize
	}
	if c.Size > MaxPoolSize {
		c.Size = MaxPoolSize
	}

	// 验证过期时间
	// 如果未设置,使用默认值
	if c.Expiry <= 0 {
		c.Expiry = DefaultWorkerExpiry
	}

	return nil
}

// Manager 定义执行器管理器接口
// 这是组件的核心接口,提供任务执行和生命周期管理
// 为什么使用接口:
// - 便于单元测试(可以 mock)
// - 支持不同的实现
// - 符合依赖倒置原则
type Manager interface {
	// Execute 向指定名称的池提交任务
	// 参数:
	//   poolName: 池名称,必须是已配置的池
	//   task: 要执行的任务函数
	// 返回:
	//   error: 提交失败时的错误
	// 可能的错误:
	//   - ErrPoolNotFound: 池不存在
	//   - ErrPoolOverload: 池已满(仅当 NonBlocking=true)
	//   - ErrManagerClosed: 管理器已关闭
	// 使用示例:
	//   err := mgr.Execute("http", func() {
	//       // 处理 HTTP 请求
	//   })
	//   if err == executor.ErrPoolOverload {
	//       // 处理过载情况
	//   }
	Execute(poolName PoolName, task func()) error

	// Reload 使用新配置热重载所有池
	// 这是一个原子操作,失败时保持原配置不变
	// 参数:
	//   configs: 新的池配置列表
	// 返回:
	//   error: 重载失败时的错误
	// 重载流程:
	//   1. 验证所有新配置
	//   2. 创建新的池实例
	//   3. 原子替换旧池
	//   4. 优雅关闭旧池
	// 并发安全:
	//   - 使用读写锁保护
	//   - 失败时保持原有池不变
	//   - 新池创建成功后才替换旧池
	// 使用示例:
	//   newConfigs := []executor.Config{
	//       {Name: "http", Size: 200, NonBlocking: true},
	//   }
	//   if err := mgr.Reload(newConfigs); err != nil {
	//       log.Error("reload failed", "error", err)
	//   }
	Reload(configs []Config) error

	// Shutdown 优雅关闭管理器
	// 停止接收新任务,等待现有任务完成
	// 流程:
	//   1. 标记管理器为已关闭状态
	//   2. 等待所有池中的任务完成(最多 ShutdownTimeout)
	//   3. 强制释放资源
	// 注意:
	//   - 调用后管理器不可再使用
	//   - 应该在应用退出前调用
	//   - 会阻塞直到所有任务完成或超时
	// 使用示例:
	//   defer mgr.Shutdown()
	Shutdown()
}
