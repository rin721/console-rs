# API 参考

本目录记录当前服务暴露的 HTTP、远程插件协议和 JSON-RPC 接口。主系统 HTTP 路由契约以 `internal/transport/http/contracts.go` 为单一事实来源，真实路由注册、后台 API catalog、权限同步和 `docs/api/openapi.yaml` 都从同一份 route contract 派生。

## 文件

| 文件 | 用途 |
| --- | --- |
| `http-api.md` | 面向人阅读的中文 HTTP API 说明，包含主要请求示例和权限说明。 |
| `rpc-api.md` | JSON-RPC 独立入口、内置系统方法和插件 RPC transport 说明。 |
| `openapi.yaml` | 主系统 HTTP OpenAPI 3.0.3 契约，覆盖 health、ready、setup、IAM、Plugins 和 System 路由，由 CLI 生成，不手写维护。 |
| `plugin-protocol/openapi.yaml` | 远程插件 HTTP wire contract，供独立插件进程或跨语言客户端接入，独立于主系统 route contract。 |
| `plugin-protocol/json-rpc.md` | 远程插件 JSON-RPC method 与 params/result 契约。 |
| `plugin-protocol/schemas/*.schema.json` | 远程插件协议 JSON Schema。 |

## 事实来源

| 内容 | 代码或契约来源 |
| --- | --- |
| 主系统 HTTP route contract | `internal/transport/http/contracts.go` |
| 主系统 HTTP 路由装配 | `internal/transport/http/router.go` |
| 主系统 OpenAPI 生成器 | `internal/transport/http/openapi.go` |
| 主系统 OpenAPI 生成命令 | `go run ./cmd/aoi api openapi --output docs/api/openapi.yaml` |
| 运行时主系统 OpenAPI | `GET /openapi.yaml` |
| 插件管理请求/响应 | `internal/plugin` |
| 远程插件外部 JSON 契约 | `docs/api/plugin-protocol` |
| 主系统内部插件 wire-model | `pkg/plugin/protocol` |
| 插件 transport | `pkg/plugin/transport/http`、`pkg/plugin/transport/ws`、`pkg/plugin/transport/rpc` |
| IAM 请求/响应 | `internal/modules/iam/handler`、`internal/modules/iam/service`、`internal/modules/iam/model` |
| System 请求/响应 | `internal/modules/system/handler`、`internal/modules/system/service`、`internal/modules/system/model` |
| 初始化向导 DTO | `internal/app/initcenter/dto` |
| 响应信封与错误码 | `types/result`、`types/errors` |
| JSON-RPC | `internal/transport/rpc`、`pkg/rpcserver` |

## 当前接口面

- 公共探针：`GET /health`、`GET /ready`。
- 公开契约：`GET /openapi.yaml`，返回当前主系统 HTTP OpenAPI YAML，不进入 `/api/v1` catalog、权限同步、操作记录或 SPA fallback。
- Setup：`/api/v1/setup/*`，用于统一初始化向导。
- IAM：首次初始化、自助注册、验证码、登录、刷新、找回/重置密码、邀请、组织、用户、角色、权限、API Token、会话、审计。
- Plugins 管理：远程插件列表、详情、心跳健康状态和能力查看。
- Plugins 协议：HTTP/WS 下的 `/plugin-api/v1/negotiate`、`register`、`heartbeat`、`lease`、`unregister`、`capabilities`、`invoke`、`events`、`subscriptions`、`context`、`injected-schema`、`status`、`metadata`、`drain` 和 `ws` envelope；RPC transport 启用后注册对应 `plugin.*` 方法。
- System：公开运行设置、菜单、配置快照、服务状态、流量劫持监控、API 目录、权限同步、操作记录、版本包、媒体库、参数、字典。
- JSON-RPC：`POST /rpc` 和 RPC 端口 `GET /health`；默认关闭。

受保护 HTTP 路由使用：

```http
Authorization: Bearer <accessToken-or-api-token>
```

## 新增主系统 API 流程

1. 在对应 module handler 中定义稳定请求/响应 DTO；handler 继续复用同一类型，避免匿名 struct 与 contract 漂移。
2. 在 `internal/transport/http/contracts.go` 为新 API 新增 route contract，声明 method、Gin 风格 path、access、permission、summary、请求/响应类型和 path/query/multipart 参数。
3. 在 `internal/transport/http/router.go` 的对应 route group 中使用 `routeSpecFor` 注册 handler；需要权限的路由走 `registerProtectedRouteSpecs`，权限只来自 route contract。
4. 运行 `go run ./cmd/aoi api openapi --output docs/api/openapi.yaml` 生成主系统契约。
5. 运行 `go test ./internal/transport/http -count=1 -mod=readonly`，确认生成 YAML 与提交文件完全一致，且所有实际 `/api/v1` 路由都有 contract registry 条目。
6. 如接口语义变化，同步 `http-api.md`；插件协议变化按 `docs/api/plugin-protocol` 独立维护。

## 维护规则

1. 不得手写修改 `docs/api/openapi.yaml`；该文件只能由 `aoi api openapi` 生成。
2. 不得新增按 path/method 推断权限、catalog 或 OpenAPI 的第二套逻辑；真实路由、`system_apis` catalog、权限同步和 OpenAPI 必须共用 route contract。
3. 主系统 path 在代码内保留 Gin 风格 `:orgId`；OpenAPI 输出由生成器转换为 `{orgId}`。
4. JSON 响应按 `types/result.Result[T]` 统一建模；下载接口使用 binary response；上传接口通过显式 multipart 参数元数据描述。
5. 无法可靠反射的动态字段只能声明为 `object/additionalProperties`，不得编写假精确 schema。
6. JSON-RPC 新增系统方法时，同步 `rpc-api.md`。
