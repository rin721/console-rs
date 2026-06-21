package cache

// 本文件属于 Redis 缓存适配器，说明连接生命周期、键值操作、批处理或热重载边界。

import (
	"context"
	"time"
)

// Cache 定义缓存操作的接口
// 提供统一的缓存访问API,隔离具体实现(Redis/Memcached等)
//
// 为什么使用接口?
//   - 抽象:可以轻松切换不同的缓存后端
//   - 测试:可以创建 mock 实现进行单元测试
//   - 扩展:可以添加装饰器(如监控、熔断)而不修改业务代码
//
// 使用示例:
//
//	// 创建 Redis 缓存
//	cache, err := cache.NewRedis(config)
//
//	// 基本操作
//	err = cache.Set(ctx, "key", "value", 1*time.Hour)
//	value, err := cache.Get(ctx, "key")
//	err = cache.Delete(ctx, "key")
//
//	// 原子操作
//	count, err := cache.Incr(ctx, "counter")
type Cache interface {
	// Get 获取指定键的值
	// 参数:
	//   ctx: 上下文,用于超时控制和取消
	//   key: 缓存键名
	// 返回:
	//   string: 键对应的值
	//   error: 如果键不存在,返回 ErrKeyNotFound;其他错误返回具体错误信息
	// 使用示例:
	//   value, err := cache.Get(ctx, "user:123")
	//   if err == cache.ErrKeyNotFound {
	//       // 键不存在,从数据库加载
	//   }
	Get(ctx context.Context, key string) (string, error)

	// Set 设置键值对
	// 参数:
	//   ctx: 上下文
	//   key: 缓存键名
	//   value: 要缓存的值,会自动序列化
	//   expiration: 过期时间,0 表示永不过期
	// 返回:
	//   error: 设置失败时的错误
	// 注意:
	//   - 如果键已存在,会被覆盖
	//   - value 可以是 string、int、struct 等,会自动转换
	// 使用示例:
	//   err := cache.Set(ctx, "user:123", user, 1*time.Hour)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	// Delete 删除一个或多个键
	// 参数:
	//   ctx: 上下文
	//   keys: 要删除的键名列表
	// 返回:
	//   error: 删除失败时的错误
	// 注意:
	//   - 如果键不存在,不会返回错误
	//   - 可以一次删除多个键,提高效率
	// 使用示例:
	//   err := cache.Delete(ctx, "user:123", "user:456")
	Delete(ctx context.Context, keys ...string) error

	// Exists 检查键是否存在
	// 参数:
	//   ctx: 上下文
	//   keys: 要检查的键名列表
	// 返回:
	//   int64: 存在的键的数量
	//   error: 检查失败时的错误
	// 使用示例:
	//   count, err := cache.Exists(ctx, "user:123", "user:456")
	//   // count = 2 表示两个键都存在
	Exists(ctx context.Context, keys ...string) (int64, error)

	// MGet 批量获取多个键的值
	// 参数:
	//   ctx: 上下文
	//   keys: 要获取的键名列表
	// 返回:
	//   []interface{}: 值的列表,顺序与 keys 一致
	//     - 如果某个键不存在,对应位置为 nil
	//   error: 获取失败时的错误
	// 使用场景:
	//   - 一次性获取多个用户信息
	//   - 批量查询缓存,减少网络往返
	// 使用示例:
	//   values, err := cache.MGet(ctx, "user:1", "user:2", "user:3")
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)

	// MSet 批量设置多个键值对
	// 参数:
	//   ctx: 上下文
	//   pairs: 键值对,格式为 key1, value1, key2, value2, ...
	// 返回:
	//   error: 设置失败时的错误
	// 注意:
	//   - pairs 必须是偶数个参数
	//   - 批量设置不支持单独的过期时间
	// 使用示例:
	//   err := cache.MSet(ctx, "key1", "value1", "key2", "value2")
	MSet(ctx context.Context, pairs ...interface{}) error

	// Expire 设置键的过期时间
	// 参数:
	//   ctx: 上下文
	//   key: 键名
	//   expiration: 过期时间
	// 返回:
	//   error: 设置失败时的错误,如果键不存在也返回错误
	// 使用场景:
	//   - 延长缓存的有效期
	//   - 设置动态过期时间
	// 使用示例:
	//   err := cache.Expire(ctx, "session:abc", 30*time.Minute)
	Expire(ctx context.Context, key string, expiration time.Duration) error

	// TTL 获取键的剩余生存时间
	// 参数:
	//   ctx: 上下文
	//   key: 键名
	// 返回:
	//   time.Duration: 剩余时间
	//     - -1 表示键存在但没有设置过期时间
	//     - -2 表示键不存在
	//   error: 查询失败时的错误
	// 使用示例:
	//   ttl, err := cache.TTL(ctx, "session:abc")
	//   if ttl < 5*time.Minute {
	//       // 快过期了,续期或重新加载
	//   }
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Incr 将键的整数值加 1
	// 参数:
	//   ctx: 上下文
	//   key: 键名
	// 返回:
	//   int64: 增加后的值
	//   error: 操作失败时的错误
	// 注意:
	//   - 如果键不存在,会被初始化为 0 再加 1
	//   - 如果键的值不是整数,会返回错误
	//   - 这是原子操作,线程安全
	// 使用场景:
	//   - 页面访问计数
	//   - ID 生成器
	//   - 限流计数
	// 使用示例:
	//   count, err := cache.Incr(ctx, "page_views")
	Incr(ctx context.Context, key string) (int64, error)

	// Decr 将键的整数值减 1
	// 参数:
	//   ctx: 上下文
	//   key: 键名
	// 返回:
	//   int64: 减少后的值
	//   error: 操作失败时的错误
	// 使用场景:
	//   - 库存扣减
	//   - 剩余次数
	// 使用示例:
	//   remaining, err := cache.Decr(ctx, "stock:item123")
	Decr(ctx context.Context, key string) (int64, error)

	// IncrBy 将键的整数值增加指定数量
	// 参数:
	//   ctx: 上下文
	//   key: 键名
	//   value: 要增加的数量,可以是负数(相当于减少)
	// 返回:
	//   int64: 增加后的值
	//   error: 操作失败时的错误
	// 使用示例:
	//   count, err := cache.IncrBy(ctx, "points:user123", 10)
	IncrBy(ctx context.Context, key string, value int64) (int64, error)

	// Ping 测试与缓存服务器的连接
	// 参数:
	//   ctx: 上下文
	// 返回:
	//   error: 如果连接正常返回 nil,否则返回错误
	// 使用场景:
	//   - 健康检查
	//   - 连接验证
	// 使用示例:
	//   if err := cache.Ping(ctx); err != nil {
	//       log.Error("cache is down", "error", err)
	//   }
	Ping(ctx context.Context) error

	// Close 关闭缓存连接
	// 应该在应用关闭时调用,释放资源
	// 返回:
	//   error: 关闭失败时的错误
	// 注意:
	//   - Close 后不应该再使用此 Cache 实例
	//   - 应该在 defer 或 shutdown 中调用
	// 使用示例:
	//   defer cache.Close()
	Close() error

	// Reload 重新加载配置(原子操作)
	// 用于配置热更新；新客户端验证通过后才替换旧客户端，调用方仍需承受底层网络切换窗口。
	// 参数:
	//   ctx: 上下文
	//   config: 新的配置
	// 返回:
	//   error: 重载失败时的错误
	// 工作流程:
	//  1. 创建新的连接
	//  2. 测试新连接是否可用
	//  3. 原子替换旧连接
	//  4. 关闭旧连接
	// 注意:
	//   - 重载过程中服务不中断
	//   - 如果新配置无效,保持使用旧配置
	// 使用示例:
	//   newConfig := &cache.Config{...}
	//   if err := cache.Reload(ctx, newConfig); err != nil {
	//       log.Error("failed to reload cache", "error", err)
	//   }
	Reload(ctx context.Context, config *Config) error
}
