# JSON-RPC API

RPC 入口随 `server` 进程装配，但监听独立端口。默认关闭；启用后监听 `rpc.host:rpc.port`。

插件 RPC transport 只有在以下条件同时满足时注册：

- `rpc.enabled=true`
- `plugins.enabled=true`
- `plugins.rpc_enabled=true`

HTTP、WebSocket 和 RPC 都只是远程插件 JSON 协议的 transport adapter。远程插件不依赖主系统 Go module；RPC params/result 契约见 `docs/api/plugin-protocol/json-rpc.md` 和 `docs/api/plugin-protocol/schemas/*.schema.json`。

## 启用配置

```yaml
rpc:
  enabled: true
  host: 127.0.0.1
  port: 10099

plugins:
  enabled: true
  allowed_transports:
    - http
    - websocket
    - rpc
  registry_backend: db
  rpc_enabled: true
```

## 内置系统方法

- `system.ping`
- `system.methods`

当前 MVP 不支持 batch 和 notification，请求必须包含 `id`。

## 插件协议方法

启用插件 RPC transport 后会额外注册：

- `plugin.negotiate`
- `plugin.register`
- `plugin.heartbeat`
- `plugin.renewLease`
- `plugin.unregister`
- `plugin.healthCheck`
- `plugin.listCapabilities`
- `plugin.invoke`
- `plugin.pushEvent`
- `plugin.subscribeEvent`
- `plugin.injectContext`
- `plugin.getInjectedSchema`
- `plugin.reportStatus`
- `plugin.syncMetadata`
- `plugin.drain`

这些方法的 `params` 使用公开 JSON Schema 契约，不使用主系统 Go 包作为外部依赖。运行时实例以 `plugin_id + instance_id` 定位，插件私有配置仍由远程插件进程自行读取和管理。
