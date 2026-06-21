# Plugins 模块

Plugins 模块是分布式远程插件宿主。插件不是主系统源码的一部分；主系统不扫描插件目录、不 import 插件实现、不读取插件私有配置。远程插件作为独立进程、服务、容器或节点启动，通过公开 JSON wire contract 主动注册到主系统。

远程插件的稳定接入契约是 `docs/api/plugin-protocol` 中的 OpenAPI、JSON Schema 和 JSON-RPC 文档。`pkg/plugin/protocol` 只是主系统内部 Go wire-model 实现，不是远程插件 SDK。

## 分层边界

| 层级 | 职责 |
| --- | --- |
| `pkg/plugin` | 通用 Host、实例入口、生命周期、注入、调用和事件编排。 |
| `pkg/plugin/protocol` | 主系统内部 Go 协议模型实现，不作为外部插件依赖。 |
| `pkg/plugin/registry` | 插件实例、租约、能力索引、事件订阅和状态 watcher 抽象；当前提供 memory、database/sql 与 polling watcher 实现。 |
| `pkg/plugin/router` | 按 `plugin_id`、`instance_id` 或 capability 选择健康实例，并调用 transport-neutral invoker。 |
| `pkg/plugin/event` | 插件事件发布与订阅抽象；当前使用 direct bus + registry subscription。 |
| `pkg/plugin/security` | 插件认证与权限裁决抽象；当前支持 none、shared_secret，signature 明确 unsupported。 |
| `pkg/plugin/observability` | 插件注册、调用、事件、注入和状态操作的审计/观测事件抽象；Host 通过 recorder 输出，不绑定具体后端。 |
| `pkg/plugin/transport/http` | HTTP transport adapter 和远程 HTTP callback invoker。 |
| `pkg/plugin/transport/ws` | WebSocket envelope、实例级 session registry 和 host-to-plugin 调用。 |
| `pkg/plugin/transport/rpc` | JSON-RPC adapter 与远程 RPC invoker；不依赖 `pkg/rpcserver`。 |
| `internal/plugin` | 项目侧插件封装，负责配置转换、项目能力注入、auth 适配和管理 API。 |
| `internal/app` | 组合根，初始化 DB registry 并装配插件宿主，不注册具体插件。 |

## 协议入口

注册请求进入 `CreatePluginApp` / `NewPluginInstance`，由统一入口规范化：

- `plugin_id + instance_id`
- `protocol`、`transport`、`endpoint`、`schema_version`
- `capabilities`、`permissions`、`hooks`、`metadata`
- `owner_host`、`lease_ttl_seconds`、`lease_expires_at`
- `registered_at`、`last_heartbeat_at`、`created_at`、`updated_at`

注册成功后实例写入 registry。运行时状态以 `(plugin_id, instance_id)` 为主键，不再以单进程内存 map 作为权威数据源。

`protocol` 表示远程插件遵守的 JSON 协议族，当前推荐值为 `aoi-plugin-json`；`transport` 表示该实例使用的传输适配器，例如 `http`、`websocket` 或 `rpc`。旧版只填 `protocol: "http"` 的注册请求会被 Host 兼容规范化，但新插件应显式提交两个字段。

## HTTP 协议端点

默认 base path 为 `/plugin-api/v1`，只在 `plugins.enabled=true` 时挂载：

| 路径 | 操作 |
| --- | --- |
| `/negotiate` | `negotiate_protocol` |
| `/register` | `register` |
| `/heartbeat` | `heartbeat` |
| `/lease` | `renew_lease` |
| `/unregister` | `unregister` |
| `/health-check` | `health_check` |
| `/capabilities` | `list_capabilities` |
| `/invoke` | `invoke` |
| `/events` | `push_event` |
| `/subscriptions` | `subscribe_event` |
| `/context` | `inject_context` |
| `/injected-schema` | `get_injected_schema` |
| `/status` | `report_status` |
| `/metadata` | `sync_metadata` |
| `/drain` | `drain` |
| `/ws` | WebSocket envelope |

RPC transport 只有在 `rpc.enabled=true`、`plugins.enabled=true`、`plugins.rpc_enabled=true` 时注册 `plugin.*` 方法。RPC wire contract 见 `docs/api/plugin-protocol/json-rpc.md`。

`unregister` 在 Host API 层按幂等语义处理。远程插件正常退出、网络重试或进程重启后重复注销同一个 `(plugin_id, instance_id)`，Host 会返回 offline 状态，不把已不存在的实例当作协议失败。

## 配置边界

主系统配置只包含宿主级和分布式基础设施级配置：

```yaml
plugins:
  enabled: false
  base_path: /plugin-api/v1
  default_protocol_version: v1
  allowed_transports: [http, websocket]
  node_id: ""
  node_address: ""
  registry_backend: db
  request_timeout_seconds: 10
  heartbeat_timeout_seconds: 30
  lease_ttl_seconds: 30
  lease_scan_interval_seconds: 15
  retry_count: 0
  router_strategy: round_robin
  allowed_permissions: []
  registration_auth_mode: none
  shared_secret_env: ""
  http_enabled: true
  ws_enabled: true
  rpc_enabled: false
  injection_enabled: true
```

插件私有配置只由远程插件进程读取。注册时提交给主系统的内容只能是公开元数据。

## 状态同步

`pkg/plugin/registry.Watcher` 是插件状态同步入口。Memory registry 在注册、续租、注销、状态上报、元数据同步、租约过期和事件订阅时即时发送 `registry.Change`；SQL registry 通过 `PollWatcher` 对共享数据库做周期观测，供多 Host 感知同一批插件状态。`StatusSynchronizer` 是轻量消费器，调用方可以把 watcher change 转换为节点内缓存刷新、审计或外部事件发布。

## 审计与观测

`pkg/plugin/observability.Recorder` 是插件系统的审计/观测出口。Host 在注册、续租、注销、调用、事件推送、事件订阅、注入上下文、查询注入 Schema、状态上报、元数据同步、下线准备和协议协商时记录 `observability.Event`，事件包含 operation、plugin_id、instance_id、capability、event、protocol、transport、request_id、trace_id、idempotency_key、状态、错误和耗时。默认 recorder 是 no-op，生产环境可以在组合根接入审计表、日志、指标、链路追踪或消息队列。

## 最小分布式闭环

1. 远程插件独立启动，读取自己的私有配置。
2. 插件按 `docs/api/plugin-protocol` 调用 `negotiate_protocol` 协商协议版本和 transport。
3. 插件调用 `register` 提交 `plugin_id + instance_id`、`protocol`、`transport`、endpoint、capabilities 和 permissions。
4. Host 将实例写入 DB-backed registry，其他主系统节点可通过同一 DB 感知状态。
5. 插件周期调用 `renew_lease` 或 `heartbeat` 续约。
6. Router 调用前从 registry 查询健康实例，按 round-robin 路由，并过滤过期租约。
7. 插件调用 `subscribe_event` 建立事件订阅，Host 通过 direct bus 推送到订阅实例。
8. Host 通过 `observability.Recorder` 输出注册、调用、事件、注入和状态操作事件。
9. watcher 或 `StatusSynchronizer` 消费 registry change，同步节点内状态或审计。
10. 插件退出时调用 `unregister`，异常退出由 lease 过期转为 offline。

## 示例

示例远程插件位于 `_examples/remote-plugins/demo1`，是独立 Go module。它通过本地 DTO 对接公开 JSON 协议契约，不依赖主系统 `pkg/...` 或 `internal/...` 包。
