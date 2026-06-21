# 配置说明

配置由 `internal/config` 加载。主系统只加载显式配置来源，不递归扫描项目目录，也不合并插件示例配置。

## 配置路径优先级

`server`、`db` 和 `iam` 命令都支持 `--config`：

```bash
go run ./cmd/aoi server --config=configs/config.yaml
go run ./cmd/aoi db --config=configs/config.yaml --operation=database --print-sql
go run ./cmd/aoi iam bootstrap-admin --config=configs/config.yaml --org-code=acme --username=admin --email=admin@example.com --password-stdin
```

`server` 命令的配置路径优先级：

1. 命令行参数 `--config`
2. 环境变量 `APP_CONFIG`
3. 环境变量 `RIN_CONFIG_PATH`
4. 默认文件 `configs/config.yaml`

## 插件宿主配置

主系统配置只允许包含插件宿主级和分布式基础设施级配置。插件私有配置由远程插件进程自行读取和管理。

```yaml
plugins:
  enabled: false
  base_path: /plugin-api/v1
  default_protocol_version: v1
  allowed_transports:
    - http
    - websocket
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

插件注册时提交的 `plugin_id`、`instance_id`、`endpoint`、`capabilities`、`permissions`、`hooks`、`metadata` 属于公开注册元数据，不属于主系统配置文件。

## 常用环境变量

| 范围 | 变量 |
| --- | --- |
| Config path | `APP_CONFIG`, `RIN_CONFIG_PATH` |
| Server | `RIN_APP_SERVER_HOST`, `RIN_APP_SERVER_PORT`, `RIN_APP_SERVER_MODE`, `RIN_APP_SERVER_READ_TIMEOUT`, `RIN_APP_SERVER_WRITE_TIMEOUT`, `RIN_APP_SERVER_IDLE_TIMEOUT` |
| RPC | `RIN_APP_RPC_ENABLED`, `RIN_APP_RPC_HOST`, `RIN_APP_RPC_PORT`, `RIN_APP_RPC_READ_TIMEOUT`, `RIN_APP_RPC_WRITE_TIMEOUT`, `RIN_APP_RPC_IDLE_TIMEOUT` |
| Plugins | `RIN_APP_PLUGINS_ENABLED`, `RIN_APP_PLUGINS_BASE_PATH`, `RIN_APP_PLUGINS_DEFAULT_PROTOCOL_VERSION`, `RIN_APP_PLUGINS_ALLOWED_TRANSPORTS`, `RIN_APP_PLUGINS_NODE_ID`, `RIN_APP_PLUGINS_NODE_ADDRESS`, `RIN_APP_PLUGINS_REGISTRY_BACKEND`, `RIN_APP_PLUGINS_REQUEST_TIMEOUT_SECONDS`, `RIN_APP_PLUGINS_HEARTBEAT_TIMEOUT_SECONDS`, `RIN_APP_PLUGINS_LEASE_TTL_SECONDS`, `RIN_APP_PLUGINS_LEASE_SCAN_INTERVAL_SECONDS`, `RIN_APP_PLUGINS_RETRY_COUNT`, `RIN_APP_PLUGINS_ROUTER_STRATEGY`, `RIN_APP_PLUGINS_ALLOWED_PERMISSIONS`, `RIN_APP_PLUGINS_REGISTRATION_AUTH_MODE`, `RIN_APP_PLUGINS_SHARED_SECRET_ENV`, `RIN_APP_PLUGINS_HTTP_ENABLED`, `RIN_APP_PLUGINS_WS_ENABLED`, `RIN_APP_PLUGINS_RPC_ENABLED`, `RIN_APP_PLUGINS_INJECTION_ENABLED` |
| Database | `RIN_APP_DB_DRIVER`, `RIN_APP_DB_SQLITE_PATH`, `RIN_APP_DB_MYSQL_HOST`, `RIN_APP_DB_MYSQL_PORT`, `RIN_APP_DB_MYSQL_USERNAME`, `RIN_APP_DB_MYSQL_PASSWORD`, `RIN_APP_DB_MYSQL_DATABASE`, `RIN_APP_DB_MYSQL_CHARSET`, `RIN_APP_DB_POSTGRES_HOST`, `RIN_APP_DB_POSTGRES_PORT`, `RIN_APP_DB_POSTGRES_USERNAME`, `RIN_APP_DB_POSTGRES_PASSWORD`, `RIN_APP_DB_POSTGRES_DATABASE`, `RIN_APP_DB_POSTGRES_SSL_MODE`, `RIN_APP_DB_POOL_MAX_OPEN_CONNS`, `RIN_APP_DB_POOL_MAX_IDLE_CONNS` |
| Cache | `RIN_APP_CACHE_DRIVER`, `RIN_APP_CACHE_LOCAL_MAX_COST`, `RIN_APP_CACHE_LOCAL_NUM_COUNTERS`, `RIN_APP_CACHE_LOCAL_BUFFER_ITEMS`, `RIN_APP_CACHE_LOCAL_DEFAULT_TTL_SECONDS`, `RIN_APP_CACHE_REDIS_ADDR`, `RIN_APP_CACHE_REDIS_USERNAME`, `RIN_APP_CACHE_REDIS_PASSWORD`, `RIN_APP_CACHE_REDIS_DB` |
| WebUI | `RIN_APP_WEBUI_ENABLED`, `RIN_APP_WEBUI_MOUNT_PATH`, `RIN_APP_WEBUI_DIST_DIR`, `RIN_APP_WEBUI_PUBLIC_BASE_URL`, `VITE_PUBLIC_API_BASE_URL` |
| Auth | `RIN_APP_AUTH_ENABLED`, `RIN_APP_AUTH_REGISTRATION_MODE`, `RIN_APP_AUTH_EMAIL_VERIFICATION_TTL_SECONDS`, `RIN_APP_AUTH_ISSUER`, `RIN_APP_AUTH_AUDIENCE`, `RIN_APP_AUTH_SIGNING_KEY`, `RIN_APP_AUTH_ACCESS_TOKEN_TTL_SECONDS`, `RIN_APP_AUTH_REFRESH_TOKEN_TTL_SECONDS`, `RIN_APP_AUTH_REFRESH_TOKEN_PEPPER`, `RIN_APP_AUTH_MFA_ISSUER`, `RIN_APP_AUTH_MFA_SECRET_KEY` |
| Migration | `RIN_APP_MIGRATION_AUTO_APPLY`, `RIN_APP_MIGRATION_DIR` |
| CORS | `RIN_APP_CORS_ENABLED`, `RIN_APP_CORS_ALLOW_ORIGINS`, `RIN_APP_CORS_ALLOW_METHODS`, `RIN_APP_CORS_ALLOW_HEADERS`, `RIN_APP_CORS_EXPOSE_HEADERS`, `RIN_APP_CORS_ALLOW_CREDENTIALS`, `RIN_APP_CORS_MAX_AGE` |

完整字段列表以 `internal/config/*` 和示例配置为准。

## 默认行为

- SQLite 默认路径为 `./data/app.db`。
- Redis 默认关闭。
- JSON-RPC 独立入口默认关闭；插件 RPC transport 需同时启用 `rpc.enabled` 和 `plugins.rpc_enabled`。
- 插件系统默认关闭；启用后默认使用 DB registry。
- `node_id` 为空时，启动期使用 `hostname-pid` 生成当前 Host 节点标识。
- 主系统不扫描插件源码目录，不 import 插件实现，不读取插件私有配置。
- 示例远程插件位于 `_examples/remote-plugins`，作为独立项目存在。

## 新增配置字段

1. 在对应的 `internal/config/*Config` 结构体中新增字段。
2. 添加 `mapstructure` 和 `envname` 标签。
3. 必要时补默认值、校验和 diagnostics。
4. 更新 `configs/config.example.yaml`、生产示例、`.env.example` 和文档。
5. 在 `internal/config` 中新增或调整测试。
