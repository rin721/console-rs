# 分层架构

系统采用装配根驱动的洋葱模型。`pkg` 提供基础设施实现，`internal/app` 负责初始化和注入，业务模块的 `service` 层依赖自己定义的最小接口，具体数据库、HTTP、SMTP、TOTP、RBAC、host metrics 等实现留在 app adapter、repository 或模块 infrastructure 中。

## 依赖方向

```text
cmd/aoi
  -> internal/app
      -> internal/config
      -> pkg
      -> internal/modules/*/repository
      -> internal/modules/*/infrastructure
      -> internal/modules/*/service
      -> internal/modules/*/handler
      -> internal/transport
      -> internal/middleware
```

模块内部稳定包名与职责：

| 包 | 架构角色 | 允许依赖 |
| --- | --- | --- |
| `model` | 领域数据结构和持久化模型 | 标准库、共享常量，避免反向依赖其他层 |
| `service` | 应用服务、领域规则、用例编排、本地接口 contract | `model`、必要的跨模块业务类型和自己定义的接口 |
| `repository` | 持久化实现 | `model`、service-local repository contract、`internal/ports` database executor |
| `infrastructure` | 模块专属基础设施实现 | service-local contract、标准库、必要基础设施细节 |
| `handler` | HTTP 输入输出适配 | service contract、`internal/ports` HTTP context、响应辅助 |

## 层职责

| 层 | 职责 |
| --- | --- |
| `cmd/aoi` | 声明命令规格、桥接执行函数和进程入口。 |
| `pkg` | 封装可复用基础能力，不导入 `internal/app` 或业务模块。 |
| `internal/config` | 配置结构、加载、环境覆盖、校验、诊断和持久化转换。 |
| `internal/app` | 创建基础设施、适配 `pkg`、装配模块、注册传输层、管理启动/关闭/重载。 |
| `internal/app/adapters` | 把 `pkg` 实现转换成模块 service 或 transport 需要的接口。 |
| `internal/ports` | 共享边缘端口，主要服务 app adapter、transport、middleware、handler 和 repository infrastructure。 |
| `internal/modules` | 承载业务模块。核心逻辑不直接初始化或管理基础设施。 |
| `internal/transport` | 注册 HTTP/RPC 路由，把请求交给 handler。 |
| `internal/middleware` | 处理 trace、认证、权限、i18n、CORS、recovery、logging 等传输链路关注点。 |

## 当前收敛规则

- `internal/modules`、`internal/middleware`、`internal/transport` 的生产代码不直接导入 `pkg/*`。
- `internal/modules/*/service` 不导入同模块 `repository`，不导入 `internal/app`，不创建数据库连接、HTTP client、SMTP client、logger 或 config loader。
- service 需要数据库、事务、外部请求、通知、授权、token、TOTP、ID、host metrics 等能力时，在本包定义最小接口并通过构造函数注入。
- repository 可以持有 SQL、ORM 条件和数据库 executor，但对 service 暴露 service-local contract，并把底层 not-found/storage-unavailable 等错误映射为 service 错误。
- module infrastructure 可以实现 HTTP proxy、secret resolver、SMTP notifier 等技术细节，但业务核心只看接口。

## 装配顺序

`internal/app/initapp` 按以下顺序构建应用：

1. 核心服务：配置、日志、国际化、ID 生成器。
2. 基础设施：数据库、缓存、执行器、存储。
3. 可选结构应用：goose 迁移。
4. 模块：IAM、Plugins、System。
5. 传输层：`pkg/web` engine、`pkg/rpcserver` registry、HTTP/RPC 注册。
6. 生命周期：启动、停止、重载和资源关闭。

## 边界测试

`internal/import_boundary_test.go` 守护关键规则：

- internal 生产代码除 `internal/app/**`、`internal/config/**` 外不得导入 `pkg/*`。
- module service 生产代码不得导入同模块 repository 或 `internal/ports`。
- module service 生产代码不得出现 `http.Client`、`smtp.`、`os.Getenv`、`database.New`、`WithExecutor` 等基础设施模式。

业务测试如果需要真实 SQLite、迁移、token、RBAC 或 TOTP，优先通过 `internal/app/testsupport` 获取夹具，避免复制生产初始化逻辑。
