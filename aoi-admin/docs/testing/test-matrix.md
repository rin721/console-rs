# 测试矩阵

测试应靠近它保护的行为。跨装配、配置、HTTP、数据库、共享类型或文档契约的变更，需要从定向测试扩大到全量验证。

## 标准命令

```powershell
go test ./internal/config -count=1 -mod=readonly
go test ./internal/transport/http -count=1 -mod=readonly
go test ./pkg/plugin/... ./internal/plugin/... ./internal/app/... ./internal/transport/http/... -count=1 -mod=readonly
go run ./cmd/aoi api openapi --output docs/api/openapi.yaml
go test ./... -count=1 -mod=readonly
go vet ./...
go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi
git diff --check
```

## 架构边界扫描

```powershell
rg -n "github\\.com/rei0721/go-scaffold/pkg/" internal/modules internal/middleware internal/transport --glob "*.go" --glob "!**/*_test.go"
rg -n "http\\.Client|smtp\\.|os\\.Getenv|database\\.New|WithExecutor|internal/modules/.*/repository" internal/modules/*/service --glob "*.go"
rg -n "github\\.com/rei0721/go-scaffold/(plugins|_examples)" cmd internal pkg types --glob "*.go"
```

`internal/import_boundary_test.go` 固定生产代码的关键边界：业务 service 不直接依赖 `pkg/*`、同模块 repository 或 `internal/ports`；正式代码不 import 插件示例；`pkg/plugin` 不 import `internal` 或 `pkg/rpcserver`。

## 归属矩阵

| 范围 | 常见测试 |
| --- | --- |
| `cmd/aoi` | 顶层命令注册、`CommandSpec` 转换、DB/IAM/init/run/service 命令行为。 |
| `internal/app` | 应用装配、生命周期、迁移、模块注入、reload、System 默认数据同步。 |
| `internal/config` | 配置加载、环境变量覆盖、校验、运行时快照和持久化保护。 |
| `internal/plugin` | 项目侧插件 module、注入 provider、管理 service/handler、禁用能力声明。 |
| `internal/transport/http` | route contract registry、真实路由注册、权限元数据、health/ready、`/openapi.yaml`、插件协议端点、WebUI 静态托管、API catalog 和 OpenAPI 生成漂移守门。 |
| `internal/transport/rpc` | JSON-RPC 系统方法注册、`/rpc` 和 RPC health；插件 RPC 方法只由 `internal/plugin` 在 feature flag 开启时注册。 |
| `internal/modules/iam` | 登录刷新、组织、用户、角色、权限、会话、MFA、邀请、密码重置、API Token、审计和 SMTP notifier 适配。 |
| `internal/modules/system` | 菜单、API、权限同步、操作记录、配置、版本、媒体、参数、字典和服务状态。 |
| `pkg/plugin` | 分布式远程插件注册、租约、注销、健康、能力、调用、事件、协议协商和注入上下文。 |
| `pkg/plugin/registry` | Memory/SQL 注册中心、租约过期、能力索引、事件订阅、watcher 和状态同步。 |
| `pkg/plugin/router` | 多实例路由、健康过滤、round-robin、重试和事件推送。 |
| `pkg/plugin/security` | shared_secret、signature unsupported 和权限范围裁决。 |
| `pkg/plugin/observability` | Host 操作事件 recorder、成功/失败状态、request_id/trace_id/idempotency_key 和错误记录。 |
| `pkg/plugin/transport/http` | HTTP JSON 编解码、错误映射和协议转发。 |
| `pkg/plugin/transport/ws` | Envelope 调度、双向消息模型和错误映射。 |
| `pkg/database` | 连接管理、ping、事务、reload。 |
| `pkg/httpserver` | 启动、关闭、reload 和异步 serve 行为。 |
| `pkg/rpcserver` | JSON-RPC handler、注册表、启动、关闭和 reload。 |
| `types` | 响应信封、错误码、API path helper 和 trace id 兼容。 |

## 扩大测试范围

以下变更应扩大验证：

- `internal/app` 装配或生命周期变更；
- 配置字段、env 覆盖、配置持久化或 reload 行为变更；
- 插件协议模型、transport adapter、注入 schema 或注册生命周期变更；
- 数据库迁移、事务、repository 错误映射或 SQL schema 行为变更；
- HTTP route contract、路由注册、中间件、权限元数据、API catalog 或 OpenAPI 生成结果变更；
- 共享响应、错误、trace id、token、授权或 MFA 逻辑变更；
- WebUI 可见页面、静态托管或构建路径变更。
