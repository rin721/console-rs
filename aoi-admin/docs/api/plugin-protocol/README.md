# 远程插件协议契约

本目录是远程插件唯一稳定依赖面。插件可以运行在任意语言、任意服务器或容器中，只要按这里的 JSON wire contract 接入主系统。

- `openapi.yaml`：HTTP transport 契约。
- `json-rpc.md`：JSON-RPC method 到 params/result 的映射。
- `schemas/*.schema.json`：各协议操作的 JSON Schema。

`pkg/plugin/protocol` 是主系统内部 Go wire-model 实现，不是远程插件 SDK，也不是外部分布式服务应 import 的包。若未来需要 Go SDK，应从本目录契约生成或单独维护独立 module。

插件公开元数据中 `protocol` 表示远程插件遵守的 JSON 协议族，当前推荐值为 `aoi-plugin-json`；`transport` 才表示本次实例使用的传输适配器，例如 `http`、`websocket` 或 `rpc`。主系统会兼容旧版只填 `protocol: "http"` 的示例，但新插件应显式提交两个字段。
