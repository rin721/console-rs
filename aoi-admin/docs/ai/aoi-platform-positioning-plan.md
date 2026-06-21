# Aoi Admin 平台定位与前端表达统一记录

## 当前结论

Aoi Admin 的当前产品定位是可运行、可扩展、可托管多条业务产品线的全栈产品平台底座。仓库保留 `github.com/rei0721/go-scaffold` 作为 Go module 技术身份，但对外产品表达不再定位为单一后台管理系统或纯 Go 后端脚手架。

长期形态为“共享主平台 + 多独立产品线”：

- 主平台统一承载账号、权限、组织租户、配置管理、审计日志、插件、媒体、版本、系统管理和基础运营能力。
- 产品线可以拥有自己的公开前台、业务后台、业务模块、领域模型和用户体验。
- 产品线必须复用主平台账号、组织、权限、配置、审计、API client、i18n、Aoi React 设计系统、测试和构建流程。

## 已确认代码事实

- `web/app` 已是 React 19 一体化前端，包含公开官网、`/setup/*` 首次安装向导、认证入口和 `/admin` 管理路由。
- Go 静态托管默认从 `/` 托管 `web/app/build/client`，并排除 `/api`、`/api/v1`、`/health`、`/ready`、`/openapi.yaml` 和插件协议路径。
- `/setup/*` 依赖后端 setup schema/status API，不应在前端硬编码未支持驱动或生产字段。
- `/admin` 已接入 IAM、System、探针、媒体、版本、插件查询和设计系统配置等后端真实能力。
- 设计系统主题页当前只有本地预览、草稿、导入导出和禁用的发布/回滚表达；后端尚未提供生产级主题持久化、审计、发布或回滚契约。
- 前端 i18n 使用 `zh-CN` 与 `en`，并通过共享 API client 将前端 `en` 映射为后端 `en-US`。

## 表达边界

- 官网可以展示平台价值、产品理念、能力边界和多产品线方向。
- 真实生产功能必须以 Go 后端 API、配置、权限、持久化和审计能力为准。
- 未来产品线入口可以作为架构方向表达，不得伪造可运行的产品线后台、主题发布、插件市场、安装编排或其他后端未提供能力。
- 历史 `docs/ai` 任务记录可保留旧状态作为证据；当前事实入口应从 `docs/README.md`、`docs/overview/project.md`、根 `AGENTS.md` 和 `web/app/AGENTS.md` 进入。

## 验证清单

- 搜索当前有效官网、locale、docs 和规则文件，确认主定位不再描述为单一后台、纯脚手架或前端迁移临时状态。
- 运行 `pnpm --dir web/app lint:i18n`。
- 运行 `pnpm --dir web/app typecheck`。
- 运行 `pnpm --dir web/app test`。
- 运行 `pnpm --dir web/app build` 并确认 `web/app/build/client/index.html` 存在。
- 对 `/`、`/about`、`/setup`、`/admin`、`/admin/design-system` 做桌面 `1440x900` 和移动端 `390x844` 视觉检查；受认证或初始化状态限制时记录检查边界。
- 运行 `git diff --check`。
