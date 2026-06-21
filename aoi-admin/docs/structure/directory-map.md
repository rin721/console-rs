# 目录地图

本项目按“进程入口、装配根、业务模块、基础能力包、文档与运行态产物”组织。目录职责比目录深度更重要：业务核心不创建基础设施，基础设施包不反向依赖应用。

## 根目录

| 路径 | 职责 |
| --- | --- |
| `cmd/aoi` | 进程入口和 CLI 命令声明。 |
| `internal` | 应用专属代码，不作为外部库复用。 |
| `pkg` | 可复用基础能力和通用库。 |
| `types` | 共享常量、错误和响应辅助。 |
| `configs` | 本地配置示例、默认配置和 locale。 |
| `deploy` | 生产配置、Compose 示例和部署材料。 |
| `docs` | 面向人的工程文档和 AI 协作记录。 |
| `web/app` | React 一体化前端，包含公开官网、首次安装向导、`/admin` 共享平台后台和未来产品线入口。 |
| `_examples/remote-plugins` | 独立远程插件示例，不参与主系统构建。 |

## internal

| 路径 | 职责 |
| --- | --- |
| `internal/app` | 组合根、生命周期、reload、模块初始化和依赖注入。 |
| `internal/app/adapters` | 把 `pkg` 类型适配为 transport 或模块 service 需要的接口。 |
| `internal/config` | 配置结构、显式加载、环境覆盖、校验、诊断和持久化。 |
| `internal/middleware` | HTTP 链路中间件。 |
| `internal/plugin` | 项目侧插件层：装配远程插件 Host、注入 schema、项目能力声明、后台管理 API 和协议 handler。 |
| `internal/transport/http` | API 路由、远程插件 HTTP/WS 协议端点、静态 WebUI 挂载和路由目录同步。 |
| `internal/transport/rpc` | JSON-RPC 系统方法注册。 |
| `internal/modules/iam` | 用户、组织、角色、权限、会话、MFA、API Token 和审计。 |
| `internal/modules/system` | 菜单、API、字典、参数、版本、媒体和服务器状态。 |
| `internal/migrations` | goose SQL 迁移。 |
| `internal/ports` | 共享边缘端口，不作为业务 service 的大接口依赖。 |

## pkg

| 路径 | 能力 |
| --- | --- |
| `pkg/plugin` | 远程插件核心：注册表、生命周期、心跳、能力索引、调用、事件和协议抽象。 |
| `pkg/plugin/protocol` | 主系统内部 Go wire-model，不是远程插件 SDK；外部契约见 `docs/api/plugin-protocol`。 |
| `pkg/plugin/registry` | 分布式插件注册中心抽象，提供 memory、database/sql 和 watcher/status sync 能力。 |
| `pkg/plugin/router` | 插件调用路由、健康过滤、基础重试和多实例选择。 |
| `pkg/plugin/event` | 插件事件发布、订阅和 direct bus 抽象。 |
| `pkg/plugin/security` | 插件认证、权限裁决和 scope 控制抽象。 |
| `pkg/plugin/observability` | 插件操作审计/观测事件和 recorder 抽象。 |
| `pkg/plugin/injection` | 通用注入钩子，输出 JSON Schema / JSON Payload。 |
| `pkg/plugin/transport/http` | HTTP transport adapter。 |
| `pkg/plugin/transport/ws` | WebSocket envelope adapter。 |
| `pkg/plugin/transport/rpc` | JSON-RPC transport adapter。 |
| `pkg/database` | GORM 数据库管理、ping、事务和 reload。 |
| `pkg/cache` | Redis 管理。 |
| `pkg/logger` | Zap 和文件轮转日志。 |
| `pkg/httpserver` | HTTP server 生命周期包装。 |
| `pkg/web` | Gin 路由、静态 SPA 和 CORS 封装。 |
| `pkg/rpcserver` | JSON-RPC 2.0 server 和 registry。 |
| `pkg/cli` | Cobra 命令路由和 Bubble Tea/Lip Gloss 首页。 |
| `pkg/token` | JWT 和 refresh token hash。 |
| `pkg/authorization` | Casbin domain RBAC。 |
| `pkg/migrator` | goose migration runner。 |
| `pkg/storage` | 文件存储和 watcher 辅助。 |
| `pkg/configloader` | YAML、dotenv、环境变量和占位符加载。 |

## 边界

- `pkg` 不 import `internal`。
- `internal/app` 初始化 `pkg` 基础能力，并注入到业务模块。
- `internal/plugin` 是项目能力进入远程插件系统的唯一汇聚点。
- 正式代码不 import `_examples/remote-plugins` 或 `plugins/...`。
- 远程插件示例是独立项目，只依赖公开 JSON 协议契约，不依赖主系统 Go module。
