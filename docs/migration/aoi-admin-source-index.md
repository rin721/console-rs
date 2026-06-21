# aoi-admin 参考来源索引

> 审计时间：2026-06-21
>
> 本索引用于证明 Aoi[葵] / console-rs 重构前已读取并分类旧 `aoi-admin/` 参考资料。旧目录只作为历史参考，不作为 Rust 新系统运行路径，也不作为新项目规则来源。

## 使用原则

| 原则 | 说明 |
| --- | --- |
| 只取产品能力和证据 | 参考旧项目的 Setup、IAM、System、WebUI、i18n、API 契约、配置和测试证据 |
| 不复制旧工程身份 | `github.com/rei0721/go-scaffold`、`Aoi Admin`、`go-scaffold`、`go-admin` 只作为旧身份证据出现 |
| 不迁移旧插件系统 | 旧插件模块、协议、配置、迁移、前端页面、文档和示例只作为删除依据 |
| 不复制旧 Agent 规则和 skills | 旧 `AGENTS.md` 和旧 `.agents/skills` 只提供风险提示，新规则已在根 `AGENTS.md` 和新 `.agents/skills/*` 重写 |
| 不保留 Go runtime | Go 代码继续位于 `aoi-admin/`，新运行路径只走 Rust workspace |

## 已读来源清单

| 分类 | 旧项目来源 | 读取到的事实 | 新项目处理 |
| --- | --- | --- | --- |
| 旧项目身份与规则 | `aoi-admin/AGENTS.md` | 旧规则明确项目是 Aoi Admin / `go-scaffold`，后端运行时是 Go 1.25.7，模块名是 `github.com/rei0721/go-scaffold`，并包含插件、Go 构建、Go i18n、旧验证命令等规则 | 不复制；根 `AGENTS.md` 已改写为 Aoi[葵] / console-rs 的 Rust 项目规则 |
| 旧 Agent skills | `aoi-admin/.agents/skills/{api-integration,auth-permission,deployment-review,design-system,frontend-implementation,marketing-site,saas-dashboard-ux,seo-content,soft-modular-product-ui,testing-qa,web-quality-check}` | 旧技能围绕 Go 后端、旧 WebUI、旧产品身份和旧验证链组织 | 不复用正文；新 skills 改为 `.agents/skills/{rust-platform,api-contract,frontend-quality,lifecycle-types,quality}` |
| 旧 AI 文档入口 | `aoi-admin/docs/ai/README.md` | 旧 AI 文档要求先读旧根 `AGENTS.md`、`docs/README.md`、`project-map.md`，并把旧 `internal/app`、`pkg`、`internal/modules` 作为当前 Go 项目事实 | 只作为历史证据；新 AI 协作入口为 `docs/ai/collaboration.md` |
| 旧 AI 迁移/审计记录 | `aoi-admin/docs/ai/react-frontend-migration-plan.md`、`react-frontend-final-audit.md`、`progressive-project-audit.md`、`admin-template-parity.md`、`server-status-dashboard-refactor-plan.md` 等 | 提供旧 React 迁移、平台定位、后台能力对齐、服务器状态和渐进式审计证据 | 已吸收为迁移矩阵和 WebUI 边界，但不把旧任务书当作当前事实 |
| Go 模块与依赖 | `aoi-admin/go.mod` | 旧模块为 `github.com/rei0721/go-scaffold`，依赖 Gin、GORM、goose、Casbin、Redis、Cobra、Zap、gopsutil、go-i18n 等 | 不迁移 Go runtime；Rust 技术栈改为 tokio + axum/tower + sqlx + tracing + clap + config + utoipa |
| 应用生命周期 | `aoi-admin/internal/app/{adapters,cliapp,initapp,initcenter,lifecycleapp,mainapp,reloadapp,testsupport}` | 旧项目把启动、初始化、重载、测试装配拆到 `internal/app` 子包 | 重写为 Rust `crates/core/app/src/app`、`service/setup.rs`、CLI 和 typed state；不保留 Go 包结构 |
| Route contract | `aoi-admin/internal/transport/http/contracts.go` | 旧 contract 统一声明 ID、method、path、tag、permission、scope、OpenAPI/catalog 开关；包含 Setup、IAM、System 路由和 scope 概念 | 保留“单一事实来源”思想，重写为 Rust `route_registry.rs`；不复制 Go DTO/import/路径构造 |
| Setup 能力 | `aoi-admin/internal/app/initcenter`、`contracts.go` 中 `setup.*` 路由、旧 `web/app/app/features/setup/*` | 旧项目有 setup status/schema/config test/run/log/complete 以及 React wizard | 重写为 Rust setup service、schema/check/run/log/complete API 和新 WebUI；配置检测不泄露 secret |
| IAM 能力 | `aoi-admin/internal/modules/iam`、`contracts.go` 中 `iam.*` 路由、`aoi-admin/internal/migrations/*iam*` | 旧项目覆盖用户、组织、角色、权限、session、API Token、邀请、密码重置、邮箱验证、MFA、审计和 scope 修正 | 重写为 Rust IAM service/repository/schema；强化事务、hash/encryption、Cookie/CSRF 配置边界 |
| System 能力 | `aoi-admin/internal/modules/system`、`contracts.go` 中 `system.*` 路由、`aoi-admin/internal/migrations/*system*` | 旧项目覆盖 API catalog、菜单、配置、字典、参数、操作记录、版本、媒体、服务器状态、流量探针 | 重写为 Rust System service/repository/schema；不 mock 后端未采集指标 |
| 旧数据库迁移 | `aoi-admin/internal/migrations/*.sql` | 旧迁移含 IAM/System 基础表，也含 `20260615000100_create_plugin_registry.sql`、`20260615000200_add_plugin_instance_transport.sql` 等插件表 | IAM/System/Setup 按 Rust schema 重建；插件表删除，不进入新 `migrations/` |
| 旧配置 | `aoi-admin/configs/config.example.yaml`、`configs/examples/*.example.yaml`、`configs/locales/**` | 旧配置含 `rpc`、`plugins`、`brand.productCode: aoi-admin`、`aoi_csrf`、SMTP 示例密码、SQLite/MySQL/Postgres/Storage 示例和后端 locale 资源 | 重写为 `configs/console*.example.yaml` 和 `.env.example`；删除插件/RPC 配置；生产密钥要求显式配置 |
| 后端 i18n | `aoi-admin/configs/locales/{api,system,ui,validation}/{zh-CN,en-US}.yaml` | 旧后端 locale 与旧 React `en`/`zh-CN` 不完全一致 | 新项目优先中文，WebUI 保持 `zh-CN`/`en` parity；后端错误和 CLI 文案以中文为主 |
| React 前端工程 | `aoi-admin/web/app/package.json`、`vite.config.ts`、`react-router.config.ts`、`playwright.config.ts` | 旧 React 使用 React 19、React Router、Vite、Tailwind v4、TanStack Query、Zustand、React Hook Form、Zod、i18next、Playwright 等 | 保留有价值的前端栈和验证链思路；新 `web/app` 只调用 Rust 后端契约 |
| React i18n | `aoi-admin/web/app/app/i18n/*`、`locales/zh-CN.json`、`locales/en.json` | 旧前端文案资源集中，但包含旧产品名和旧 API 假设 | 重建到新 `web/app/src/i18n/*`，默认 `zh-CN`，保持 `en` parity |
| React API client | `aoi-admin/web/app/app/lib/api/{client,endpoints,auth,iam,setup,system,plugins,runtime,types}.ts` | 旧 API client 集中 endpoint，但包含 `plugins.ts` 和旧 cookie/product code 假设 | 重写到新 `web/app/src/lib/api/*`；删除插件 API client；client type、Cookie、CSRF、product code 由 Rust runtime/config 驱动 |
| React 路由与设计 | `aoi-admin/web/app/app/routes/{public,setup,admin}`、`app/components/aoi/*`、`design/rules.md` | 旧前端已覆盖 public/setup/admin、Aoi React 组件分层和设计规则，也把插件页面列入共享平台 console | 参考布局、i18n、API client、验证链；删除插件页面；设计系统仅展示 Rust 后端已有能力或本地预览 |
| React E2E | `aoi-admin/web/app/tests/e2e/smoke.spec.ts` | 旧测试包含 public、login、setup、admin、token、localStorage/sessionStorage、旧 cookie 名称和旧 productCode | 重建为新 Playwright smoke，重点检查 Rust API 契约、Setup/IAM/System 和 token 不落地 |
| 旧插件代码 | `aoi-admin/internal/plugin/*`、`pkg/plugin/*`、`pkg/pluginapi/*`、`internal/app/initapp/plugins_test.go` | 旧项目实现插件宿主、HTTP/WS/RPC transport、registry、injection、security、admin service 和测试 | 删除，不迁移；新架构禁止插件系统、插件市场和远程插件协议 |
| 旧插件配置与迁移 | `aoi-admin/configs/examples/plugins-remote-rpc.example.yaml`、`aoi-admin/internal/migrations/*plugin*` | 旧项目提供 remote plugin host/RPC 配置和插件 registry schema | 删除，不进入新配置和新 schema |
| 旧插件文档与示例 | `aoi-admin/docs/api/plugin-protocol/*`、`docs/modules/plugins.md`、`docs/architecture/distributed-plugin-system.md`、`_examples/remote-plugins/*` | 旧项目维护独立插件协议、schema、文档和示例 | 删除，不迁移到 Aoi[葵] 新运行契约 |

## 迁移类别索引

| 类别 | 来源 | 目标 |
| --- | --- | --- |
| 迁移 | route contract 单一事实来源、Setup/IAM/System 基础能力、React public/setup/admin、i18n parity、API client 集中化、Playwright 验证链 | `docs/migration/aoi-admin-migration-matrix.md` 中“迁移/重写”条目 |
| 重写 | Go app lifecycle、Gin/GORM/goose/Casbin、旧配置、旧 token/cookie/product code 假设、旧前后端 locale 差异 | Rust workspace、axum/sqlx/tracing/config、类型化配置和请求上下文 |
| 删除 | 旧插件系统、旧插件协议、旧插件配置、旧插件迁移、旧插件前端、旧插件文档、旧远程插件示例、旧项目身份 | `docs/migration/aoi-admin-migration-matrix.md` 中“删除”条目 |
| 暂缓 | PostgreSQL/MySQL 目标环境验收证明、bucket 生命周期/权限策略控制面、外部版本部署执行器、更深主机指标、托管外部 MQ/告警策略 | 等真实环境、配置、权限、后端采集器和测试闭环就绪后再实现；Prometheus 文本导出已迁移到当前 Rust 实现并复用真实 `sysinfo` CPU/内存/磁盘/load/network 字段，最终失败通知的脱敏 dead-letter 报表和仅恢复 pending secret 的安全 requeue 已迁移到当前 Rust 实现 |

## 后续维护规则

1. 新增迁移能力时，先在本索引确认旧来源，再更新迁移矩阵。
2. 若旧来源包含插件、Go runtime 或旧身份，只能写入“删除/禁止迁移”证据。
3. 若旧来源只是前端展示，必须先确认 Rust route registry、OpenAPI、权限和持久化已经存在。
4. 若新增长期规则，写入根 `AGENTS.md` 或新项目专属 `.agents/skills`，不得从旧目录复制正文。
