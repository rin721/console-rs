package cache

// 本文件属于 Redis 缓存适配器，说明连接生命周期、键值操作、批处理或热重载边界。

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisCache Redis 缓存实现
// 封装 go-redis/v9 客户端,实现 Cache 接口
// 为什么封装 Redis客户端?
//   - 统一接口:业务代码只依赖 Cache 接口,不直接依赖 Redis
//   - 便于测试:可以轻松 mock Cache 接口
//   - 功能扩展:可以添加监控、熔断等功能
//   - 配置管理:支持配置热更新
type redisCache struct {
	// client Redis 客户端实例
	// 使用 go-redis/v9 库
	client *redis.Client

	// config 当前配置
	// 保存配置用于 Reload 时对比
	config *Config

	// mu 互斥锁
	// 保护 client 和 config 的并发访问
	// 为什么需要锁?
	//   - Reload 会替换 client
	//   - 其他方法可能同时在使用 client
	//   - 需要确保线程安全
	mu sync.RWMutex

	// logger 日志记录器(可选)
	// 用于记录连接、操作等日志
	logger Logger
}

// Logger 日志接口
// 定义 cache 包需要的日志方法
// 这样可以兼容不同的日志库
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// NewRedis 创建一个新的 Redis 缓存实例
// 参数:
//
//	config: Redis 配置
//	logger: 日志记录器,可以为 nil
//
// 返回:
//
//	Cache: Cache 接口实例
//	error: 创建失败时的错误
//
// 工作流程:
//  1. 验证配置
//  2. 创建 Redis 客户端
//  3. 测试连接
//  4. 返回 Cache 实例
//
// 使用示例:
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
func NewRedis(config *Config, logger Logger) (Cache, error) {
	if config == nil {
		config = DefaultConfig()
	}
	// 1. 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 2. 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		// 连接地址
		Addr: fmt.Sprintf("%s:%d", config.Host, config.Port),

		// 密码
		Username: config.Username,
		Password: config.Password,

		// 数据库索引
		DB: config.DB,

		// 连接池配置
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,

		// 重试配置
		MaxRetries: config.MaxRetries,

		// 超时配置
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	// 3. 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if logger != nil {
		logger.Info(MsgCacheConnecting,
			"host", config.Host,
			"port", config.Port,
			"db", config.DB)
	}

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		if logger != nil {
			logger.Error(MsgCacheConnectionFailed, "error", err)
		}
		return nil, fmt.Errorf(ErrMsgConnectionFailed, err)
	}

	if logger != nil {
		logger.Info(MsgCacheConnected)
	}

	// 4. 返回实例
	return &redisCache{
		client: client,
		config: config,
		logger: logger,
	}, nil
}

// Get 获取键的值
// 实现 Cache 接口
func (r *redisCache) Get(ctx context.Context, key string) (string, error) {
	// 使用读锁,允许并发读取
	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	// 执行 GET 命令
	result, err := client.Get(ctx, key).Result()
	if err != nil {
		// 检查是否是键不存在错误
		if errors.Is(err, redis.Nil) {
			// redis.Nil 表示键不存在,这是预期的情况,不是错误
			return "", fmt.Errorf(ErrMsgKeyNotFound, key)
		}
		// 其他错误
		return "", fmt.Errorf(ErrMsgOperationFailed, "get", err)
	}

	return result, nil
}

// Set 设置键值对
// 实现 Cache 接口
func (r *redisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	// 执行 SET 命令
	// expiration 为 0 表示永不过期
	err := client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return fmt.Errorf(ErrMsgOperationFailed, "set", err)
	}

	return nil
}

// Delete 删除键
// 实现 Cache 接口
func (r *redisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	// 执行 DEL 命令
	err := client.Del(ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf(ErrMsgOperationFailed, "delete", err)
	}

	return nil
}

// Exists 检查键是否存在
// 实现 Cache 接口
func (r *redisCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	// 执行 EXISTS 命令
	count, err := client.Exists(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf(ErrMsgOperationFailed, "exists", err)
	}

	return count, nil
}

// MGet 批量获取
// 实现 Cache 接口
func (r *redisCache) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	if len(keys) == 0 {
		return []interface{}{}, nil
	}

	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	// 执行 MGET 命令
	results, err := client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf(ErrMsgOperationFailed, "mget", err)
	}

	return results, nil
}

// MSet 批量设置
// 实现 Cache 接口
func (r *redisCache) MSet(ctx context.Context, pairs ...interface{}) error {
	if len(pairs) == 0 {
		return nil
	}

	// 验证参数数量必须是偶数
	if len(pairs)%2 != 0 {
		return fmt.Errorf("mset requires an even number of arguments")
	}

	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	// 执行 MSET 命令
	err := client.MSet(ctx, pairs...).Err()
	if err != nil {
		return fmt.Errorf(ErrMsgOperationFailed, "mset", err)
	}

	return nil
}

// Expire 设置过期时间
// 实现 Cache 接口
func (r *redisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	// 执行 EXPIRE 命令
	ok, err := client.Expire(ctx, key, expiration).Result()
	if err != nil {
		return fmt.Errorf(ErrMsgOperationFailed, "expire", err)
	}

	// ok 为 false 表示键不存在
	if !ok {
		return fmt.Errorf(ErrMsgKeyNotFound, key)
	}

	return nil
}

// TTL 获取剩余生存时间
// 实现 Cache 接口
func (r *redisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	// 执行 TTL 命令
	ttl, err := client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf(ErrMsgOperationFailed, "ttl", err)
	}

	return ttl, nil
}

// Incr 原子加 1
// 实现 Cache 接口
func (r *redisCache) Incr(ctx context.Context, key string) (int64, error) {
	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	// 执行 INCR 命令
	result, err := client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf(ErrMsgOperationFailed, "incr", err)
	}

	return result, nil
}

// Decr 原子减 1
// 实现 Cache 接口
func (r *redisCache) Decr(ctx context.Context, key string) (int64, error) {
	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	// 执行 DECR 命令
	result, err := client.Decr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf(ErrMsgOperationFailed, "decr", err)
	}

	return result, nil
}

// IncrBy 原子增加指定值
// 实现 Cache 接口
func (r *redisCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	// 执行 INCRBY 命令
	result, err := client.IncrBy(ctx, key, value).Result()
	if err != nil {
		return 0, fmt.Errorf(ErrMsgOperationFailed, "incrby", err)
	}

	return result, nil
}

// Ping 测试连接
// 实现 Cache 接口
func (r *redisCache) Ping(ctx context.Context) error {
	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	// 执行 PING 命令
	err := client.Ping(ctx).Err()
	if err != nil {
		if r.logger != nil {
			r.logger.Error(MsgCachePingFailed, "error", err)
		}
		return fmt.Errorf(ErrMsgPingFailed, err)
	}

	if r.logger != nil {
		r.logger.Info(MsgCachePingSuccess)
	}

	return nil
}

// Close 关闭连接
// 实现 Cache 接口
func (r *redisCache) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.client == nil {
		return nil
	}

	if r.logger != nil {
		r.logger.Info(MsgCacheClosing)
	}

	// 关闭 Redis 客户端
	err := r.client.Close()
	if err != nil {
		if r.logger != nil {
			r.logger.Error("failed to close redis", "error", err)
		}
		return err
	}

	r.client = nil

	if r.logger != nil {
		r.logger.Info(MsgCacheClosed)
	}

	return nil
}

// Reload 重新加载配置(原子操作)
// 实现 Cache 接口
// 这是配置热更新的关键方法
func (r *redisCache) Reload(ctx context.Context, newConfig *Config) error {
	// 1. 验证新配置
	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid new config: %w", err)
	}

	if r.logger != nil {
		r.logger.Info(MsgCacheReloading,
			"old_host", r.config.Host,
			"new_host", newConfig.Host,
			"old_port", r.config.Port,
			"new_port", newConfig.Port)
	}

	// 2. 创建新的 Redis 客户端
	newClient := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", newConfig.Host, newConfig.Port),
		Username:     newConfig.Username,
		Password:     newConfig.Password,
		DB:           newConfig.DB,
		PoolSize:     newConfig.PoolSize,
		MinIdleConns: newConfig.MinIdleConns,
		MaxRetries:   newConfig.MaxRetries,
		DialTimeout:  newConfig.DialTimeout,
		ReadTimeout:  newConfig.ReadTimeout,
		WriteTimeout: newConfig.WriteTimeout,
	})

	// 3. 测试新连接
	// 如果新配置无效,这里会失败,不会影响旧连接
	if err := newClient.Ping(ctx).Err(); err != nil {
		newClient.Close()
		if r.logger != nil {
			r.logger.Error(MsgCacheReloadFailed, "error", err)
		}
		return fmt.Errorf(ErrMsgReloadFailed, err)
	}

	// 4. 原子替换(使用写锁)
	// 这一步很快,不会阻塞太久
	r.mu.Lock()
	oldClient := r.client
	r.client = newClient
	r.config = newConfig
	r.mu.Unlock()

	// 5. 关闭旧连接
	// 在锁外执行,避免阻塞
	if oldClient != nil {
		oldClient.Close()
	}

	if r.logger != nil {
		r.logger.Info(MsgCacheReloaded)
	}

	return nil
}
