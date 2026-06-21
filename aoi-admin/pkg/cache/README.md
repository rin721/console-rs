# pkg/cache - Redis 缓存封装

## 概述

Cache 是一个基于 Redis v9 的缓存库封装，提供统一的缓存操作接口，支持配置热更新（原子化 Reload）。

## API 分类

- 定位：[CONFIRMED] 公共基础设施 API。
- 稳定边界：`Cache` 接口、`Config`、`DefaultConfig`、`NewRedis`。
- 当前风险：[CONFIRMED] Redis 依赖路径已有隔离行为测试；使用方仍应通过接口注入而不是依赖具体实现。
- 非目标：[CONFIRMED] 本包不承载业务缓存 key 约定。

### 特性

- ✅ **统一接口** - 抽象 Cache 接口，隔离具体实现
- ✅ **Redis v9** - 使用最新的 go-redis/v9 客户端
- ✅ **原子化重载** - 支持配置热更新，不中断服务
- ✅ **线程安全** - 所有操作都是并发安全的
- ✅ **连接池** - 高效的连接池管理
- ✅ **批量操作** - 支持 MGet/MSet 提高性能
- ✅ **原子操作** - Incr/Decr/IncrBy 计数器操作
- ✅ **详细注释** - 完整的中文注释，适合初学者

## 安装

```bash
go get github.com/redis/go-redis/v9
```

## 快速开始

### 1. 创建缓存实例

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/rei0721/go-scaffold/pkg/cache"
)

func main() {
    // 创建配置
    config := &cache.Config{
        Host:     "localhost",
        Port:     6379,
        Password: "",
        DB:       0,
        PoolSize: 10,
    }

    // 创建缓存实例(logger 可以为 nil)
    cache, err := cache.NewRedis(config, logger)
    if err != nil {
        log.Fatal(err)
    }
    defer cache.Close()
}
```

### 2. 基本操作

```go
ctx := context.Background()

// 设置缓存(1小时过期)
err := cache.Set(ctx, "user:123", "John Doe", 1*time.Hour)

// 获取缓存
value, err := cache.Get(ctx, "user:123")
if err != nil {
    log.Error("get failed", "error", err)
}

// 删除缓存
err = cache.Delete(ctx, "user:123")

// 检查是否存在
count, err := cache.Exists(ctx, "user:123", "user:456")
```

### 3. 批量操作

```go
// 批量获取
values, err := cache.MGet(ctx, "key1", "key2", "key3")
for i, value := range values {
    if value != nil {
        fmt.Printf("key%d = %v\n", i+1, value)
    }
}

// 批量设置
err = cache.MSet(ctx,
    "key1", "value1",
    "key2", "value2",
    "key3", "value3",
)
```

### 4. 原子操作

```go
// 页面访问计数
views, err := cache.Incr(ctx, "page:home:views")

// 库存扣减
remaining, err := cache.Decr(ctx, "product:123:stock")

// 积分增加
points, err := cache.IncrBy(ctx, "user:123:points", 10)
```

### 5. 过期时间管理

```go
// 设置过期时间
err := cache.Expire(ctx, "session:abc", 30*time.Minute)

// 查询剩余时间
ttl, err := cache.TTL(ctx, "session:abc")
if ttl < 5*time.Minute {
    log.Info("session will expire soon")
}
```

### 6. 配置热更新

```go
// 创建新配置
newConfig := &cache.Config{
    Host:     "new-redis-host.example.com",
    Port:     6379,
    Password: "new-password",
    PoolSize: 20,
}

// 原子化重载配置
// 如果新配置无效，会保持使用旧配置
err := cache.Reload(ctx, newConfig)
if err != nil {
    log.Error("failed to reload config", "error", err)
}
```

## API 文档

### Config 配置

```go
type Config struct {
    Host         string        // Redis 主机地址
    Port         int           // Redis 端口 (默认 6379)
    Password     string        // 密码
    DB           int           // 数据库索引 (0-15)
    PoolSize     int           // 连接池大小 (默认 10)
    MinIdleConns int           // 最小空闲连接 (默认 5)
    MaxRetries   int           // 最大重试次数 (默认 3)
    DialTimeout  time.Duration // 连接超时 (默认 5秒)
    ReadTimeout  time.Duration // 读取超时 (默认 3秒)
    WriteTimeout time.Duration // 写入超时 (默认 3秒)
}
```

**默认配置**：

```go
config := cache.DefaultConfig()
```

**配置验证**：

```go
if err := config.Validate(); err != nil {
    log.Fatal("invalid config:", err)
}
```

### Cache 接口

#### 基本操作

| 方法                        | 说明     | 示例                                                 |
| --------------------------- | -------- | ---------------------------------------------------- |
| `Get(ctx, key)`             | 获取值   | `value, err := cache.Get(ctx, "key")`                |
| `Set(ctx, key, value, exp)` | 设置值   | `err := cache.Set(ctx, "key", "value", 1*time.Hour)` |
| `Delete(ctx, keys...)`      | 删除键   | `err := cache.Delete(ctx, "key1", "key2")`           |
| `Exists(ctx, keys...)`      | 检查存在 | `count, err := cache.Exists(ctx, "key1", "key2")`    |

#### 批量操作

| 方法                  | 说明     | 示例                                             |
| --------------------- | -------- | ------------------------------------------------ |
| `MGet(ctx, keys...)`  | 批量获取 | `values, err := cache.MGet(ctx, "k1", "k2")`     |
| `MSet(ctx, pairs...)` | 批量设置 | `err := cache.MSet(ctx, "k1", "v1", "k2", "v2")` |

#### 过期时间

| 方法                    | 说明         | 示例                                              |
| ----------------------- | ------------ | ------------------------------------------------- |
| `Expire(ctx, key, exp)` | 设置过期     | `err := cache.Expire(ctx, "key", 10*time.Minute)` |
| `TTL(ctx, key)`         | 查询剩余时间 | `ttl, err := cache.TTL(ctx, "key")`               |

#### 原子操作

| 方法                      | 说明 | 示例                                           |
| ------------------------- | ---- | ---------------------------------------------- |
| `Incr(ctx, key)`          | 加 1 | `count, err := cache.Incr(ctx, "counter")`     |
| `Decr(ctx, key)`          | 减 1 | `remaining, err := cache.Decr(ctx, "stock")`   |
| `IncrBy(ctx, key, value)` | 加 N | `count, err := cache.IncrBy(ctx, "score", 10)` |

#### 连接管理

| 方法                  | 说明     | 示例                                  |
| --------------------- | -------- | ------------------------------------- |
| `Ping(ctx)`           | 测试连接 | `err := cache.Ping(ctx)`              |
| `Close()`             | 关闭连接 | `err := cache.Close()`                |
| `Reload(ctx, config)` | 重载配置 | `err := cache.Reload(ctx, newConfig)` |

## 使用场景

### 场景 1: 缓存数据库查询结果

```go
func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
    // 1. 尝试从缓存获取
    key := fmt.Sprintf("user:%d", id)
    data, err := s.cache.Get(ctx, key)
    if err == nil {
        var user User
        json.Unmarshal([]byte(data), &user)
        return &user, nil
    }

    // 2. 缓存未命中，从数据库加载
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // 3. 写入缓存
    data, _ := json.Marshal(user)
    s.cache.Set(ctx, key, string(data), 1*time.Hour)

    return user, nil
}
```

### 场景 2: 会话管理

```go
// 存储会话
func StoreSession(ctx context.Context, cache cache.Cache, sessionID string, data interface{}) error {
    key := cache.KeyPrefixSession + sessionID
    jsonData, _ := json.Marshal(data)
    return cache.Set(ctx, key, string(jsonData), 30*time.Minute)
}

// 获取会话
func GetSession(ctx context.Context, cache cache.Cache, sessionID string) (interface{}, error) {
    key := cache.KeyPrefixSession + sessionID
    data, err := cache.Get(ctx, key)
    if err != nil {
        return nil, err
    }

    var session SessionData
    json.Unmarshal([]byte(data), &session)
    return &session, nil
}
```

### 场景 3: API 限流

```go
func RateLimit(ctx context.Context, cache cache.Cache, userID string) (bool, error) {
    key := fmt.Sprintf("rate_limit:%s", userID)

    // 获取当前计数
    count, err := cache.Incr(ctx, key)
    if err != nil {
        return false, err
    }

    // 第一次访问，设置过期时间
    if count == 1 {
        cache.Expire(ctx, key, 1*time.Minute)
    }

    // 检查是否超过限制 (每分钟100次)
    return count <= 100, nil
}
```

### 场景 4: 分布式锁

```go
func AcquireLock(ctx context.Context, cache cache.Cache, resource string, ttl time.Duration) (bool, error) {
    key := cache.KeyPrefixLock + resource

    // 尝试设置锁 (NX: 仅当不存在时设置)
    err := cache.Set(ctx, key, "locked", ttl)
    if err != nil {
        return false, nil // 锁已被占用
    }

    return true, nil
}

func ReleaseLock(ctx context.Context, cache cache.Cache, resource string) error {
    key := cache.KeyPrefixLock + resource
    return cache.Delete(ctx, key)
}
```

## 常量定义

### 键前缀常量

```go
cache.KeyPrefixUser      // "user:"
cache.KeyPrefixSession   // "session:"
cache.KeyPrefixCache     // "cache:"
cache.KeyPrefixLock      // "lock:"
cache.KeyPrefixCounter   // "counter:"
```

### 过期时间常量

```go
cache.ExpirationShort   // 5分钟
cache.ExpirationMedium  // 1小时
cache.ExpirationLong    // 24小时
cache.ExpirationNever   // 永不过期
```

## 最佳实践

### 1. 始终设置过期时间

```go
// ✅ 好的做法
err := cache.Set(ctx, "key", "value", 1*time.Hour)

// ❌ 不好的做法 - 永不过期可能导致内存泄漏
err := cache.Set(ctx, "key", "value", 0)
```

### 2. 使用命名空间

```go
// ✅ 使用前缀，避免键冲突
key := cache.KeyPrefixUser + "123"

// ❌ 直接使用数字或简单字符串
key := "123"
```

### 3. 处理缓存穿透

```go
value, err := cache.Get(ctx, key)
if err != nil {
    // 从数据库加载
    data, err := db.Query(...)
    if err != nil {
        return err
    }

    // 即使数据为空，也缓存一个特殊值
    if data == nil {
        cache.Set(ctx, key, "NULL", 5*time.Minute)
    } else {
        cache.Set(ctx, key, data, 1*time.Hour)
    }
}
```

### 4. 错误处理

```go
// 缓存失败不应该导致服务不可用
data, err := cache.Get(ctx, key)
if err != nil {
    log.Warn("cache miss", "key", key, "error", err)
    // 降级到数据库
    data, err = db.Get(id)
}
```

### 5. 批量操作提高性能

```go
// ✅ 批量获取
values, _ := cache.MGet(ctx, "key1", "key2", "key3")

// ❌ 逐个获取
val1, _ := cache.Get(ctx, "key1")
val2, _ := cache.Get(ctx, "key2")
val3, _ := cache.Get(ctx, "key3")
```

## 性能考虑

### 连接池配置

| 应用规模              | PoolSize | MinIdleConns |
| --------------------- | -------- | ------------ |
| 小型 (< 1000 QPS)     | 10       | 5            |
| 中型 (1000-10000 QPS) | 50       | 20           |
| 大型 (> 10000 QPS)    | 100+     | 50+          |

### 值大小

- **推荐**: < 100KB
- **最大**: < 512MB (Redis 限制)
- **大对象**: 考虑压缩或拆分

## 错误处理

```go
value, err := cache.Get(ctx, key)
if err != nil {
    // 检查具体错误类型
    if strings.Contains(err.Error(), "key not found") {
        // 键不存在，从数据库加载
    } else if strings.Contains(err.Error(), "timeout") {
        // 超时，可能需要重试
    } else {
        // 其他错误
        log.Error("cache error", "error", err)
    }
}
```

## 项目结构

```
pkg/cache/
├── constants.go    # 常量定义
├── config.go       # 配置结构
├── cache.go        # Cache 接口定义
├── redis.go        # Redis 实现
├── doc.go          # 包文档
└── README.md       # 本文档
```

## 参考链接

- [Redis 官方文档](https://redis.io/docs/)
- [go-redis/v9](https://github.com/redis/go-redis)
- [Redis 最佳实践](https://redis.io/docs/manual/patterns/)
