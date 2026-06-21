# pkg/database - 数据库基础设施

提供统一的数据库抽象层,支持 PostgreSQL、MySQL 和 SQLite,并提供连接池管理和配置热更新功能。

## API 分类

- 定位：[CONFIRMED] 公共基础设施 API。
- 稳定边界：`Database`、`Executor`、`QueryOption`、`Result`、`ErrNotFound`、`ErrSQLDBUnavailable`、`Reloader`、事务接口、`SQLDB`、`Config`、`New`、`NewWithHooks`。
- 当前风险：[RISK] Hook、Reload 和多驱动路径测试覆盖仍有限。
- 非目标：[CONFIRMED] 本包不定义业务模型或生产迁移流程。

## 防腐层边界

`pkg/database` 是项目唯一允许直接依赖 GORM 和数据库驱动的数据库基础设施包。`internal/` 层只能依赖本包暴露的项目自有类型,例如 `database.Database`、`database.Executor`、`database.QueryOption` 和 `database.Result`。

本包对外不暴露 `*gorm.DB`、`gorm.ErrRecordNotFound` 或 GORM 查询构造器。仓储层需要通过 `Create`、`First`、`Find`、`Update`、`Delete`、`Exec`、`Raw`、`Count`、`HasTable` 等方法完成访问,并通过 `database.ErrNotFound` 处理未命中记录。

`SQLDB()` 是迁移层专用边界，用于把标准 `*sql.DB` 交给 `pkg/migrator`。业务 repository 不应绕过 `Executor` 直接使用标准库连接。

事务通过 `WithTx(ctx, func(ctx context.Context, tx database.Executor) error { ... })` 注入同一个 `Executor`,从而保留事务语义,同时阻断底层 ORM 类型向业务层泄漏。

## 特性

- ✅ **多数据库支持**: PostgreSQL、MySQL、SQLite
- ✅ **连接池管理**: 自动管理连接复用,提高性能
- ✅ **配置热更新**: 支持运行时动态更新数据库配置
- ✅ **健康检查**: 内置 Ping 方法验证连接状态
- ✅ **Hook 支持**: 可扩展的回调机制
- ✅ **接口抽象**: 便于测试和切换实现
- ✅ **迁移边界**: 向 `pkg/migrator` 提供标准 `*sql.DB`,同时继续隐藏 GORM
- ✅ **防腐封装**: 隐藏 GORM 细节,为 `internal/` 提供稳定项目接口

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/rei0721/go-scaffold/pkg/database"
)

func main() {
    ctx := context.Background()

    // 1. 配置数据库
    cfg := &database.Config{
        Driver:       database.DriverPostgres,
        Host:         "localhost",
        Port:         5432,
        User:         "postgres",
        Password:     "your_password",
        DBName:       "myapp",
        SSLMode:      "disable",
        MaxOpenConns: 25,
        MaxIdleConns: 10,
        MaxLifetime:  15 * time.Minute,
    }

    // 2. 创建数据库连接
    db, err := database.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // 3. 健康检查
    if err := db.Ping(ctx); err != nil {
        log.Fatal("database connection failed:", err)
    }

    // 4. 使用项目自有数据库接口进行操作
    var users []User
    if err := db.Find(ctx, &users, database.Order("id DESC"), database.Limit(20)); err != nil {
        log.Fatal(err)
    }
}

type User struct {
    ID   uint   `gorm:"primaryKey"`
    Name string `gorm:"size:100"`
}
```

### 使用不同的数据库

#### MySQL

```go
cfg := &database.Config{
    Driver:       database.DriverMySQL,
    Host:         "localhost",
    Port:         3306,
    User:         "root",
    Password:     "your_password",
    DBName:       "myapp",
    MaxOpenConns: 25,
    MaxIdleConns: 10,
    MaxLifetime:  15 * time.Minute,
}

db, err := database.New(cfg)
```

#### SQLite

```go
cfg := &database.Config{
    Driver:       database.DriverSQLite,
    DBName:       "./data/app.db", // 文件路径
    MaxOpenConns: 1,                // SQLite 建议单连接
    MaxIdleConns: 1,
    MaxLifetime:  time.Hour,
}

db, err := database.New(cfg)
```

## 数据访问接口

仓储层依赖 `database.Executor`,这样普通连接和事务连接可以使用同一组方法。

```go
type TodoRepository struct {
    db database.Executor
}

func NewTodoRepository(db database.Executor) *TodoRepository {
    return &TodoRepository{db: db}
}

func (r *TodoRepository) Create(ctx context.Context, todo *Todo) error {
    return r.db.Create(ctx, todo)
}

func (r *TodoRepository) MarkCreated(ctx context.Context, id uint) error {
    _, err := r.db.Update(
        ctx,
        &Todo{},
        map[string]any{"status": "created"},
        database.Where("id = ?", id),
    )
    return err
}

func (r *TodoRepository) FindActive(ctx context.Context, limit int) ([]Todo, error) {
    var todos []Todo
    err := r.db.Find(
        ctx,
        &todos,
        database.Where("deleted_at IS NULL"),
        database.Order("id DESC"),
        database.Limit(limit),
    )
    if err != nil {
        return nil, err
    }
    return todos, nil
}

func (r *TodoRepository) FindByID(ctx context.Context, id uint) (*Todo, error) {
    var todo Todo
    if err := r.db.First(ctx, &todo, database.Where("id = ? AND deleted_at IS NULL", id)); err != nil {
        return nil, err
    }
    return &todo, nil
}
```

`First` 未命中时会返回 `database.ErrNotFound`,调用方不需要识别 `gorm.ErrRecordNotFound`。

```go
todo, err := repo.FindByID(ctx, id)
if errors.Is(err, database.ErrNotFound) {
    return nil, ErrTodoNotFound
}
```

## 事务

事务入口位于 `database.Database`,事务体收到的是 `database.Executor`,不会暴露底层 ORM 事务对象。

```go
err := db.WithTx(ctx, func(txCtx context.Context, tx database.Executor) error {
    repo := NewTodoRepository(tx)

    if err := repo.Create(txCtx, todo); err != nil {
        return err
    }
    return repo.MarkCreated(txCtx, todo.ID)
})
```

如果仓储方法被事务调用,它只需要继续使用传入的 `Executor`;如果没有事务,传入 `database.Database` 本身即可。

## 配置详解

### Config 结构体

| 字段           | 类型            | 说明              | PostgreSQL | MySQL     | SQLite |
| -------------- | --------------- | ----------------- | ---------- | --------- | ------ |
| `Driver`       | `Driver`        | 数据库驱动类型    | ✅         | ✅        | ✅     |
| `Host`         | `string`        | 服务器地址        | ✅         | ✅        | ❌     |
| `Port`         | `int`           | 端口号            | ✅ (5432)  | ✅ (3306) | ❌     |
| `User`         | `string`        | 用户名            | ✅         | ✅        | ❌     |
| `Password`     | `string`        | 密码              | ✅         | ✅        | ❌     |
| `DBName`       | `string`        | 数据库名/文件路径 | ✅         | ✅        | ✅     |
| `SSLMode`      | `string`        | SSL 连接模式      | ✅         | ⚠️        | ❌     |
| `MaxOpenConns` | `int`           | 最大连接数        | ✅         | ✅        | ✅     |
| `MaxIdleConns` | `int`           | 最大空闲连接数    | ✅         | ✅        | ✅     |
| `MaxLifetime`  | `time.Duration` | 连接最大生命周期  | ✅         | ✅        | ✅     |

### SSL 模式说明

#### PostgreSQL

- `disable`: 禁用 SSL(默认,开发环境)
- `require`: 需要 SSL,但不验证证书
- `verify-ca`: 验证证书颁发机构
- `verify-full`: 验证证书和主机名(生产环境推荐)

#### MySQL

- `true`: 启用 SSL
- `false`: 禁用 SSL
- `skip-verify`: 启用 SSL 但不验证证书
- `preferred`: 优先使用 SSL

## 连接池最佳实践

### 参数调优指南

```go
cfg := &database.Config{
    // 根据应用并发量设置:
    // - 低并发(< 100 QPS): 10-25
    // - 中并发(100-1000 QPS): 25-50
    // - 高并发(> 1000 QPS): 50-100
    MaxOpenConns: 50,

    // 建议设置为 MaxOpenConns 的 50%-100%
    // 保持足够的空闲连接避免频繁创建/销毁
    MaxIdleConns: 25,

    // 连接最大生命周期
    // 推荐: 5-30 分钟
    // 原因: 定期刷新连接,避免数据库端超时
    MaxLifetime: 15 * time.Minute,
}
```

### 连接池大小计算公式

```
MaxOpenConns = ((核心数 * 2) + 有效磁盘数)
```

例如:

- 4 核 CPU,1 个磁盘: `(4 * 2) + 1 = 9` → 推荐 10-15
- 8 核 CPU,2 个磁盘: `(8 * 2) + 2 = 18` → 推荐 20-25

### 监控指标

为了保持防腐层边界,当前公共 API 不返回底层 `*sql.DB` 或 `*gorm.DB`。调用侧可以通过 `Ping(ctx)` 做健康探测,连接池大小由 `Config` 统一管理。

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

if err := db.Ping(ctx); err != nil {
    log.Printf("database health check failed: %v", err)
}
```

如果后续需要公开连接池统计,应新增项目自有的 `database.Stats` 值对象,不要把底层驱动对象返回给上层。

## 配置热更新 (Reload)

支持运行时动态更新数据库配置,无需重启应用。

### 使用场景

- 配置文件变更时自动重载
- 动态调整连接池参数
- 切换数据库端点
- 更新 SSL/TLS 配置

### 使用方法

```go
// 创建初始连接
db, err := database.New(cfg)
if err != nil {
    log.Fatal(err)
}

// 监听配置变更
go func() {
    for newCfg := range configChangeChannel {
        // 热更新数据库配置
        if err := db.Reload(newCfg); err != nil {
            log.Printf("failed to reload database: %v", err)
            // 重载失败,继续使用旧配置
        } else {
            log.Println("database configuration reloaded successfully")
        }
    }
}()
```

### 重载机制说明

`Reload()` 方法的执行流程:

1. ✅ **验证新配置**: 使用新配置创建数据库连接
2. ✅ **Ping 测试**: 确保新连接可用
3. ✅ **原子替换**: 将新连接替换旧连接
4. ✅ **优雅关闭**: 关闭旧连接池
5. ✅ **失败保护**: 如果失败,保持原有连接不变

```go
// Reload 的内部逻辑
func (d *database) Reload(cfg *Config) error {
    ctx := context.Background()

    // 1. 创建新连接
    newDB, err := New(cfg)
    if err != nil {
        return err // 保持原连接
    }

    // 2. 验证新连接
    if err := newDB.Ping(ctx); err != nil {
        newDB.Close()
        return err // 保持原连接
    }

    // 3. 在包内原子替换底层连接,不向调用方暴露驱动对象
    // 4. 关闭旧连接

    return nil
}
```

### 注意事项

⚠️ **重要提示:**

- **进行中的查询**: 重载时可能有查询正在使用旧连接,`sql.DB` 会安全处理
- **失败回退**: 如果新连接创建或验证失败,自动保持原连接
- **Hooks 不重载**: 当前实现不会重新注册 hooks,hooks 在初始化时注册
- **线程安全**: ✅ `Reload()` 方法是线程安全的,使用读写锁保护并发访问
- **原子性**: ✅ 连接替换操作是原子的,不会出现中间状态

## Hooks 扩展

使用 Hooks 在数据库操作前后执行自定义逻辑。

### 实现 Hook 接口

```go
type AuditHook struct {
    logger *log.Logger
}

func (h *AuditHook) BeforeCreate(ctx context.Context, tx database.Executor) {
    h.logger.Println("Creating record...")
}

func (h *AuditHook) AfterCreate(ctx context.Context, tx database.Executor) {
    h.logger.Println("Record created")
}

func (h *AuditHook) BeforeQuery(ctx context.Context, tx database.Executor) {
    h.logger.Println("Querying...")
}

func (h *AuditHook) AfterQuery(ctx context.Context, tx database.Executor) {
    h.logger.Println("Query completed")
}
```

### 注册 Hooks

```go
auditHook := &AuditHook{logger: log.Default()}

db, err := database.NewWithHooks(cfg, auditHook)
if err != nil {
    log.Fatal(err)
}
```

### Hook 使用场景

- 📊 **审计日志**: 记录所有数据变更
- ⏱️ **性能监控**: 统计查询执行时间
- ✅ **数据验证**: 在保存前验证数据
- 🔐 **权限控制**: 添加租户隔离条件
- 🕒 **自动填充**: 自动设置 `created_at`、`updated_at` 等字段

## 健康检查

### HTTP 健康检查端点

```go
func healthCheckHandler(db database.Database) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if err := db.Ping(r.Context()); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            json.NewEncoder(w).Encode(map[string]string{
                "status": "unhealthy",
                "error":  err.Error(),
            })
            return
        }

        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "healthy",
        })
    }
}
```

### 定期健康检查

```go
func periodicHealthCheck(db database.Database, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for range ticker.C {
        if err := db.Ping(context.Background()); err != nil {
            log.Printf("database health check failed: %v", err)
            // 发送告警...
        }
    }
}

// 使用
go periodicHealthCheck(db, 30*time.Second)
```

## 完整示例

### Web 应用集成

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/rei0721/go-scaffold/pkg/database"
    "github.com/rei0721/go-scaffold/pkg/web"
)

func main() {
    // 1. 配置数据库
    cfg := &database.Config{
        Driver:       database.DriverPostgres,
        Host:         os.Getenv("RIN_APP_DB_POSTGRES_HOST"),
        Port:         5432,
        User:         os.Getenv("RIN_APP_DB_POSTGRES_USERNAME"),
        Password:     os.Getenv("RIN_APP_DB_POSTGRES_PASSWORD"),
        DBName:       os.Getenv("RIN_APP_DB_POSTGRES_DATABASE"),
        SSLMode:      "require",
        MaxOpenConns: 50,
        MaxIdleConns: 25,
        MaxLifetime:  15 * time.Minute,
    }

    // 2. 初始化数据库
    db, err := database.New(cfg)
    if err != nil {
        log.Fatal("failed to connect to database:", err)
    }
    defer db.Close()

    // 3. 验证连接
    if err := db.Ping(context.Background()); err != nil {
        log.Fatal("database ping failed:", err)
    }
    log.Println("database connected successfully")

    // 4. 创建 HTTP 服务器。应用层依赖 pkg/web,不直接依赖 Gin。
    r := web.New("release")
    r.Use(web.Recovery())

    // 健康检查端点
    r.GET("/health", func(c web.Context) {
        if err := db.Ping(c.RequestContext()); err != nil {
            c.JSON(http.StatusServiceUnavailable, map[string]any{
                "status": "unhealthy",
                "error":  err.Error(),
            })
            return
        }
        c.JSON(http.StatusOK, map[string]any{"status": "healthy"})
    })

    // 业务路由...
    r.GET("/users", func(c web.Context) {
        var users []User
        if err := db.Find(c.RequestContext(), &users, database.Order("id ASC"), database.Limit(100)); err != nil {
            c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, users)
    })

    // 5. 启动服务器
    srv := &http.Server{
        Addr:    ":8080",
        Handler: r,
    }

    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal("server error:", err)
        }
    }()

    // 6. 优雅关闭
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatal("server forced to shutdown:", err)
    }

    log.Println("server exited")
}

type User struct {
    ID   uint   `gorm:"primaryKey"`
    Name string `gorm:"size:100"`
}
```

## 最佳实践

### 生产环境建议

1. **使用环境变量存储敏感信息**

   ```go
   cfg := &database.Config{
       Password: os.Getenv("RIN_APP_DB_POSTGRES_PASSWORD"),
       // 不要硬编码密码!
   }
   ```

2. **启用 SSL/TLS**

   ```go
   cfg.SSLMode = "verify-full" // PostgreSQL
   ```

3. **合理设置连接池**

   ```go
   cfg.MaxOpenConns = 50
   cfg.MaxIdleConns = 25
   cfg.MaxLifetime = 15 * time.Minute
   ```

4. **实现健康检查**

   ```go
   go func() {
       ticker := time.NewTicker(30 * time.Second)
       for range ticker.C {
           if err := db.Ping(context.Background()); err != nil {
               // 发送告警
           }
       }
   }()
   ```

5. **优雅关闭**
   ```go
   defer func() {
       if err := db.Close(); err != nil {
           log.Printf("failed to close database: %v", err)
       }
   }()
   ```

### 避免常见错误

❌ **不要在循环中创建连接**

```go
// 错误示例
for i := 0; i < 100; i++ {
    db, _ := database.New(cfg) // 每次都创建新连接!
    // ...
}
```

✅ **复用同一个连接**

```go
// 正确示例
db, _ := database.New(cfg)
defer db.Close()

ctx := context.Background()
for i := 0; i < 100; i++ {
    if err := db.Create(ctx, &record); err != nil {
        log.Printf("failed to create record: %v", err)
    }
}
```

❌ **不要设置过大的连接池**

```go
// 错误示例
cfg.MaxOpenConns = 1000 // 太大了!
```

✅ **根据实际并发量设置**

```go
// 正确示例
cfg.MaxOpenConns = 50  // 适合大多数应用
cfg.MaxIdleConns = 25
```

## 支持的数据库版本

| 数据库     | 支持版本 | 推荐版本   |
| ---------- | -------- | ---------- |
| PostgreSQL | 9.6+     | 14.x, 15.x |
| MySQL      | 5.7+     | 8.0+       |
| SQLite     | 3.x      | 3.35+      |

## 包内依赖项

- [GORM](https://gorm.io/) - 包内 ORM 实现,不作为 `internal/` 公共依赖暴露
- `gorm.io/driver/postgres` - PostgreSQL 驱动
- `gorm.io/driver/mysql` - MySQL 驱动
- `gorm.io/driver/sqlite` - SQLite 驱动

## 相关资源

- [GORM 官方文档](https://gorm.io/docs/) - 仅供维护 `pkg/database` 内部实现时参考
- [PostgreSQL 文档](https://www.postgresql.org/docs/)
- [MySQL 文档](https://dev.mysql.com/doc/)
- [SQLite 文档](https://www.sqlite.org/docs.html)
- [数据库连接池最佳实践](https://github.com/brettwooldridge/HikariCP/wiki/About-Pool-Sizing)

## 故障排查

### 连接失败

**问题**: `failed to connect to database`

**解决方案**:

1. 检查数据库服务是否运行
2. 验证 Host、Port、User、Password 是否正确
3. 检查网络连接和防火墙设置
4. 确认数据库是否存在

### 连接超时

**问题**: `connection timeout`

**解决方案**:

1. 增加连接池大小: `MaxOpenConns`
2. 检查数据库性能
3. 优化慢查询
4. 考虑使用读写分离

### 连接数过多

**问题**: `too many connections`

**解决方案**:

1. 减少 `MaxOpenConns`
2. 检查是否有连接泄漏
3. 确保调用 `db.Close()`
4. 增加数据库的最大连接数配置

## 许可证

本项目使用 MIT 许可证。
