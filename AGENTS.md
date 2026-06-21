# Aoi[葵] / console-rs Agent Rules

本文件只描述 `github.com/rin721/console-rs` 中 Aoi[葵] Rust 项目的长期规则。旧 `aoi-admin/AGENTS.md`、旧 `.agents/skills` 和旧 Go 目录结构只作为审计资料，不得复制为新项目规则。

## 产品定位

- Aoi[葵] 是当前项目的行动代号和产品叙事代号，不是内部工程命名前缀。
- console-rs 是 Rust 共享产品底座与管理控制台，承载账号、组织、权限、配置、审计、媒体、版本、初始化、系统观测、API 契约和前端基础能力。
- 项目方向是“共享平台 + 多产品线”。平台能力统一承载，未来产品线复用基础设施并扩展自身业务域。
- 不使用 `Aoi Admin`、`go-scaffold`、`go-admin`、`github.com/rei0721/go-scaffold` 等旧身份作为新项目身份。

## 架构规则

- 根目录是 Rust Cargo workspace；`aoi-admin/` 只保留为参考目录，不在其中继续开发。
- 内部 crate、模块、目录、包名、feature 和类型前缀使用中性工程命名，例如 `core/app`、`core/types`、`core/config`、`domain`、`application`、`interfaces`、`infrastructure`；禁止使用 `aoi-*`、`Aoi*` 作为内部工程命名。
- 当前核心 crate 放在 `crates/core/app`、`crates/core/config`、`crates/core/types`。后续拆分 IAM/System/HTTP/Infra 时也必须保持分层边界，不得退化成无边界 `common` 或 `utils`。
- 后端采用模块化单体优先，分层为 handler/controller、service/usecase、domain/model、repository、middleware、config、migration、scheduler、observability。
- Handler 只负责 HTTP 输入输出；业务规则、事务语义和安全判断进入 service/usecase。
- 数据库、缓存、邮件、存储、指标采集、Token/MFA 等基础设施只能通过 service 所需的接口注入。
- 加密、hash、token、TOTP 等通用工具放在 `crates/tools/crypto`，工具库只返回工具错误；service/usecase 负责转换成应用错误和安全语义。
- 跨模块共享的 `AppResult`、`AppError`、`AppState`、`RequestContext`、运行状态和配置快照归属于 app 生命周期层；当前 `AppResult`/`AppError` 位于 `crates/core/app/src/error.rs`，`crates/core/types` 只作为未来生命周期契约占位，不得承载业务模型、HTTP DTO、数据库 DTO、前端类型或仓储私有结构。
- Route registry 或等价机制必须同时驱动路由注册、权限元数据、API catalog 和 OpenAPI。
- 不引入插件系统、插件市场、插件 HTTP/WS/RPC 协议、插件迁移表、插件权限或插件示例。

## 安全与业务边界

- 明确区分 platform、tenant、product scope；平台权限不能隐式等同于租户权限。
- 会话 Cookie、CSRF、client type、product code 必须来自配置和请求上下文，不得硬编码旧项目名称。
- 注册、邀请、邮箱验证、密码重置等会产生 pending 数据的流程必须有事务回滚或补偿。
- API Token、refresh token、MFA secret、会话 token 只允许安全存储哈希或密文；日志、URL、localStorage、测试快照不得泄露敏感值。
- 邀请、密码重置、邮箱验证的一次性通知令牌只能通过加密投递 secret vault 交给 worker；不得写入 outbox payload、HTTP 响应、日志或 URL。
- System 指标只能展示后端真实采集数据；前端不能 mock 不存在的生产能力。

## 中文优先

- 注释以中文为主，重点解释业务边界、数据流、安全原因和复杂 Rust 类型设计；标识符保持英文。
- i18n 默认 `zh-CN`，前端至少保持 `zh-CN` 与 `en` 文案 parity。
- README、docs、运行手册、测试矩阵、AI 协作说明以中文为主。

## 验证要求

- Rust 变更默认运行 `cargo fmt --all --check`、`cargo check --workspace`、`cargo clippy --workspace --all-targets -- -D warnings`、`cargo test --workspace`、`cargo build --workspace` 和 `git diff --check`。
- 运行配置、WebUI API client、public settings 或请求上下文变更额外运行 `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/runtime-config-boundary-scan.ps1`，确保产品码、client type、Header/CSRF 名称来自配置和 public settings，前端 API 路径保持集中。
- 交付物、迁移矩阵、项目 skills、OpenAPI/route registry、WebUI i18n、Go 参考目录边界或目标证据文档变更额外运行 `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/goal-deliverable-audit.ps1`。
- `aoi-admin/` 来源索引、迁移矩阵来源范围或旧参考目录边界变更额外运行 `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/aoi-admin-source-audit.ps1`，确保旧代码、文档、路由契约、迁移、配置、i18n、React、旧规则和插件删除证据仍被覆盖。
- 阶段验收报告或最终报告结构变更额外运行 `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/stage-acceptance-report-audit.ps1`，确保必备章节、验证命令、残留风险和未完成不得 100% 的边界仍可机器检查。
- 准备声明 `/goal` 完成前运行 `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/goal-completion-audit.ps1 -TargetAcceptanceReport "<report-json>" -RequireReady`，确保目标验收清单已回填、最终目标环境 JSON 已通过非本地 HTTPS/full/passed 校验，且阶段报告不再声明未完成。
- 目标验收报告、报告校验器或发布证据规则变更额外运行 `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/target-acceptance-report-validator-smoke.ps1`。
- 数据库运行时相关变更额外运行 `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-runtime-smoke.ps1`，证明临时 SQLite runtime 下的 plan、ping、insert-id-probe、migrate、setup-repository-probe、migration-history、schema-check、database-preflight 和二次 migrate 幂等报告都可用。
- 部署路径相关变更额外运行 `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-deploy-preflight.ps1 -ApplyMigrations`，并使用目标环境或临时环境变量指向待部署数据库；该门禁必须覆盖 plan、ping、insert-id-probe、migrate、history、schema-check 和 preflight，且所有 driver 都必须得到 `serve_ready=true`。
- PostgreSQL/MySQL 方言或迁移执行器变更还必须通过 `scripts/database-external-smoke.ps1` 在真实数据库服务上验证；本地没有外部服务时，以 CI 的 `external-database-smoke` job 为准并在报告中说明。
- 外部数据库 repository 改造必须运行 `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/repository-dialect-audit.ps1`；PostgreSQL/MySQL 还必须通过 `database-external-smoke.ps1` 或 CI 服务容器证明 SetupRepository、IamRepository、NotificationRepository 与 SystemRepository 在真实数据库上可用。
- 前端变更必须额外运行 typecheck、lint/i18n、build 和 Playwright 检查。
- 无法运行的命令必须说明原因、影响范围和残余风险。
