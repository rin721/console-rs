// Package httpserver 提供统一的 HTTP 服务器接口
// 基于标准库 net/http，支持配置热更新、运行期错误等待和优雅关闭
//
// # 设计目标
//
// - 统一接口: 抽象 HTTP 服务器实现
// - 易于使用: 简单直观的 API
// - 线程安全: 支持并发访问和配置热更新
// - 优雅关闭: 等待现有请求完成
// - 错误传播: 通过 Wait 接收异步 Serve 错误
// - 配置热更新: 支持运行时更新配置
//
// # 核心概念
//
// HTTPServer (HTTP 服务器):
//   - 封装标准库 http.Server
//   - 接受 http.Handler (通常是 Gin Router)
//   - 支持配置热更新、运行期等待、优雅启动和关闭
//
// Config (配置):
//   - Host: 监听地址
//   - Port: 监听端口
//   - ReadTimeout: 读取超时
//   - WriteTimeout: 写入超时
//   - IdleTimeout: 空闲连接超时
//
// # 使用示例
//
// 创建 HTTP Server 实例:
//
//	router := gin.Default()
//	config := &httpserver.Config{
//	    Host: "localhost",
//	    Port: 8080,
//	    ReadTimeout:  15 * time.Second,
//	    WriteTimeout: 15 * time.Second,
//	    IdleTimeout:  60 * time.Second,
//	}
//	server, err := httpserver.New(router, config, logger)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// 启动服务器:
//
//	// 非阻塞启动
//	if err := server.Start(context.Background()); err != nil {
//	    log.Error("start server", "error", err)
//	}
//
// 等待运行期错误:
//
//	go func() {
//	    if err := server.Wait(context.Background()); err != nil {
//	        log.Error("server runtime error", "error", err)
//	    }
//	}()
//
// 优雅关闭:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	if err := server.Shutdown(ctx); err != nil {
//	    log.Error("shutdown error", "error", err)
//	}
//
// 配置热更新:
//
//	newConfig := &httpserver.Config{
//	    Host: "0.0.0.0",
//	    Port: 8081,
//	    ReadTimeout:  20 * time.Second,
//	    WriteTimeout: 20 * time.Second,
//	    IdleTimeout:  90 * time.Second,
//	}
//	if err := server.Reload(context.Background(), newConfig); err != nil {
//	    log.Error("reload error", "error", err)
//	}
//
// # 使用场景
//
// 1. Web 应用服务器:
//   - RESTful API 服务
//   - Web 应用后端
//   - 微服务 HTTP 接口
//
// 2. 配置热更新:
//   - 运行时调整超时参数
//   - 动态切换监听地址和端口
//   - 无需重启服务即可应用配置
//
// 3. 优雅关闭:
//   - 容器编排平台 (K8s, Docker)
//   - 滚动更新场景
//   - 确保请求不丢失
//
// # 最佳实践
//
// 1. 超时配置:
//   - ReadTimeout: 防止慢速客户端占用连接
//   - WriteTimeout: 防止响应写入时间过长
//   - IdleTimeout: 及时回收空闲连接
//
// 2. 端口选择:
//   - 开发环境: 使用高位端口 (>1024)
//   - 生产环境: 使用标准端口 (80, 443)
//   - 使用 utils.GetAvailablePort 自动分配端口
//
// 3. 错误处理:
//   - 启动失败应该终止程序
//   - 运行期 Serve 错误应该通过 Wait 传回主进程
//   - 关闭失败记录日志但继续清理其他资源
//   - 热更新失败保持原配置运行
//
// 4. 并发安全:
//   - 所有方法都是线程安全的
//   - 可以在多个 goroutine 中调用
//   - 热更新时使用原子操作
//
// # 线程安全
//
// HTTPServer 的所有方法都是线程安全的，可以在多个 goroutine 中并发调用。
// 内部使用 sync.RWMutex 保护配置和服务器实例。
//
// # 与其他包的区别
//
// pkg/httpserver:
//   - 管理 **HTTP 服务器**
//   - 封装 net/http.Server
//   - 支持配置热更新
//
// pkg/cache:
//   - 管理 **缓存数据**
//   - 基于 Redis
//   - 提供键值存储
//
// pkg/database:
//   - 管理 **数据库连接**
//   - 基于 GORM
//   - 提供持久化存储
//
// pkg/executor:
//   - 管理 **协程池**
//   - 基于 ants
//   - 执行异步任务
package httpserver

// 本文件承载包级 Godoc 入口，集中说明该包在脚手架架构中的定位、使用边界和非目标能力。
