# 远程插件 JSON-RPC 契约

插件 RPC transport 使用 JSON-RPC 2.0。远程插件只依赖本 wire contract，不依赖主系统 Go module。`params` 字段使用 `schemas/*.schema.json` 中对应请求体；`result` 字段放在 JSON-RPC 标准响应的 `result` 中。

## 主系统暴露给远程插件的方法

| Method | Params Schema | 说明 |
| --- | --- | --- |
| `plugin.negotiate` | `schemas/negotiate.schema.json` | 协商协议版本和 transport。 |
| `plugin.register` | `schemas/register.schema.json` | 注册远程插件实例。 |
| `plugin.heartbeat` | `schemas/heartbeat.schema.json` | 心跳保活。 |
| `plugin.renewLease` | `schemas/renew_lease.schema.json` | 租约续期。 |
| `plugin.unregister` | `schemas/unregister.schema.json` | 注销插件实例。 |
| `plugin.healthCheck` | `schemas/health_check.schema.json` | 查询实例健康状态。 |
| `plugin.listCapabilities` | `schemas/list_capabilities.schema.json` | 查询能力声明。 |
| `plugin.invoke` | `schemas/invoke.schema.json` | 调用插件能力。 |
| `plugin.pushEvent` | `schemas/push_event.schema.json` | 推送事件。 |
| `plugin.subscribeEvent` | `schemas/subscribe_event.schema.json` | 订阅事件。 |
| `plugin.injectContext` | `schemas/inject_context.schema.json` | 获取受控注入上下文。 |
| `plugin.getInjectedSchema` | `schemas/get_injected_schema.schema.json` | 查询注入能力 Schema。 |
| `plugin.reportStatus` | `schemas/report_status.schema.json` | 上报运行状态。 |
| `plugin.syncMetadata` | `schemas/sync_metadata.schema.json` | 同步公开元数据。 |
| `plugin.drain` | `schemas/drain.schema.json` | 通知实例进入下线准备。 |

## 远程插件回调方法

当插件注册的 `transport` 为 `rpc` 时，`endpoint` 指向远程插件自己的 JSON-RPC endpoint。主系统会调用以下方法：

| Method | Params Schema | Result |
| --- | --- | --- |
| `plugin.invoke` | `schemas/invoke.schema.json` | `{ "capability": "...", "result": <json> }` |
| `plugin.pushEvent` | `schemas/push_event.schema.json` | `{ "accepted": true, "event": "..." }` |
| `plugin.drain` | `schemas/drain.schema.json` | `{ "plugin": <plugin_snapshot> }` 或空对象 |

## 示例

```json
{
  "jsonrpc": "2.0",
  "id": "req-001",
  "method": "plugin.renewLease",
  "params": {
    "request_id": "req-001",
    "trace_id": "trace-001",
    "plugin_id": "demo1",
    "instance_id": "demo1-a"
  }
}
```

错误响应使用 JSON-RPC 标准 `error` 字段；主系统内部错误码仍会映射为协议错误码，例如 `invalid_plugin`、`plugin_not_found`、`unauthorized`。
