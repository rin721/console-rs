# 分布式远程插件系统

本文记录当前仓库中远程插件系统的架构约束。它描述已经落入代码的事实，也标注仍需后续补齐的生产能力。

## 定位

插件是独立进程、独立服务、独立二进制、独立容器或独立节点，不是主系统源码目录、本地 Go 包或 `internal` 模块。主系统只暴露远程注册协议入口，插件独立读取自己的私有配置，并主动向主系统注册。

主系统禁止：

- 扫描插件源码目录；
- import `_examples/remote-plugins` 或 `plugins/...`；
- 读取、合并或托管插件私有配置；
- 在 `internal/app` 中注册具体插件实现；
- 让业务模块绕过 `internal/plugin` 直接调用 transport。

## 分层

| 层级 | 职责 |
| --- | --- |
| `pkg/plugin` | 通用远程插件 Host、实例入口、生命周期、调用编排和状态 watcher。 |
| `pkg/plugin/protocol` | 主系统内部 Go wire-model；外部插件依赖 `docs/api/plugin-protocol`。 |
| `pkg/plugin/registry` | 插件实例、能力索引、租约、事件订阅、watcher 和状态同步抽象。 |
| `pkg/plugin/router` | 根据 capability 或实例 ID 选择健康实例并调用 transport-neutral invoker。 |
| `pkg/plugin/injection` | 输出 JSON Schema / JSON Payload 的通用注入钩子。 |
| `pkg/plugin/event` | 远程插件事件发布与订阅抽象。 |
| `pkg/plugin/security` | 插件认证、授权、scope 裁决和审计信息模型。 |
| `pkg/plugin/observability` | 插件操作的审计/观测事件抽象，不绑定日志、指标或审计存储实现。 |
| `pkg/plugin/transport/*` | HTTP、WebSocket、RPC transport adapter。 |
| `internal/plugin` | 项目侧能力聚合、auth 适配、注入 provider 和管理 API。 |
| `internal/app` | 组合根，装配 Host、registry、router、security、transport 和生命周期。 |

`pkg/plugin` 不依赖 `internal`。`pkg/plugin` 核心不 import 具体 transport adapter。HTTP、WebSocket、RPC 只能作为 adapter 出现。

## 协议与 Transport

远程插件协议是跨语言 JSON 契约。外部稳定契约位于 `docs/api/plugin-protocol`：

- `openapi.yaml`：HTTP transport；
- `json-rpc.md`：JSON-RPC method 映射；
- `schemas/*.schema.json`：通用请求和响应 schema。

注册元数据中：

- `protocol` 表示插件遵守的协议族，当前推荐 `aoi-plugin-json`；
- `transport` 表示传输适配器，当前支持 `http`、`websocket`、`rpc`；
- `schema_version` 表示协议 schema 版本。

Host 会兼容旧版 `protocol: "http"` 写法并规范化为 `protocol: "aoi-plugin-json"`、`transport: "http"`。新插件必须显式提交 `protocol` 和 `transport`。

## 注册流程

```text
Remote Plugin
  -> negotiate_protocol
  -> register(plugin_id, instance_id, protocol, transport, endpoint, capabilities, permissions)
  -> Host.CreatePluginApp / NewPluginInstance
  -> security.Authorizer(register)
  -> registry.RegisterInstance
  -> registry.Watcher emits registered/change
  -> heartbeat / renew_lease
  -> unregister 或 lease 过期下线
```

插件运行时状态以 `(plugin_id, instance_id)` 为主键。DB-backed registry 是默认启用插件时的共享状态层；memory registry 仅用于开发测试或显式配置。

远程协议入口的 `unregister` 按幂等语义处理。重复注销同一个 `(plugin_id, instance_id)` 应返回 offline，而不是让插件退出重试因为 registry not found 失败。

## 状态同步

`registry.Watcher` 是插件状态同步的统一入口。

Memory registry 在以下变化时即时发送 `registry.Change`：

- 注册；
- 租约续期；
- 注销；
- 状态上报；
- 元数据同步；
- 租约过期；
- 事件订阅。

SQL registry 使用 `PollWatcher` 周期性读取共享数据库并输出同一种 change 流。`StatusSynchronizer` 是轻量消费器，用于把 watcher change 转成节点内缓存刷新、审计记录或外部消息发布。

当前未内置 Redis、etcd、Consul、NATS、Kafka 等推送型 registry 后端；这些后端必须实现相同 `registry.Registry` / `registry.Watcher` 契约。

## 调用流程

业务代码不得直接调用具体远程插件或 transport。调用链为：

```text
business module
  -> internal/plugin service
  -> pkg/plugin.Host.Invoke
  -> pkg/plugin/router
  -> registry.ListByCapability 或 GetInstance
  -> transport invoker
  -> remote plugin endpoint
```

Router 必须过滤离线或租约过期实例，并根据 transport 查找 adapter。写操作应携带 `idempotency_key`，避免重试造成重复执行。

## 安全

插件注册、调用、事件订阅和注入上下文都必须经过 security authorizer。插件注册成功不代表拥有全部系统能力；插件只能访问：

- 注册协议允许的能力；
- scope 明确授权的能力；
- 注入钩子明确暴露的 JSON 能力；
- 当前用户、租户和权限上下文允许的能力。

远程插件不能直接访问主系统数据库、Redis、内部 service、完整配置、应用容器或其他插件私有状态。

## 审计与观测

`observability.Recorder` 是插件系统统一的审计/观测出口。Host 当前会为 register、renew_lease、unregister、invoke、push_event、subscribe_event、inject_context、get_injected_schema、report_status、sync_metadata、drain 和 negotiate_protocol 输出 `observability.Event`。事件只包含跨进程 JSON 协议可理解的标识和元数据，例如 operation、plugin_id、instance_id、capability、event、protocol、transport、request_id、trace_id、idempotency_key、状态、错误和耗时。

该抽象不直接写数据库、不绑定日志库，也不反向依赖 `internal/plugin`；生产接入应由组合根提供 recorder，把事件转成审计表、指标、日志、链路追踪或消息队列。

## 防漂移

运行以下命令检查边界：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\tools\ai\check-architecture.ps1
go test ./pkg/plugin/... ./internal/plugin/... ./internal/app/... -count=1 -mod=readonly
```

`tools/ai/check-architecture.ps1` 会检查正式代码是否 import 插件示例、`pkg/plugin` 是否依赖 `internal` 或具体 transport、业务模块是否绕过 `internal/plugin`、配置加载器是否扫描插件目录、协议 metadata 是否同时保留 `protocol` 与 `transport`，以及插件审计/观测抽象是否仍存在。
