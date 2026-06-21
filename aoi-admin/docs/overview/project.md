# 项目概述

Aoi Admin 是一个可运行、可扩展的全栈产品平台底座。仓库的 Go module 仍为 `github.com/rei0721/go-scaffold`，但当前产品定位是“共享主平台 + 多独立产品线”：主平台统一承载 Go 后端服务、公开官网、首次安装初始化、`/admin` 平台后台、账号权限、组织租户、配置管理、审计日志、插件、媒体、版本和基础运营能力；未来产品线在同一底座上扩展自己的公开前台、业务后台、业务模块和领域模型。

## 当前能力

| 能力域 | 当前状态 |
| --- | --- |
| 产品形态 | 共享主平台提供通用能力，未来产品线独立扩展业务体验并复用主平台基础设施 |
| 进程入口 | `cmd/aoi` 声明命令，`internal/app` 负责装配和生命周期 |
| HTTP 服务 | `/health`、`/ready`、`/api/v1/*` 和 `/admin` 静态托管 |
| JSON-RPC | 独立端口，默认关闭，当前注册系统探针类方法 |
| 配置 | YAML、`.env`、环境变量覆盖、校验、诊断、监听和受控持久化 |
| 数据库 | SQLite、MySQL、PostgreSQL；goose 迁移 |
| IAM | 本地账号、组织租户、角色、权限、JWT、API Token、会话、邀请、密码重置、TOTP MFA、审计 |
| System | 菜单、API 目录、字典、参数、操作记录、服务器状态、版本发布包、媒体库 |
| Plugins | 远程插件宿主、主动注册、心跳状态、能力声明和管理视图 |
| WebUI | React 一体化前端，默认由 Go 服务从 `/` 托管，包含公开官网、首次安装向导、`/admin` 共享平台后台和未来产品线入口 |
| CLI/TUI | Cobra 命令路由、Bubble Tea 首页、System Center 受管服务命令 |
| 构建部署 | Dockerfile、Compose 示例、生产配置示例、`deploy.sh` 和 GitHub Actions |

## 当前非目标

- 不提供 SSO/OIDC/SAML 外部身份提供商。
- 不提供短信 MFA、邮件验证码 MFA 或企业消息网关。
- 不内置插件市场、插件安装、插件打包或插件进程编排。
- 不把 `pkg/sqlgen`、`pkg/yaml2go` 暴露为后台代码生成产品。
- 不在前端虚构后端尚未暴露的生产能力；产品线方向可以规划和说明边界，真实功能必须以后端 API、权限、持久化和审计能力为准。
- 不允许产品线重复实现主平台已有的账号、组织、权限、配置、审计、API client、i18n、设计系统、测试和构建流程。
- 不承诺 v1 发布兼容性；生产发布仍需独立的迁移、备份、回滚和审计流程。

## 默认运行时

本地默认从 `configs/config.yaml` 或 `configs/config.example.yaml` 读取配置。服务监听 `127.0.0.1:9999`，使用 SQLite `./data/app.db`，启用 IAM、System 和 React WebUI，关闭 Redis、Plugins 和 JSON-RPC。

本地默认可以自动应用迁移，便于首次打开 `/` 并进入 `/setup` 完成初始化。生产配置示例位于 `deploy/config.production.example.yaml`，默认关闭自动迁移，应通过 `go run ./cmd/aoi db migrate status/up` 显式检查和执行。

媒体库记录依赖数据库；普通上传、断点上传、本地下载和本地对象删除还依赖 Storage。只浏览记录或导入外链时可以不启用对象存储。
