# 项目代号 ｢<ruby>Aoi<rp>（</rp><rt>[深作葵](https://www.anisearch.com/character/43848,aoi-fukasaku)</rt><rp>）</rp></ruby>｣ / 工程文档

[![CI](https://github.com/rin721/aoi-server/actions/workflows/ci.yml/badge.svg)](https://github.com/rin721/aoi-server/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25.7-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](../LICENSE)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/rin721/aoi-admin)

Aoi Admin 是一个可运行、可扩展的全栈产品平台底座。仓库保留 `github.com/rei0721/go-scaffold` 作为 Go module 技术身份，但产品定位已经收敛为“共享主平台 + 多独立产品线”：主平台统一提供 Go 后端服务、公开官网、首次安装初始化、`/admin` 平台后台、账号权限、组织租户、配置管理、审计日志、插件、媒体、版本和基础运营能力；未来产品线在同一底座上扩展自己的公开前台、业务后台、业务模块和领域模型。

<p align="center">
  <img src="../configs/logo.png" alt="Aoi Admin logo" width="180">
</p>

## 标星历史

<a href="https://www.star-history.com/?repos=rin721%2Faoi-admin&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=rin721/aoi-admin&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=rin721/aoi-admin&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=rin721/aoi-admin&type=date&legend=top-left" />
 </picture>
</a>

## 按读者进入

| 你是谁 | 推荐入口 | 读完应获得什么 |
| --- | --- | --- |
| 第一次打开项目 | [新人接手指南](onboarding/getting-started.md) | 本地启动、目录定位、第一轮验证命令 |
| 后端开发者 | [分层架构](architecture/layers.md)、[分布式远程插件系统](architecture/distributed-plugin-system.md)、[新增模块](extension/adding-modules.md) | 如何放代码、如何注入依赖、如何避免架构漂移 |
| 模块维护者 | [IAM](modules/iam.md)、[System](modules/system.md)、[Plugins](modules/plugins.md) | 每个模块的能力、边界、权限和测试入口 |
| API 使用者 | [API 参考](api/README.md)、[HTTP API](api/http-api.md)、[OpenAPI](api/openapi.yaml) | 当前 HTTP 路由、认证要求和接口分组 |
| 运维维护者 | [配置说明](environment/configuration.md)、[部署说明](release/deployment.md)、[维护指南](maintenance/maintenance-guide.md) | 配置来源、迁移、部署、运行态维护 |
| AI 协作者 | [AI Workspace](ai/README.md)、[Agent 项目地图](ai/project-map.md) | 当前事实入口、历史证据和交接格式 |

## 当前能力

| 能力域 | 当前状态 |
| --- | --- |
| 产品底座 | Aoi Admin 以共享主平台承载通用能力，未来产品线复用账号、组织、权限、配置、审计、API client、i18n、设计系统和质量工具 |
| 服务入口 | `cmd/aoi` 提供 `server`、`db`、`iam`、`init`、`run`、`service` 等命令；发布包打包使用 `scripts/package.py` |
| HTTP | `internal/transport/http` 注册路由，`pkg/web` 封装 Gin，`pkg/httpserver` 管理服务生命周期 |
| JSON-RPC | `internal/transport/rpc` 注册方法，`pkg/rpcserver` 提供独立端口，默认关闭 |
| 配置 | `internal/config` 支持 YAML、`.env`、环境变量覆盖、校验、监听和受控持久化 |
| 数据库 | `pkg/database` 支持 SQLite、MySQL、PostgreSQL；迁移由 `pkg/migrator` 封装 goose |
| IAM | 本地账号、组织、角色、权限、JWT、API Token、会话、邀请、重置密码、TOTP MFA、审计 |
| System | 菜单、API 目录、字典、参数、操作记录、服务器状态、版本发布包、媒体库 |
| Plugins | 远程插件宿主、主动注册、心跳状态、能力声明和管理视图 |
| WebUI | `web/app` React 一体化前端，默认由 Go 服务从 `/` 托管，包含公开官网、首次安装向导、`/admin` 共享平台后台和未来产品线入口 |
| 构建部署 | Dockerfile、生产配置示例、Compose 示例、`deploy.sh`、GitHub Actions |

## 快速启动

无参数运行会进入交互式 CLI 首页：

```powershell
go run ./cmd/aoi
```

直接启动本地 HTTP 服务：

```powershell
go run ./cmd/aoi server
curl http://127.0.0.1:9999/health
curl http://127.0.0.1:9999/ready
```

首次进入系统可以使用浏览器打开 `http://127.0.0.1:9999/`；共享平台后台位于 `http://127.0.0.1:9999/admin`。本地默认使用 SQLite `./data/app.db`，启用 IAM、System、React WebUI，并默认自动应用迁移；生产示例默认关闭自动迁移。

## 架构边界

当前依赖方向是：

```text
cmd/aoi
  -> internal/app
      -> internal/config
      -> pkg 基础设施
      -> internal/modules/*
      -> internal/transport
      -> internal/middleware
```

模块内仍保留稳定包名：

```text
model <- service -> handler
model <- service <- repository/infrastructure
```

- `pkg` 提供数据库、缓存、日志、HTTP/RPC server、token、RBAC、TOTP、存储、主机指标等基础设施实现。
- `internal/app` 是装配根，负责创建基础设施、模块实现、生命周期和依赖注入。
- `service` 是业务应用层，定义最小接口并通过构造函数接收依赖，不直接初始化数据库、HTTP client、SMTP、配置或日志框架。
- `repository` 和模块 `infrastructure` 是实现层，可以持有 SQL、ORM、HTTP client、SMTP、secret resolver 等细节。
- `handler`、`middleware`、`transport` 只处理传输层和适配层职责。

## 文档地图

| 分区 | 内容 |
| --- | --- |
| [overview](overview/project.md) | 项目能力、非目标和默认运行时 |
| [structure](structure/directory-map.md) | 目录职责、包边界和文档位置 |
| [architecture](architecture/layers.md) | 分层、分布式远程插件系统、依赖方向、装配顺序和边界测试 |
| [runtime](runtime/startup-flow.md) | 启动、HTTP、配置和错误流 |
| [modules](modules/iam.md) | IAM、System、Plugins 模块说明 |
| [api](api/README.md) | HTTP/OpenAPI/RPC 文档入口 |
| [environment](environment/configuration.md) | YAML、`.env`、环境变量、WebUI、System 配置 API |
| [workflows](workflows/db-cli.md) | DB 和 IAM CLI 工作流 |
| [build](build/docker-and-ci.md) | 本地构建、Docker、CI 和质量门禁 |
| [release](release/deployment.md) | 部署配置、脚本和发布清单 |
| [maintenance](maintenance/maintenance-guide.md) | 日常维护、审查清单和运行态治理 |
| [testing](testing/test-matrix.md) | 测试归属、命令和扩大范围规则 |
| [backlog](backlog/known-gaps.md) | 已知缺口和未实现能力 |
| [ai](ai/README.md) | AI 协作索引、历史证据和交接模板 |

## 常用命令

```powershell
go test ./... -count=1 -mod=readonly
go vet ./...
go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi
go run ./cmd/aoi db migrate status
go run ./cmd/aoi db migrate up
```

React WebUI 静态产物：

```powershell
cd web/app
pnpm install
pnpm lint:i18n
pnpm lint
pnpm typecheck
pnpm build
```

Go 服务默认读取 `web/app/build/client` 并从 `/` 托管统一 SPA。`/api`、`/api/v1`、`/health`、`/ready` 和插件协议路径不会进入 SPA fallback。

## 文档维护规则

- 文档只描述当前行为；未来能力和缺口写入 [已知缺口](backlog/known-gaps.md)。
- 配置变更必须同步 `configs/config.example.yaml`、`.env.example`、`deploy/config.production.example.yaml` 和 [配置说明](environment/configuration.md)。
- HTTP 路由变更必须同步 [HTTP API](api/http-api.md) 和 [OpenAPI](api/openapi.yaml)。
- AI 长任务记录可以保留历史状态，但当前事实必须从 [AI Workspace](ai/README.md) 和 [项目地图](ai/project-map.md) 进入。

## 生产提示


真实部署前需要审查数据库、Redis、存储、日志、CORS、健康/就绪检查、备份和回滚策略。

## IDE

建议使用以下任意平台进行开发：

[![VSCode](https://img.shields.io/badge/-Visual%20Studio%20Code-007ACC?style=flat-square&logo=visual-studio-code&logoColor=white)](https://code.visualstudio.com/)

## 测试用浏览器

[![Google Chrome](https://img.shields.io/badge/-Google%20Chrome-4285F4?style=for-the-badge&logo=google-chrome&logoColor=white)](https://www.google.cn/chrome/index.html)
[![Microsoft Edge](https://img.shields.io/badge/-Microsoft%20Edge-0078D7?style=for-the-badge&logo=microsoft-edge&logoColor=white)](https://www.microsoft.com/edge/download)

## 格式规范

* **缩进：** 2 Spaces (当前项目配置) / TAB (模板建议)
* **行尾：** LF
* **引号：** 双引号
* **文件末尾**加空行

## 许可证

MIT，见 [LICENSE](../LICENSE)。
