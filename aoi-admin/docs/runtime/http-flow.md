# HTTP 流程

HTTP 路由位于 `internal/transport/http`。路由器在应用启动期间创建，此时配置、基础设施、模块 service 和 handler 已由 `internal/app/initapp` 注入。

`/api/v1` 是当前公开 API 前缀，由 `types/constants` 维护，不是运行时配置项。HTTP server 的监听由 `pkg/httpserver` 承担；`Start` 只负责绑定地址并非阻塞启动 `Serve` goroutine，主进程使用 `HTTPServer.Wait` 等待异步 `Serve` 错误或优雅关闭完成。

## Route Contract

主系统 HTTP API 的单一事实来源是 `internal/transport/http/contracts.go`：

- route contract 声明 method、Gin 风格 path、access、permission、summary、请求/响应 DTO 和参数。
- `router.go` 使用同一份 contract 注册真实路由和权限中间件。
- System API catalog 和权限同步使用同一份 contract，不再按 path/method 推断权限。
- `openapi.go` 使用同一份 contract 生成 `docs/api/openapi.yaml` 和运行时 `GET /openapi.yaml`。
- OpenAPI 输出会把 Gin 风格 `:orgId` 转成 `{orgId}`。

新增或修改主系统 API 后必须运行：

```powershell
go run ./cmd/aoi api openapi --output docs/api/openapi.yaml
go test ./internal/transport/http -count=1 -mod=readonly
```

## 中间件顺序

1. i18n，可用时启用。
2. trace ID。
3. CORS。
4. 请求日志。
5. panic recovery。

受保护业务路由还会进入 IAM Bearer token 认证和 Casbin domain RBAC 权限校验。中间件只处理传输链路关注点，业务校验留在 service。

## 路由组

| 路由 | 来源 | 权限 |
| --- | --- | --- |
| `GET /health` | route contract + transport router | 公开 |
| `GET /ready` | route contract + transport router | 公开，包含数据库 ping |
| `GET /openapi.yaml` | OpenAPI 生成器 | 公开，不进入 API catalog、权限同步、操作记录或 SPA fallback |
| `/api/v1/setup/*` | Setup handler | 公开，用于统一初始化向导 |
| `/api/v1/auth/setup/*` | IAM setup handler | 公开，用于首次管理员初始化 |
| `/api/v1/auth/signup` | IAM auth handler | 公开，受配置控制 |
| `/api/v1/auth/captcha` | IAM auth handler | 公开，受配置控制 |
| `/api/v1/auth/login`、`/refresh`、`/password/*` | IAM auth handler | 公开 |
| `/api/v1/invitations/{token}/accept` | IAM invitation handler | 公开 |
| `/api/v1/auth/logout`、`/switch-org`、`/mfa/*` | IAM account handler | 认证 |
| `/api/v1/me`、`/api/v1/me/orgs` | IAM account handler | 认证 |
| `/api/v1/orgs/*` | IAM organization handler | `org/user/role/permission/api_token/session/audit:*` |
| `/api/v1/plugins/*` | Plugins handler | `plugin:read` |
| `/api/v1/system/public-settings` | System handler | 公开 |
| `/api/v1/system/*` | System handler | 对应 `config/server/permission/operation/version/media/parameter/dictionary:*` |
| `/`、公开页面、`/setup/**`、`/admin/**` | React WebUI 静态产物 | 由前端路由处理；`/api`、`/api/v1`、`/health`、`/ready`、`/openapi.yaml` 和插件协议路径不进入 SPA fallback |

IAM 路由只在 `auth.enabled=true` 且模块装配成功时注册。插件管理路由只在 `plugins.enabled=true` 且 IAM 可用时注册。

## 请求流

```text
HTTP request
  -> global middleware
  -> optional Auth / RequirePermission
  -> handler bind/parse
  -> service validation/business rules
  -> service-local repository/infrastructure interface
  -> concrete repository/infrastructure implementation
  -> handler response helper
  -> JSON response
```

handler 不隐藏事务或业务规则。service 负责业务校验、用例编排和事务意图，repository 或 infrastructure 负责数据库、HTTP client、SMTP、secret resolver、storage 等技术细节。

## API 目录和操作记录

System API catalog 来自 route contract registry，用于权限同步和后台 API 管理页面。目录同步只关心 contract 中标记为 catalog 的 `/api/v1` 业务路径，避免 WebUI fallback、健康检查、就绪检查、`/openapi.yaml` 或插件协议路径进入权限同步。

操作记录中间件挂在受保护的 Plugins 和 System 路由组上，用于记录后台管理操作。配置更新接口会跳过响应体记录中的敏感值。
