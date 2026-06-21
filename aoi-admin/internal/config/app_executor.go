package config

// 本文件定义一个配置分区及其校验规则，是外部配置进入运行时基础设施前的类型化边界。

import (
	"errors"
	"fmt"
)

// ExecutorConfig 执行器配置
// 管理多个异步任务池
type ExecutorConfig struct {
	// Enabled 是否启用执行器
	// false 时,应用不会创建执行器
	Enabled bool `mapstructure:"enabled" envname:"EXECUTOR_ENABLED"`

	// Pools 池配置列表
	// 每个池可以有不同的参数
	Pools []ExecutorPoolConfig `mapstructure:"pools"`
}

// ExecutorPoolConfig 单个执行器池的配置
// 定义了一个协程池的行为参数
type ExecutorPoolConfig struct {
	// Name 池的唯一标识符
	// 例如: "http", "database", "background"
	// 业务层应该定义常量来引用这些名称
	Name string `mapstructure:"name"`

	// Size 池的容量,即最大并发 worker 数量
	// 推荐值:
	// - CPU 密集型: runtime.NumCPU() * 2
	// - IO 密集型: 100-500
	Size int `mapstructure:"size"`

	// Expiry worker 的过期时间(秒)
	// 闲置超过此时间的 worker 会被回收
	// 推荐: 10-60 秒
	Expiry int `mapstructure:"expiry"`

	// NonBlocking 是否使用非阻塞模式
	// true:  池满时立即返回错误
	// false: 池满时阻塞等待
	// 推荐使用 true
	NonBlocking bool `mapstructure:"non_blocking"`
}

// ValidateName 返回当前配置分区在聚合校验错误中的稳定名称。
func (c *ExecutorConfig) ValidateName() string {
	return AppExecutorName
}

// ValidateRequired 声明当前配置分区是否必须出现在完整应用配置中。
func (c *ExecutorConfig) ValidateRequired() bool {
	return true
}

// Validate 验证执行器配置
// 实现 Configurable 接口
func (c *ExecutorConfig) Validate() error {
	// 如果未启用,跳过验证
	if !c.Enabled {
		return nil
	}

	// 启用时必须至少有一个池
	if len(c.Pools) == 0 {
		return errors.New("at least one pool is required when executor is enabled")
	}

	// 验证每个池的配置
	poolNames := make(map[string]bool)
	for i, pool := range c.Pools {
		// 验证池名称
		if pool.Name == "" {
			return fmt.Errorf("pool %d: name is required", i)
		}

		// 检查重复名称
		if poolNames[pool.Name] {
			return fmt.Errorf("duplicate pool name: %s", pool.Name)
		}
		poolNames[pool.Name] = true

		// 验证池大小
		if pool.Size <= 0 {
			return fmt.Errorf("pool %s: size must be positive", pool.Name)
		}
		if pool.Size > 10000 {
			return fmt.Errorf("pool %s: size must not exceed 10000", pool.Name)
		}

		// 验证过期时间
		if pool.Expiry < 0 {
			return fmt.Errorf("pool %s: expiry must be non-negative", pool.Name)
		}
	}

	return nil
}
