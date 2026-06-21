/*
Package executor 提供基于 ants 的通用并发任务调度器

# 概述

executor 包为 rei0721 项目提供了统一的异步任务执行基础设施。
它基于 panjf2000/ants 协程池库,并添加了以下增强功能:

  - 多维资源隔离: 支持多个独立的协程池,按业务特征隔离
  - 全链路安全防护: 内置 panic 捕获,确保任何任务异常都不会导致进程崩溃
  - 动态热重载: 支持运行时原子更新池配置,无需重启进程
  - 接口化设计: 便于依赖注入和单元测试

# 核心概念

## 池隔离 (Pool Isolation)

不同类型的任务使用不同的池,防止资源争抢:

	configs := []executor.Config{
	    {Name: "http", Size: 200, NonBlocking: true},      // HTTP 请求处理
	    {Name: "database", Size: 50, NonBlocking: false},  // 数据库操作
	    {Name: "background", Size: 30, NonBlocking: true}, // 后台任务
	}

## 非阻塞模式 (Non-Blocking Mode)

推荐使用非阻塞模式,让业务层决定降级策略:

  - HTTP 服务: 池满时返回 503,避免雪崩
  - CLI 工具: 可以选择阻塞等待
  - 后台任务: 可以重试或丢弃

## 原子热重载 (Atomic Hot-Reload)

配置变更时无缝切换,不影响进行中的任务:

 1. 创建新池
 2. 原子替换
 3. 旧池优雅退出

# 使用示例

## 基本用法

	package main

	import (
	    "log"
	    "github.com/rei0721/go-scaffold/pkg/executor"
	)

	func main() {
	    // 1. 创建配置
	    configs := []executor.Config{
	        {
	            Name:        "http",
	            Size:        200,
	            Expiry:      10 * time.Second,
	            NonBlocking: true,
	        },
	    }

	    // 2. 创建管理器
	    mgr, err := executor.NewManager(configs)
	    if err != nil {
	        log.Fatal(err)
	    }
	    defer mgr.Shutdown()

	    // 3. 提交任务
	    err = mgr.Execute("http", func() {
	        // 处理业务逻辑
	        processRequest()
	    })

	    if err == executor.ErrPoolOverload {
	        // 处理过载情况
	        log.Println("pool overloaded, rejecting request")
	    }
	}

## 热重载

	// 配置变更时
	func onConfigChange(old, new *Config) {
	    newConfigs := []executor.Config{
	        {Name: "http", Size: new.HTTPPoolSize, NonBlocking: true},
	    }

	    if err := mgr.Reload(newConfigs); err != nil {
	        log.Printf("reload failed: %v", err)
	        // 失败时继续使用旧配置
	    }
	}

## 错误处理

	err := mgr.Execute("mypool", task)
	switch err {
	case nil:
	    // 成功
	case executor.ErrPoolNotFound:
	    // 池不存在,检查配置
	case executor.ErrPoolOverload:
	    // 池已满,降级处理
	case executor.ErrManagerClosed:
	    // 管理器已关闭,停止提交任务
	default:
	    // 其他错误
	}

## 与依赖注入集成

	// internal/app/app.go
	type App struct {
	    Executor executor.Manager
	    // ...
	}

	func (a *App) Init() error {
	    configs := buildExecutorConfigs(a.Config)
	    mgr, err := executor.NewManager(configs)
	    if err != nil {
	        return err
	    }
	    a.Executor = mgr
	    return nil
	}

	// internal/modules/iam/service/user.go
	type UserService struct {
	    executor executor.Manager
	}

	func (s *UserService) SendEmail(userID int) error {
	    return s.executor.Execute("email", func() {
	        // 异步发送邮件
	        sendEmailToUser(userID)
	    })
	}

# 配置建议

## 池大小 (Pool Size)

  - CPU 密集型: runtime.NumCPU() * 2
  - IO 密集型: 100-500(根据实际测试调整)
  - 混合型: 根据 CPU/IO 比例调整

## 过期时间 (Expiry)

  - 高频任务: 5-10 秒
  - 低频任务: 30-60 秒
  - 平衡资源利用和响应速度

## 非阻塞模式 (NonBlocking)

  - HTTP/RPC 服务: 推荐 true
  - CLI 工具: 可选 false
  - 后台任务: 推荐 true

# 最佳实践

// ## 1. 定义池名称常量
//
// 推荐在应用层常量包中统一定义池名称字符串常量:
//
//	// app/constants/executor.go
//	package constants
//
//	const (
//	    PoolHTTP       = "http"
//	    PoolDatabase   = "database"
//	    PoolCache      = "cache"
//	    PoolLogger     = "logger"
//	    PoolBackground = "background"
//	)
//
// 在业务代码中使用常量:
//
//	import "your/app/constants"
//
//	// 在 Service 层
//	func (s *UserService) SendWelcomeEmail(userID int64) error {
//	    return s.executor.Execute(constants.PoolBackground, func() {
//	        // 异步发送欢迎邮件
//	    })
//	}
//
//	// 在 HTTP Handler 中
//	func (h *Handler) LogRequest() error {
//	    return h.executor.Execute(constants.PoolHTTP, func() {
//	        // 异步记录请求日志
//	    })
//	}
//
//	// 在 Repository 层
//	func (r *UserRepo) UpdateStatsAsync() error {
//	    return r.executor.Execute(constants.PoolDatabase, func() {
//	        // 异步更新统计数据
//	    })
//	}

## 2. 监控池状态

定期检查池的运行状态:

	pool.Running()  // 当前运行的 worker 数
	pool.Free()     // 当前空闲的 worker 数
	pool.Cap()      // 池容量

## 3. 优雅关闭

确保在应用退出时调用 Shutdown:

	defer mgr.Shutdown()

## 4. 错误处理

始终检查 Execute 返回的错误,并做相应处理。

# 并发安全

所有公开方法都是并发安全的:

  - Execute: 可以被多个 goroutine 同时调用
  - Reload: 使用写锁保护,确保原子性
  - Shutdown: 可以安全地与 Execute 并发调用

# 性能考虑

  - Execute 使用读锁,并发性能高
  - 任务包装开销极小(仅一个 defer)
  - Reload 在锁外创建新池,最小化锁持有时间
  - 使用 atomic.Bool 进行快速状态检查

# 注意事项

1. 任务函数不应 panic,虽然有恢复机制,但会影响性能
2. Shutdown 会阻塞直到所有任务完成或超时
3. 池名称必须唯一,重复会导致创建失败
4. 配置验证会自动修正不合理的值(如负数、过大值)
*/
package executor

// 本文件承载包级 Godoc 入口，集中说明该包在脚手架架构中的定位、使用边界和非目标能力。
