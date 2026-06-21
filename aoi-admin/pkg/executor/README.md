# pkg/executor - 协程池任务执行器

提供通用的并发任务调度器,基于高性能的 [ants](https://github.com/panjf2000/ants) 协程池库,支持多池隔离、配置热更新和全链路 panic 恢复。

## API 分类

- 定位：[CONFIRMED] 公共基础设施 API。
- 稳定边界：`Manager`、`Config`、`PoolName`、`NewManager`。
- 当前风险：[CONFIRMED] reload、shutdown、overload 和 panic handler 已有最小包级测试；业务降级策略仍由调用方定义。
- 非目标：[CONFIRMED] 本包不调度业务任务优先级，也不定义业务降级策略。

> HTTP handler 示例只说明异步任务提交位置。当前应用内部 HTTP 边界使用 `pkg/web`，业务模块不要因为示例直接依赖 Gin 类型。

## 特性

- ✅ **多维资源隔离**: 支持多个独立协程池,防止资源争抢
- ✅ **动态热重载**: 运行时原子更新池配置,零停机
- ✅ **全链路安全**: 自动捕获 panic,确保进程不崩溃
- ✅ **非阻塞模式**: 池满时立即返回错误,业务层决定降级策略
- ✅ **线程安全**: 所有操作并发安全
- ✅ **接口化设计**: 便于依赖注入和单元测试

## 快速开始

### 基本使用

```go
package main

import (
    "time"
    "github.com/rei0721/go-scaffold/pkg/executor"
)

func main() {
    // 1. 创建池配置
    configs := []executor.Config{
        {
            Name:        "http",
            Size:        200,
            Expiry:      10 * time.Second,
            NonBlocking: true,
        },
        {
            Name:        "background",
            Size:        50,
            Expiry:      30 * time.Second,
            NonBlocking: false,
        },
    }

    // 2. 创建 executor 管理器
    mgr, err := executor.NewManager(configs)
    if err != nil {
        panic(err)
    }
    defer mgr.Shutdown()

    // 3. 提交异步任务
    err = mgr.Execute("background", func() {
        // 你的业务逻辑
        sendEmail()
    })

    if err == executor.ErrPoolOverload {
        // 池已满,处理过载情况
        log.Warn("pool overloaded")
    }
}
```

## 配置详解

### Config 结构体

```go
type Config struct {
    Name        PoolName       // 池的唯一标识符
    Size        int            // 池容量 (最大并发 worker 数)
    Expiry      time.Duration  // worker 过期时间
    NonBlocking bool           // 是否非阻塞模式
}
```

### 配置参数说明

#### Name (池名称)

池的唯一标识符,用于提交任务时指定目标池。

```go
// ✅ 在业务层定义常量
const (
    PoolNameHTTP       executor.PoolName = "http"
    PoolNameDatabase   executor.PoolName = "database"
    PoolNameBackground executor.PoolName = "background"
)

// 使用常量提交任务
mgr.Execute(PoolNameHTTP, task)
```

**命名建议**:

- 使用小写字母和下划线
- 语义化名称 (描述用途,不是业务细节)
- 避免硬编码字符串

#### Size (池容量)

池中最大并发 worker 数量。

| 任务类型     | 推荐值       | 说明                      |
| ------------ | ------------ | ------------------------- |
| CPU 密集型   | NumCPU() × 2 | 避免过多上下文切换        |
| IO 密集型    | 100-500      | 可以设置较大,根据测试调整 |
| 混合型       | 50-200       | 根据实际负载调整          |
| 低频后台任务 | 10-30        | 节省资源                  |

**示例**:

```go
// CPU 密集型任务
{
    Name: "cpu_intensive",
    Size: runtime.NumCPU() * 2,
}

// IO 密集型任务 (HTTP 请求)
{
    Name: "http",
    Size: 200,
}

// 低频后台任务
{
    Name: "background",
    Size: 30,
}
```

#### Expiry (过期时间)

闲置 worker 的过期时间,超过此时间会被回收。

| 场景     | 推荐值    | 原因                  |
| -------- | --------- | --------------------- |
| 高频任务 | 5-10 秒   | 快速响应,减少创建开销 |
| 中频任务 | 30-60 秒  | 平衡资源和性能        |
| 低频任务 | 60-300 秒 | 根据实际使用频率调整  |

**示例**:

```go
// 高频任务 (每秒多次)
Expiry: 10 * time.Second

// 低频任务 (每分钟几次)
Expiry: 60 * time.Second
```

#### NonBlocking (非阻塞模式)

控制池满时的行为。

| 模式  | 池满时行为               | 适用场景             |
| ----- | ------------------------ | -------------------- |
| true  | 立即返回 ErrPoolOverload | HTTP 服务 (快速失败) |
| false | 阻塞等待                 | CLI 工具 (可以等待)  |

**示例**:

```go
// HTTP 服务 - 非阻塞,快速失败
{
    Name:        "http",
    NonBlocking: true,
}

// CLI 工具 - 阻塞等待
{
    Name:        "cli_tasks",
    NonBlocking: false,
}
```

## API 文档

### Manager 接口

| 方法                             | 说明                  |
| -------------------------------- | --------------------- |
| `Execute(poolName, task) error`  | 提交任务到指定池      |
| `Reload(configs []Config) error` | 热重载所有池配置      |
| `Shutdown()`                     | 优雅关闭,等待任务完成 |

### Execute - 提交任务

```go
func (m *Manager) Execute(poolName PoolName, task func()) error
```

**参数**:

- `poolName`: 池名称,必须是已配置的池
- `task`: 要执行的任务函数 (无参数,无返回值)

**返回值**:

- `nil`: 任务成功提交
- `ErrPoolNotFound`: 池不存在
- `ErrPoolOverload`: 池已满 (仅当 NonBlocking=true)
- `ErrManagerClosed`: 管理器已关闭

**使用示例**:

```go
// 提交任务
err := mgr.Execute("background", func() {
    // 发送邮件
    sendEmail(userID, "Welcome!")
})

// 错误处理
switch err {
case nil:
    // 任务成功提交
case executor.ErrPoolOverload:
    // 池已满,降级处理
    log.Warn("pool overloaded, task rejected")
    // 选项: 重试、同步执行、记录失败等
case executor.ErrPoolNotFound:
    // 池配置错误
    log.Error("pool not found", "name", "background")
case executor.ErrManagerClosed:
    // 管理器已关闭
    log.Warn("executor closed, ignoring task")
}
```

### Reload - 热重载配置

```go
func (m *Manager) Reload(configs []Config) error
```

**原子热重载机制**:

1. ✅ 在锁外创建所有新池
2. ✅ 验证所有池创建成功
3. ✅ 获取写锁 → 原子替换 → 释放写锁
4. ✅ 在锁外优雅关闭旧池
5. ✅ 失败时保持原配置不变

**使用示例**:

```go
// 配置文件变更时
func onConfigChange(newCfg *Config) {
    // 构建新的池配置
    newConfigs := []executor.Config{
        {Name: "http", Size: newCfg.HTTPPoolSize, NonBlocking: true},
        {Name: "background", Size: newCfg.BGPoolSize, NonBlocking: false},
    }

    // 热重载
    if err := mgr.Reload(newConfigs); err != nil {
        log.Error("reload failed", "error", err)
        // 失败时继续使用旧配置
    } else {
        log.Info("executor reloaded successfully")
    }
}
```

**注意事项**:

⚠️ **重要提示**:

- **并发安全**: ✅ `Reload()` 是线程安全的
- **原子性**: ✅ 配置替换是原子操作
- **正在执行的任务**: ✅ 旧池会等待任务完成后才关闭
- **零停机**: ✅ 重载过程中任务提交不会中断

### Shutdown - 优雅关闭

```go
func (m *Manager) Shutdown()
```

**关闭流程**:

1. 标记管理器为已关闭状态
2. 停止接收新任务
3. 等待所有池中的任务完成 (最多 5 秒)
4. 释放所有资源

**使用示例**:

```go
func main() {
    mgr, _ := executor.NewManager(configs)

    // 确保在退出时优雅关闭
    defer mgr.Shutdown()

    // 你的应用逻辑...
}
```

## 使用场景

### 场景 1: HTTP 服务异步任务

```go
// 用户注册处理
func (h *UserHandler) Register(c *gin.Context) {
    // 同步: 创建用户
    user, err := h.service.CreateUser(req)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // 异步: 发送欢迎邮件
    _ = h.executor.Execute("background", func() {
        sendWelcomeEmail(user.Email)
        logRegistrationEvent(user.ID)
    })

    // 立即返回响应,不等待异步任务
    c.JSON(200, user)
}
```

### 场景 2: 批量任务处理

```go
func ProcessOrders(mgr executor.Manager, orders []Order) {
    for _, order := range orders {
        order := order // 捕获循环变量

        err := mgr.Execute("order_processing", func() {
            // 处理订单
            processOrder(order)

            // 更新库存
            updateInventory(order)

            // 通知用户
            notifyUser(order.UserID)
        })

        if err == executor.ErrPoolOverload {
            // 池已满,同步执行
            log.Warn("pool full, processing synchronously", "orderId", order.ID)
            processOrder(order)
        }
    }
}
```

### 场景 3: 降级策略

```go
func SendNotification(mgr executor.Manager, userID int, msg string) error {
    // 尝试异步发送
    err := mgr.Execute("notifications", func() {
        sendPushNotification(userID, msg)
    })

    // 根据错误类型降级
    switch err {
    case nil:
        // 成功提交异步任务
        return nil

    case executor.ErrPoolOverload:
        // 池已满,降级: 放入消息队列
        log.Warn("notification pool overloaded, queuing message")
        return enqueueMessage(userID, msg)

    case executor.ErrManagerClosed:
        // 服务关闭中,同步发送(确保送达)
        log.Warn("executor closed, sending synchronously")
        sendPushNotification(userID, msg)
        return nil

    default:
        return err
    }
}
```

### 场景 4: 与依赖注入集成

```go
// 服务层
type UserService struct {
    repo     UserRepository
    executor executor.Manager
}

func NewUserService(repo UserRepository, exec executor.Manager) *UserService {
    return &UserService{
        repo:     repo,
        executor: exec,
    }
}

func (s *UserService) Register(ctx context.Context, req *RegisterRequest) error {
    // 同步: 创建用户
    user, err := s.repo.Create(ctx, req)
    if err != nil {
        return err
    }

    // 异步: 后续处理
    _ = s.executor.Execute("background", func() {
        sendWelcomeEmail(user.Email)
        createUserProfile(user.ID)
        logAnalyticsEvent("user_registered", user.ID)
    })

    return nil
}
```

## Panic 恢复机制

所有提交的任务都会自动包装 panic 恢复逻辑。

### 自动恢复

```go
// 即使任务 panic,也不会导致进程崩溃
mgr.Execute("http", func() {
    panic("something went wrong")  // ✅ 会被捕获
})

// 池继续正常工作
mgr.Execute("http", func() {
    fmt.Println("still working!")  // ✅ 继续执行
})
```

### 自定义 Panic 处理器 (可选)

```go
// 定义自定义处理器
type MyPanicHandler struct {
    logger logger.Logger
}

func (h *MyPanicHandler) HandlePanic(poolName executor.PoolName, recovered interface{}) {
    h.logger.Error("task panicked",
        "pool", poolName,
        "panic", recovered,
    )
    // 发送告警、记录指标等
}

// 设置全局处理器
executor.SetPanicHandler(&MyPanicHandler{logger: log})
```

## 配置热更新

### 在配置文件中定义

`config.yaml`:

```yaml
executor:
  enabled: true
  pools:
    - name: http
      size: 200
      expiry: 10
      nonBlocking: true

    - name: database
      size: 50
      expiry: 30
      nonBlocking: false

    - name: background
      size: 30
      expiry: 60
      nonBlocking: true
```

### 配置变更自动重载

```go
// 注册配置变更钩子
configManager.RegisterHook(func(old, new *Config) {
    if isExecutorConfigChanged(old, new) {
        // 转换配置格式
        newConfigs := buildExecutorConfigs(new)

        // 热重载
        if err := app.Executor.Reload(newConfigs); err != nil {
            log.Error("executor reload failed", "error", err)
        } else {
            log.Info("executor reloaded", "pools", len(newConfigs))
        }
    }
})
```

## 最佳实践

### 1. 使用常量定义池名称

```go
// ✅ 在业务层定义常量
package constants

const (
    PoolHTTP       executor.PoolName = "http"
    PoolDatabase   executor.PoolName = "database"
    PoolBackground executor.PoolName = "background"
)

// 使用
mgr.Execute(constants.PoolBackground, task)

// ❌ 不要使用硬编码字符串
mgr.Execute("background", task)  // 容易拼写错误
```

### 2. 处理所有错误

```go
// ✅ 处理所有可能的错误
err := mgr.Execute("http", task)
switch err {
case nil:
    // 成功
case executor.ErrPoolOverload:
    // 过载处理
case executor.ErrPoolNotFound:
    // 配置错误
case executor.ErrManagerClosed:
    // 关闭中
}

// ❌ 不要忽略错误
_ = mgr.Execute("http", task)
```

### 3. 避免在任务中捕获外部变量

```go
// ✅ 显式传递参数
for _, user := range users {
    user := user  // 捕获循环变量
    mgr.Execute("background", func() {
        processUser(user)
    })
}

// ❌ 不要直接使用循环变量
for _, user := range users {
    mgr.Execute("background", func() {
        processUser(user)  // 可能使用错误的 user
    })
}
```

### 4. 合理设置池大小

| 池类型        | 推荐配置                                      |
| ------------- | --------------------------------------------- |
| HTTP 请求处理 | Size: 200, NonBlocking: true                  |
| 数据库操作    | Size: 50, NonBlocking: false                  |
| 后台任务      | Size: 30, NonBlocking: true                   |
| 邮件发送      | Size: 10, NonBlocking: true                   |
| 文件处理      | Size: runtime.NumCPU() × 2, NonBlocking: true |

### 5. 确保优雅关闭

```go
// ✅ 使用 defer 确保关闭
func main() {
    mgr, _ := executor.NewManager(configs)
    defer mgr.Shutdown()  // ✅ 在退出时优雅关闭

    // 应用逻辑...
}
```

### 6. 任务应该是幂等的

```go
// ✅ 幂等任务
mgr.Execute("background", func() {
    // 即使执行多次也安全
    cache.Set(key, value)
    updateUserLastSeen(userID, time.Now())
})

// ⚠️ 非幂等任务需要额外处理
mgr.Execute("background", func() {
    // 不是幂等的,需要去重机制
    incrementPageViewCount(articleID)
})
```

## 性能考虑

### 性能特点

- **零分配**: 使用对象池,减少 GC 压力
- **高并发**: 基于 ants,单机支持百万级 goroutine
- **低开销**: Execute 操作极快 (~100ns)

### 性能优化建议

1. **合理设置池大小**

   ```go
   // 根据系统资源和负载测试确定
   Size: 200  // 不是越大越好
   ```

2. **避免过度使用**

   ```go
   // ✅ 适合异步任务
   mgr.Execute("background", sendEmail)

   // ❌ 不适合: 极简单任务
   mgr.Execute("background", func() {
       i++  // 开销大于收益
   })
   ```

3. **设置合理的 Expiry**

   ```go
   // 高频任务: 短 expiry
   Expiry: 10 * time.Second

   // 低频任务: 长 expiry
   Expiry: 60 * time.Second
   ```

## 常见问题

### 问：如何选择阻塞/非阻塞模式？

**非阻塞模式 (推荐)**:

- HTTP/RPC 服务: 快速失败,返回 503
- 高流量场景: 防止雪崩
- 需要降级策略: 业务层决定如何处理

**阻塞模式**:

- CLI 工具: 可以等待资源
- 批处理任务: 必须全部完成
- 低频场景: 不担心阻塞

### 问：任务 panic 会发生什么？

所有任务都自动包装 panic 恢复:

1. Panic 被捕获并记录
2. 进程不会崩溃
3. 池继续正常工作
4. 可以通过 `SetPanicHandler` 自定义处理

### 问：Reload 会丢失正在执行的任务吗？

不会。重载流程保证:

1. ✅ 新任务提交到新池
2. ✅ 旧池等待任务完成 (最多 5 秒)
3. ✅ 所有任务都会执行完成

### 问：如何监控池的状态？

目前版本暂无内置指标,未来版本会添加:

```go
// 计划中的 API
stats := mgr.Stats("http")
// stats.Running  // 正在运行的任务数
// stats.Free     // 空闲 worker 数
// stats.Cap      // 池容量
```

现在可以通过日志记录提交失败:

```go
if err == executor.ErrPoolOverload {
    metrics.Inc("executor.pool.overload", "pool", poolName)
}
```

## 项目结构

```
pkg/executor/
├── constants.go    # 错误常量和默认值
├── executor.go     # Manager 接口和 Config 定义
├── manager.go      # Manager 实现 (原子重载)
├── pool.go         # poolWrapper (ants 包装器)
├── doc.go          # Go doc 文档
└── README.md       # 本文档
```

## 依赖项

- [github.com/panjf2000/ants/v2](https://github.com/panjf2000/ants) - 高性能协程池

## 相关资源

- [ants 官方文档](https://pkg.go.dev/github.com/panjf2000/ants/v2)
- [Go 并发模式](https://go.dev/blog/pipelines)

## 许可证

本项目使用 MIT 许可证。
