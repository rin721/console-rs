# API 契约

本目录保存从 Rust route registry 生成的 API 契约快照。`openapi.yaml` 不是手写文件，必须通过 CLI 从当前代码生成：

```powershell
cargo run -p app -- openapi --config configs/console.example.yaml --output docs/api/openapi.yaml
```

运行时 `/openapi.yaml`、API catalog、权限同步和路由注册都以 `crates/core/app/src/transport/http/route_registry.rs` 为事实来源。OpenAPI 会从同一契约派生 `operationId`、path 参数、`requestBody`、响应 `$ref`、`components.schemas`、敏感字段扩展标记、权限扩展字段、scope 和产品编码。修改 HTTP 路由、权限元数据、请求/响应 schema 名称或产品上下文时，应同步运行上面的生成命令，并执行：

```powershell
cargo test --workspace
cargo run -p app -- routes --config configs/console.example.yaml
```

`crates/core/app/tests/http_smoke.rs` 中的 `route_registry_syncs_runtime_api_catalog_and_permissions` 会在启动真实 SQLite 测试库后，比对 `system_apis` 与 `iam_permissions` 的数据库集合是否完全来自 route registry。`crates/core/app/src/transport/http/route_registry.rs` 的 `docs_openapi_snapshot_matches_route_registry` 会比对 `docs/api/openapi.yaml` 与当前生成结果。新增路由时不要绕过这些测试去手写 catalog、权限种子或 OpenAPI 快照。

旧 `aoi-admin/docs/api/plugin-protocol/*` 属于参考目录中的插件协议资料，不进入 Aoi[葵] / console-rs 的新运行契约。
