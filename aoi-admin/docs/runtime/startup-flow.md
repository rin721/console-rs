# 启动流程

主进程从 `cmd/aoi` 开始。命令层只声明入口和参数，真实启动、装配和生命周期由 `internal/app` 负责。

```text
cmd/aoi
  -> pkg/cli app
  -> server/db/iam/build/init/run/service command
  -> app.New
  -> initapp.NewCore/NewInfrastructure/NewModules
  -> initcenter.New
  -> initapp.NewTransport
  -> lifecycleapp.Run
```

## CLI 入口

`cmd/aoi/main.go` 装配 `pkg/cli` 应用并注册命令。`server --config` 显式参数优先级最高；未显式传参时配置路径按 `APP_CONFIG`、`RIN_CONFIG_PATH`、`configs/config.yaml` 解析。

## 应用构建

`internal/app/initapp` 分阶段创建运行态：

1. `Core`：配置、日志、国际化、ID 生成器。
2. `Infrastructure`：数据库、缓存、执行器、存储。
3. 结构准备：按配置执行 goose 迁移。
4. `Modules`：IAM、Plugins、System。
5. `InitCenter`：统一初始化状态、步骤编排、Web setup API 和旧 IAM setup 兼容适配。
6. `Transport`：HTTP router、插件 HTTP/WS 协议端点、可选插件 RPC adapter、WebUI 静态托管、RPC registry。
7. `lifecycleapp`：启动、关闭、reload 和资源释放。

## 模块装配

- IAM：数据库 executor、密码、token、RBAC、ID、TOTP、notifier -> service -> handler。
- Plugins：`internal/plugin` 装配 `pkg/plugin.Host`、DB-backed registry、router、event bus、security、项目注入 provider、管理 service/handler 和 HTTP/WS/RPC 协议 adapter；不注册具体远程插件。
- System：repository、permission store、ID、host metrics、storage -> service -> handler。

基础设施实例由 `internal/app` 创建。模块 service 只接收接口，不管理基础设施生命周期。

## HTTP、插件协议和 RPC

HTTP 服务由 `pkg/httpserver` 管理，路由由 `internal/transport/http` 注册。WebUI 默认从 `/` 托管统一 React SPA，公开官网、`/setup` 和 `/admin` 由前端路由处理；API、插件协议、健康检查和就绪检查优先于 SPA fallback。

插件宿主默认关闭。启用 `plugins.enabled=true` 后，主系统挂载 `/plugin-api/v1/*` HTTP/WS JSON 协议端点，并默认使用 DB registry 共享 `plugin_id + instance_id` 运行状态；当 `rpc.enabled=true` 且 `plugins.rpc_enabled=true` 时额外注册插件 RPC transport 方法。远程插件独立读取自己的配置，启动后主动调用 `negotiate`、`register`、`renew_lease`、`subscribe_event`、`unregister` 等协议接口。主系统不扫描插件目录、不 import 插件源码、不读取插件私有配置。

JSON-RPC 默认关闭。启用 `rpc.enabled=true` 后，服务会额外监听 `rpc.host:rpc.port` 的 `/rpc` 和 `/health`。RPC 当前只注册系统方法，不承载插件协议。

## 关闭流程

`cmd/aoi/run.go` 监听 `SIGINT` 和 `SIGTERM`。关闭顺序：

1. HTTP server；
2. RPC server；
3. storage；
4. executor；
5. cache；
6. database；
7. logger sync。

reload 由 `reloadapp` 协调，只把配置变化交给支持 reload 的子系统，不让业务模块自行监听或关闭基础设施。
