# pkg/logger - 结构化日志封装

提供统一的日志接口,基于高性能的 [Zap](https://github.com/uber-go/zap) 日志库,支持结构化日志、配置热更新和多种输出格式。

## API 分类

- 定位：[CONFIRMED] 公共基础设施 API。
- 稳定边界：`Logger`、`Reloader`、`Config`、`New`、`Default`。
- 当前风险：[RISK] 文件输出和轮转路径测试覆盖有限。
- 非目标：[CONFIRMED] 本包不定义业务审计字段或日志采集后端。

> README 中的 Gin middleware 片段是包级独立集成示例。当前应用内部 HTTP 边界使用 `pkg/web` 和 `internal/middleware`，业务模块不要直接依赖 Gin 类型。

## 特性

- ✅ **高性能**: 基于 Zap,比标准库快 10 倍以上
- ✅ **结构化日志**: 支持键值对,便于日志分析
- ✅ **配置热更新**: 支持运行时动态更新日志配置
- ✅ **多种格式**: JSON 和 Console 格式
- ✅ **多种输出**: stdout 和文件输出
- ✅ **日志轮转**: 自动按大小、数量、时间轮转
- ✅ **线程安全**: 所有操作都是并发安全的
- ✅ **上下文支持**: With() 方法添加持久字段

## 快速开始

### 基本使用

```go
package main

import (
    "github.com/rei0721/go-scaffold/pkg/logger"
)

func main() {
    // 1. 创建配置
    cfg := &logger.Config{
        Level:  "info",
        Format: "console",
        Output: "stdout",
    }

    // 2. 创建 logger
    log, err := logger.New(cfg)
    if err != nil {
        panic(err)
    }
    defer log.Sync()

    // 3. 记录日志
    log.Info("application started", "version", "1.0.0", "port", 8080)
    log.Error("database connection failed", "error", err)
}
```

### 使用默认 Logger

```go
// 快速开始,使用默认配置
log := logger.Default()
defer log.Sync()

log.Debug("debug message")
log.Info("info message")
log.Warn("warning message")
log.Error("error message")
```

## 配置详解

### Config 结构体

```go
type Config struct {
    Level      string  // 日志级别: debug, info, warn, error
    Format     string  // 输出格式: json, console
    Output     string  // 输出目标: stdout, file
    FilePath   string  // 日志文件路径 (Output="file" 时)
    MaxSize    int     // 单个文件最大大小 (MB)
    MaxBackups int     // 保留的旧文件最大数量
    MaxAge     int     // 保留旧文件的最大天数
}
```

### 日志级别

| 级别    | 包含的日志               | 使用场景     |
| ------- | ------------------------ | ------------ |
| `debug` | debug, info, warn, error | 开发环境     |
| `info`  | info, warn, error        | 生产环境默认 |
| `warn`  | warn, error              | 生产环境优化 |
| `error` | error                    | 仅记录错误   |

**示例**:

```go
// 开发环境 - 详细日志
cfg := &logger.Config{
    Level: "debug",
}

// 生产环境 - 关键信息
cfg := &logger.Config{
    Level: "info",
}
```

### 输出格式

#### JSON 格式 (生产环境推荐)

```go
cfg := &logger.Config{
    Format: "json",
}
```

输出示例:

```json
{
  "level": "info",
  "time": "2024-01-11T07:00:00+08:00",
  "caller": "main.go:42",
  "message": "user created",
  "userId": 123,
  "username": "alice"
}
```

**适用场景**:

- 日志收集系统 (ELK, Splunk)
- 生产环境
- 机器解析

#### Console 格式 (开发环境推荐)

```go
cfg := &logger.Config{
    Format: "console",
}
```

输出示例:

```
2024-01-11T07:00:00+08:00	info	main.go:42	user created	userId=123	username=alice
```

**适用场景**:

- 开发环境
- 人类阅读
- 终端输出

### 输出目标

#### 标准输出 (容器环境推荐)

```go
cfg := &logger.Config{
    Output: "stdout",
}
```

**适用场景**:

- Docker/K8s 容器环境
- 日志收集器捕获
- 开发环境调试

#### 文件输出 (传统部署推荐)

```go
cfg := &logger.Config{
    Output:     "file",
    FilePath:   "/var/log/app/app.log",
    MaxSize:    100,    // 100MB
    MaxBackups: 10,     // 保留 10 个旧文件
    MaxAge:     30,     // 保留 30 天
}
```

**日志轮转规则**:

1. **按大小**: 超过 `MaxSize` MB 自动创建新文件
2. **按数量**: 超过 `MaxBackups` 数量删除最旧文件
3. **按时间**: 超过 `MaxAge` 天删除旧文件
4. **自动压缩**: 旧文件自动 gzip 压缩

## API 文档

### 日志方法

| 方法                                 | 级别  | 说明                         |
| ------------------------------------ | ----- | ---------------------------- |
| `Debug(msg, keysAndValues...)`       | DEBUG | 详细调试信息                 |
| `Info(msg, keysAndValues...)`        | INFO  | 重要事件                     |
| `Warn(msg, keysAndValues...)`        | WARN  | 警告信息                     |
| `Error(msg, keysAndValues...)`       | ERROR | 错误信息                     |
| `Fatal(msg, keysAndValues...)`       | FATAL | 致命错误,会调用 os.Exit(1)   |
| `With(keysAndValues...) Logger`      | -     | 返回带上下文的子 logger      |
| `Sync() error`                       | -     | 刷新缓冲的日志               |
| `Reload(cfg *Config) error`          | -     | 热更新配置                   |
| `SetExecutor(exec executor.Manager)` | -     | 设置协程池管理器（延迟注入） |

### 使用示例

#### 结构化日志

```go
// 使用键值对记录结构化信息
log.Info("user login",
    "userId", 12345,
    "username", "alice",
    "ip", "192.168.1.100",
    "timestamp", time.Now(),
)
```

#### With() 上下文

```go
// 创建带有持久字段的子 logger
requestLog := log.With(
    "requestId", "abc-123",
    "userId", 456,
    "path", "/api/users",
)

// 所有日志自动包含上下文
requestLog.Info("processing request")     // 自动包含 requestId, userId, path
requestLog.Error("request failed", "error", err)  // 也包含上下文
```

#### 错误处理

```go
if err := db.Connect(); err != nil {
    log.Error("database connection failed",
        "error", err,
        "host", "localhost",
        "port", 5432,
    )
    return err
}
```

## 协程池集成 (SetExecutor)

Logger 支持协程池集成，用于异步执行某些日志操作（如 Sync 刷新），提高性能。

### 接口说明

```go
SetExecutor(exec executor.Manager)
```

**参数**：

- `exec` (executor.Manager) - 协程池管理器实例，为 nil 时禁用异步功能

**用途**：

- 异步执行日志刷新操作（Sync）
- 避免阻塞主流程
- 提高系统吞吐量

**线程安全**：

- 使用原子操作保证并发安全
- 可以在运行时动态设置

### 使用示例

#### 基本使用

```go
import (
    "github.com/rei0721/go-scaffold/pkg/logger"
    "github.com/rei0721/go-scaffold/pkg/executor"
)

func main() {
    // 1. 创建 logger
    log, _ := logger.New(&logger.Config{
        Level:  "info",
        Format: "json",
        Output: "stdout",
    })

    // 2. 创建 executor
    executorMgr, _ := executor.NewManager([]executor.Config{
        {Name: "logger", Size: 10, NonBlocking: true},
    })

    // 3. 注入 executor 到 logger
    log.SetExecutor(executorMgr)

    // 4. 现在 Sync() 等操作可以异步执行
    defer log.Sync()

    // 正常使用 logger
    log.Info("application started")
}
```

#### 应用初始化模式

在应用初始化时统一设置：

```go
// internal/app/app.go
func (app *App) Init() error {
    // 创建所有组件
    app.Logger = createLogger()
    app.HTTPServer = createHTTPServer()
    app.Cache = createCache()
    app.Executor = createExecutor()

    // 延迟注入 Executor（统一管理异步任务）
    app.Logger.SetExecutor(app.Executor)
    app.HTTPServer.SetExecutor(app.Executor)
    // ... 其他需要异步的组件

    return nil
}
```

#### 动态设置

```go
// 运行时动态设置或移除 executor
log.SetExecutor(executorMgr)  // 启用异步

// 禁用异步（传 nil）
log.SetExecutor(nil)
```

### 使用场景

1. **高并发环境**：避免日志刷新阻塞业务逻辑
2. **性能优化**：异步执行耗时的 I/O 操作
3. **微服务**：统一管理所有组件的异步任务

### 注意事项

⚠️ **重要提示**：

- SetExecutor 是**可选的**，不设置也能正常工作
- 设置后，某些操作（如 Sync）会异步执行
- 可以传 nil 禁用 executor 功能
- 建议在应用初始化时统一设置
- 确保在应用关闭前调用 `executor.Shutdown()` 等待任务完成

## 配置热更新 (Reload)

支持运行时动态更新日志配置,无需重启应用。

### 使用场景

- 配置文件变更时自动重载
- 动态调整日志级别 (生产环境降低噪音)
- 切换日志输出目标
- 修改日志格式

### 使用方法

```go
// 创建初始 logger
log, _ := logger.New(&logger.Config{
    Level:  "info",
    Format: "console",
    Output: "stdout",
})

// 监听配置变更
go func() {
    for newCfg := range configChangeChannel {
        // 热更新日志配置
        if err := log.Reload(newCfg); err != nil {
            log.Error("failed to reload logger", "error", err)
            // 重载失败,继续使用旧配置
        } else {
            log.Info("logger configuration reloaded successfully")
        }
    }
}()
```

### 重载机制说明

`Reload()` 方法的执行流程:

1. ✅ **验证新配置**: 使用新配置创建 logger
2. ✅ **原子替换**: 将新 logger 替换旧 logger
3. ✅ **同步旧 logger**: 刷新旧 logger 缓冲
4. ✅ **失败保护**: 如果失败,保持原配置不变
5. ✅ **线程安全**: 使用读写锁保护,并发安全

### 注意事项

⚠️ **重要提示:**

- **子 logger**: 使用 `With()` 创建的子 logger 不会自动重载,需要重新创建
- **线程安全**: ✅ `Reload()` 是线程安全的
- **零停机**: ✅ 重载过程中日志记录不会中断
- **原子性**: ✅ 配置替换是原子操作

## 使用场景

### 场景 1: Web 应用日志

```go
func main() {
    // 创建全局 logger
    log, _ := logger.New(&logger.Config{
        Level:  "info",
        Format: "json",        // 生产环境使用 JSON
        Output: "stdout",      // K8s 环境输出到 stdout
    })
    defer log.Sync()

    // HTTP 中间件
    router.Use(func(c *gin.Context) {
        start := time.Now()

        // 创建请求专用 logger
        reqLog := log.With(
            "requestId", c.GetHeader("X-Request-ID"),
            "method", c.Request.Method,
            "path", c.Request.URL.Path,
        )

        c.Set("logger", reqLog)
        c.Next()

        reqLog.Info("request completed",
            "status", c.Writer.Status(),
            "duration", time.Since(start),
        )
    })
}
```

### 场景 2: 错误追踪

```go
func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    log := ctx.Value("logger").(logger.Logger)

    log.Info("creating user", "username", user.Name)

    if err := s.repo.Create(user); err != nil {
        log.Error("failed to create user",
            "error", err,
            "username", user.Name,
            "email", user.Email,
        )
        return err
    }

    log.Info("user created successfully", "userId", user.ID)
    return nil
}
```

### 场景 3: 定时任务日志

```go
func RunJob(log logger.Logger) {
    jobLog := log.With("job", "data-sync", "run", time.Now())

    jobLog.Info("job started")

    if err := syncData(); err != nil {
        jobLog.Error("job failed", "error", err)
        return
    }

    jobLog.Info("job completed")
}
```

### 场景 4: 多环境配置

```go
func NewLogger(env string) (logger.Logger, error) {
    var cfg *logger.Config

    switch env {
    case "development":
        cfg = &logger.Config{
            Level:  "debug",
            Format: "console",
            Output: "stdout",
        }
    case "production":
        cfg = &logger.Config{
            Level:  "info",
            Format: "json",
            Output: "file",
            FilePath: "/var/log/app/app.log",
            MaxSize: 100,
            MaxBackups: 10,
            MaxAge: 30,
        }
    }

    return logger.New(cfg)
}
```

## 最佳实践

### 1. 始终调用 Sync()

```go
// ✅ 确保日志被刷新到磁盘
log, _ := logger.New(cfg)
defer log.Sync()
```

### 2. 使用结构化日志

```go
// ✅ 使用键值对
log.Info("user created", "userId", 123, "username", "alice")

// ❌ 不要使用字符串拼接
log.Info("user created: userId=123, username=alice")
```

### 3. 合理使用 With()

```go
// ✅ 为一组操作创建带上下文的 logger
userLog := log.With("userId", 123, "module", "user-service")
userLog.Info("querying database")
userLog.Info("updating cache")

// ❌ 不要每次都重复传递相同字段
log.Info("querying database", "userId", 123, "module", "user-service")
log.Info("updating cache", "userId", 123, "module", "user-service")
```

### 4. 避免在循环中记录过多日志

```go
// ✅ 批量汇总
processedCount := 0
for _, item := range items {
    process(item)
    processedCount++
}
log.Info("batch processing completed", "count", processedCount)

// ❌ 每个项都记录
for _, item := range items {
    log.Info("processing item", "item", item)  // 可能产生数千条日志
}
```

### 5. 生产环境使用 JSON 格式

```go
// ✅ 生产环境
cfg := &logger.Config{
    Level:  "info",
    Format: "json",    // 便于日志分析
}

// ✅ 开发环境
cfg := &logger.Config{
    Level:  "debug",
    Format: "console",  // 易于阅读
}
```

### 6. 合理设置日志级别

| 环境   | 推荐级别 | 原因                  |
| ------ | -------- | --------------------- |
| 开发   | debug    | 需要详细信息调试      |
| 测试   | info     | 关注主要流程          |
| 预发布 | info     | 与生产环境保持一致    |
| 生产   | info     | 平衡信息量和性能      |
| 生产   | warn     | 高流量场景,减少日志量 |

### 7. 日志文件配置建议

```go
cfg := &logger.Config{
    Output:     "file",
    FilePath:   "/var/log/app/app.log",
    MaxSize:    100,    // 100MB - 适合大多数应用
    MaxBackups: 10,     // 保留 10 个文件 = ~1GB
    MaxAge:     30,     // 30 天 - 满足审计要求
}
```

## 性能考虑

### 性能对比

| 操作                   | 标准库 log  | Zap (本包) |
| ---------------------- | ----------- | ---------- |
| 简单日志               | 3000 ns/op  | 300 ns/op  |
| 结构化日志 (10 个字段) | 15000 ns/op | 800 ns/op  |
| 内存分配               | 多次        | 零分配     |

### 性能优化技巧

1. **使用 Info 而不是 Debug (生产环境)**

   ```go
   cfg.Level = "info"  // 跳过所有 Debug 日志
   ```

2. **避免复杂的日志值计算**

   ```go
   // ✅ 只在需要时计算
   if log.Level == "debug" {
       log.Debug("complex data", "data", expensiveFunction())
   }

   // ❌ 总是计算
   log.Debug("complex data", "data", expensiveFunction())  // 即使 level=info 也会计算
   ```

3. **批量操作使用单条汇总日志**

   ```go
   // ✅ 一条汇总日志
   log.Info("processed items", "count", 1000, "duration", elapsed)

   // ❌ 1000 条日志
   for i := 0; i < 1000; i++ {
       log.Debug("processing item", "index", i)
   }
   ```

## 常见问题

### 问：如何在 Gin 中使用？

```go
func LoggerMiddleware(log logger.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        reqLog := log.With(
            "method", c.Request.Method,
            "path", c.Request.URL.Path,
            "ip", c.ClientIP(),
        )

        c.Set("logger", reqLog)
        c.Next()

        reqLog.Info("request",
            "status", c.Writer.Status(),
            "duration", time.Since(start),
        )
    }
}

// 使用
router.Use(LoggerMiddleware(log))
```

### 问：如何禁用 stdout 的 sync 错误？

```go
// Sync() 可能在某些平台返回无害错误
// 例如: "sync /dev/stdout: invalid argument"
if err := log.Sync(); err != nil {
    // 忽略 stdout/stderr 的 sync 错误
    if !strings.Contains(err.Error(), "invalid argument") {
        log.Error("failed to sync logger", "error", err)
    }
}
```

### 问：Fatal 和 Error 的区别？

- `Error()`: 记录错误,程序继续运行
- `Fatal()`: 记录错误,调用 `os.Exit(1)` 终止程序

```go
// ✅ 可恢复错误使用 Error
if err := db.Query(); err != nil {
    log.Error("query failed", "error", err)
    return err
}

// ✅ 不可恢复错误使用 Fatal
if err := loadConfig(); err != nil {
    log.Fatal("failed to load config", "error", err)
    // 程序在这里退出
}
```

## 项目结构

```
pkg/logger/
├── constants.go    # 常量定义 (默认级别、格式、输出)
├── logger.go       # Logger 和 Reloader 接口定义
├── zap.go          # Zap 实现
├── zap_test.go     # 单元测试 (包含并发测试)
└── README.md       # 本文档
```

## 测试

```bash
# 运行所有测试
go test ./pkg/logger/...

# 运行特定测试
go test ./pkg/logger/... -run TestReload

# 查看覆盖率
go test ./pkg/logger/... -cover
```

## 依赖项

- [go.uber.org/zap](https://github.com/uber-go/zap) - 高性能日志库
- [gopkg.in/natefinch/lumberjack.v2](https://github.com/natefinch/lumberjack) - 日志轮转

## 相关资源

- [Zap 官方文档](https://pkg.go.dev/go.uber.org/zap)
- [结构化日志最佳实践](https://github.com/uber-go/guide/blob/master/style.md#logging)
- [Lumberjack 文档](https://pkg.go.dev/gopkg.in/natefinch/lumberjack.v2)

## 许可证

本项目使用 MIT 许可证。
