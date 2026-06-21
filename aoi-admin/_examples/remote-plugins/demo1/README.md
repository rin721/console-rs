# Demo Remote Plugin

这是一个分布式远程插件最小示例。

它是独立 Go module，不依赖主系统 Go module。

示例不会 import `pkg/plugin/protocol` 或 `internal/...`。

远程插件只依赖主系统公开的 JSON wire contract：

- `docs/api/plugin-protocol/openapi.yaml`
- `docs/api/plugin-protocol/json-rpc.md`
- `docs/api/plugin-protocol/schemas/*.schema.json`

本示例为了保持可读性，在本地 `protocol/` 目录复制了少量 DTO。其他语言或生产插件可以按 OpenAPI / JSON Schema 生成客户端。

先启动主系统，并启用插件宿主与 DB registry：

```powershell
go run ./cmd/aoi server --config=./configs/examples/plugins-remote-rpc.example.yaml
```

再启动示例插件：

```powershell
cd _examples/remote-plugins/demo1
go run . --host http://127.0.0.1:9999/plugin-api/v1 --endpoint http://127.0.0.1:10098
```

示例会：

- 启动本地 callback server；
- 生成或读取 `instance_id`；
- 调用 `/negotiate` 协商协议版本和 transport；
- 注册时提交 `protocol: "aoi-plugin-json"` 与协商得到的 `transport`，不要再把 `protocol` 当成 HTTP/WS/RPC 传输类型；
- 调用 `/register` 注册公开元数据；
- 调用 `/subscriptions` 订阅 `demo.event`；
- 调用 `/status` 上报运行状态；
- 周期调用 `/lease` 续约；
- 退出时调用 `/unregister`；
- 暴露 `/invoke` 处理 `demo.echo`；
- 暴露 `/events` 接收主系统推送事件；
- 暴露 `/drain` 演示下线通知；
- 暴露 `/rpc` 演示远程 JSON-RPC callback。

共享密钥模式：

```powershell
go run . --secret "dev-plugin-secret"
```

RPC transport 示例：

```powershell
go run . --transports rpc --endpoint http://127.0.0.1:10098
```

自定义实例 ID：

```powershell
go run . --instance demo1-a
```
