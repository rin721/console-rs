# 配置流程

配置是运行时输入，由 `internal/config` 负责加载、校验、覆盖、诊断、监听和受控持久化。业务模块只接收 `internal/app` 注入的配置片段，不直接读取配置文件或环境变量。

## 首次加载

```text
--config / RIN_CONFIG_PATH / default path
  -> load .env
  -> read YAML
  -> replace ${VAR} and ${VAR:default}
  -> unmarshal by mapstructure
  -> apply envname override
  -> validate and normalize defaults
  -> atomic store
```

环境变量覆盖优先使用动态前缀，例如当前 `AppPrefix=Rin` 时为 `RIN_APP_*`；未加前缀的兼容变量仍可作为 fallback。

## env_override

`env_override.disabled_paths` 用于跳过指定 `mapstructure` 点路径的环境变量覆盖。它适合必须保留 YAML 文件值的字段，例如本地调试时固定某个 secret 来源。

## 监听和重载

server 模式会监听配置文件变化。文件变化后，配置管理器会影子加载新配置，完成占位符替换、环境覆盖和校验，再写入当前快照并通知变更处理器。

`reloadapp` 只重载支持 reload 的子系统：

- 日志；
- 数据库；
- 缓存；
- 执行器；
- HTTP 服务；
- 存储。

goose migration、System 默认数据种子属于启动和初始化行为，普通配置重载不会重新执行。

## System 配置 API

`GET /api/v1/system/config` 返回脱敏后的当前配置快照。`PATCH /api/v1/system/config` 支持受控持久化，只写回后端明确支持的标量字段和字符串列表字段；带 `${VAR}` 或 `${VAR:default}` 的占位值不会被接口回写成明文。

该 API 不改变 HTTP API 前缀，也不重建 React 静态产物。修改 WebUI 挂载路径、静态目录或 API base URL 时，需要同时更新 Go `webui.*` 配置并重新执行 `pnpm --dir web/app build` 或重新构建 Docker 镜像。

## 维护清单

1. 更新 `internal/config` 结构、`mapstructure`、`envname`、默认值和校验。
2. 更新 `configs/config.example.yaml`、`.env.example`、`deploy/config.production.example.yaml`。
3. 更新 `docs/environment/configuration.md` 和相关部署文档。
4. 判断字段是否支持 reload 或只在启动时生效。
5. 补充配置示例、环境变量覆盖和诊断测试。
