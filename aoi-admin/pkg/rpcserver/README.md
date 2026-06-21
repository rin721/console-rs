# pkg/rpcserver - JSON-RPC 服务封装

`pkg/rpcserver` 提供 JSON-RPC 2.0 单请求入口、方法注册表和独立 HTTP 服务生命周期。应用层通过 `internal/transport/rpc` 注册方法；服务默认关闭，开启后监听独立端口并暴露 `/rpc` 和 `/health`。

## API 分类

- 定位：[CONFIRMED] 公共基础设施 API。
- 稳定边界：`Server`、`Config`、`Registry`、`HandlerFunc`、`Request`、`Response`、`RPCError`。
- 当前风险：[CONFIRMED] 当前 MVP 不支持 batch request 和 notification，请求必须包含 `id`。
- 非目标：[CONFIRMED] 本包不定义业务方法、不做 IAM 认证、不共享主 HTTP 端口路由。

## 使用示例

```go
registry := rpcserver.NewRegistry()
_ = registry.Register("system.ping", func(ctx context.Context, params json.RawMessage) (any, error) {
    return map[string]any{"ok": true}, nil
})

srv, err := rpcserver.New(registry, &rpcserver.Config{
    Enabled: true,
    Host:    "127.0.0.1",
    Port:    10099,
}, log)
if err != nil {
    return err
}

if err := srv.Start(ctx); err != nil {
    return err
}
defer srv.Shutdown(ctx)
```

## HTTP 入口

- `POST /rpc`：JSON-RPC 2.0 单请求入口。
- `GET /health`：返回服务状态和已注册方法数量。

标准错误码使用 JSON-RPC 2.0 约定，例如 `-32700` parse error、`-32601` method not found、`-32602` invalid params。
