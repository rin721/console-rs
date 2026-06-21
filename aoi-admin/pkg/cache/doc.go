// Package cache 提供统一的缓存操作接口
// 基于 Redis v9,支持常用的缓存操作和配置热更新
//
// # 设计目标
//
// - 统一接口:抽象不同的缓存后端
// - 易于使用:简单直观的 API
// - 线程安全:支持并发访问
// - 配置热更新:支持运行时更新配置
// - 性能优化:连接池、批量操作等
//
// # 核心概念
//
// Cache(缓存):
//   - 键值存储,用于临时数据存储
//   - 提高应用性能,减轻数据库压力
//   - 支持过期时间,自动清理
//
// Redis:
//   - 高性能的内存数据库
//   - 支持丰富的数据结构
//   - 本包目前使用 String 类型
//
// # 使用示例
//
// 创建缓存实例:
//
//	config := &cache.Config{
//	    Host: "localhost",
//	    Port: 6379,
//	    DB:   0,
//	}
//	cache, err := cache.NewRedis(config, logger)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cache.Close()
//
// 基本操作:
//
//	// 设置缓存
//	err = cache.Set(ctx, "user:123", "John", 1*time.Hour)
//
//	// 获取缓存
//	value, err := cache.Get(ctx, "user:123")
//
//	// 删除缓存
//	err = cache.Delete(ctx, "user:123")
//
// 原子操作:
//
//	// 计数器
//	count, err := cache.Incr(ctx, "page_views")
//
//	// 增加指定值
//	points, err := cache.IncrBy(ctx, "user:points", 10)
//
// 批量操作:
//
//	// 批量获取
//	values, err := cache.MGet(ctx, "key1", "key2", "key3")
//
//	// 批量设置
//	err = cache.MSet(ctx, "key1", "value1", "key2", "value2")
//
// 配置热更新:
//
//	newConfig := &cache.Config{
//	    Host: "new-redis-host",
//	    Port: 6379,
//	}
//	err = cache.Reload(ctx, newConfig)
//
// # 使用场景
//
// 1. 数据缓存:
//   - 缓存数据库查询结果
//   - 减少数据库访问
//   - 提高响应速度
//
// 2. 会话存储:
//   - 存储用户会话信息
//   - 支持分布式部署
//   - 自动过期清理
//
// 3. 计数器:
//   - 页面访问统计
//   - API 调用限流
//   - 实时排行榜
//
// 4. 分布式锁:
//   - 防止并发冲突
//   - 保证操作原子性
//   - 使用 SET NX 实现
//
// # 最佳实践
//
// 1. 键命名:
//   - 使用命名空间:user:123
//   - 见名知意:session:abc
//   - 统一前缀:使用包提供的常量
//
// 2. 过期时间:
//   - 始终设置过期时间
//   - 防止内存泄漏
//   - 使用包提供的时间常量
//
// 3. 错误处理:
//   - 区分键不存在和其他错误
//   - 缓存失败不应该导致服务不可用
//   - 实现降级策略
//
// 4. 性能优化:
//   - 使用批量操作
//   - 合理设置连接池大小
//   - 避免存储过大的值
//
// # 线程安全
//
// 所有 Cache 方法都是线程安全的,可以在多个 goroutine 中并发调用
//
// # 与其他包的区别
//
// pkg/cache:
//   - 管理**缓存数据**
//   - 临时存储,提高性能
//   - 基于 Redis
//
// pkg/database:
//   - 管理**持久化数据**
//   - 永久存储,保证数据安全
//   - 基于 MySQL/PostgreSQL
//
// pkg/daemon:
//   - 管理**长期运行的服务**
//   - HTTP 服务器、gRPC 等
//
// pkg/scheduler:
//   - 管理**短期异步任务**
//   - 发送邮件、记录日志等
package cache

// 本文件承载包级 Godoc 入口，集中说明该包在脚手架架构中的定位、使用边界和非目标能力。
