# Aoi[葵] Rust 重构阶段验收报告

> 日期：2026-06-22
>
> 本报告用于记录 `github.com/rin721/console-rs` 当前 Rust 重构阶段的可验证状态。它不是最终完成声明；只要目标环境外部数据库验收、部分 System 外延能力和最终部署级验收仍未完成，就不得把 `/goal` 标记为 100% 完成。

## 已完成内容

### 项目定位与迁移治理

- 根目录已经是新的 Rust Cargo workspace，`aoi-admin/` 只保留为历史参考目录。
- 新项目定位为 Rust 共享产品底座与管理控制台，不使用 `Aoi Admin`、`go-scaffold`、`go-admin` 或 `github.com/rei0721/go-scaffold` 作为新身份。
- 已建立 `docs/migration/aoi-admin-source-index.md` 和 `docs/migration/aoi-admin-migration-matrix.md`，覆盖旧代码、文档、路由契约、迁移、配置、i18n、React 前端、旧 `AGENTS.md`、旧 `.agents/skills` 和旧 `docs/ai` 的迁移、重写、删除、暂缓分类。
- 根目录 `AGENTS.md` 与 `.agents/skills/*` 均为 console-rs/Aoi[葵] Rust 项目规则，不复用旧 Aoi Admin 技能正文。

### Rust 后端主干

- Cargo workspace 包含 `crates/core/app`、`crates/core/config`、`crates/core/types`、`crates/tools/crypto`。
- 后端已使用 tokio、axum/tower、serde、sqlx、tracing、clap、config、utoipa 风格生成、thiserror/anyhow 和 tower-http。
- 当前运行服务已覆盖 `/health`、`/ready`、`/openapi.yaml`、Setup、IAM、System 主要 API。
- `route_registry.rs` 是路由注册、权限元数据、OpenAPI 和 API catalog 同步的事实源；app 装配层把 registry 转换为领域 `ApiCatalogEntry`/`SystemMenuEntry` 后同步到 repository，避免仓储层依赖 HTTP contract。
- `AppResult`/`AppError` 已移入 `crates/core/app/src/error.rs` app 生命周期层；`crates/core/types` 不再依赖 `sqlx` 或 `std::io`，避免全局类型层暴露基础设施错误细节。
- `scripts/architecture-boundary-scan.ps1` 已固定检查 handler/service/repository/infrastructure/types 的依赖边界，并正向验证 service 通过 trait 对象接收 repository、通知、存储、指标和探针能力。

### Setup/IAM/System 能力

- Setup 已实现状态、schema、配置检测、初始化运行、步骤日志和完成状态。
- IAM 已实现首个管理员、登录、refresh cookie 轮换、会话快照、登出、组织、用户、角色、权限、API Token、自助注册、邀请、密码重置、邮箱验证、TOTP MFA、恢复码、审计。
- IAM pending 流程已通过事务回滚测试证明：outbox 中段失败时不会留下 pending 用户、组织、成员、邀请、重置、邮箱验证、outbox 或投递 secret 半截数据；通知投递支持 file/log/SMTP/本地加密 queue driver，并提供不返回 payload/raw token/密文的 `notification-dead-letters` 脱敏失败通知报表和只恢复 pending secret 的 `notification-requeue-failed` 安全重排队命令。
- API Token、refresh token、MFA secret、恢复码和 pending 通知令牌采用 hash 或密文存储；脚本会扫描日志、URL、localStorage/sessionStorage、测试产物和前端源码中的泄漏出口。
- System 已实现菜单、API catalog、权限同步、系统配置、字典、参数、操作记录查询/汇总/导出/留存清理、服务器状态、版本包、媒体库、当前 storage driver 对象浏览/删除、真实 HTTP 流量探针基础闭环和流量探针告警 SSE 事件流。
- System 配置和参数会拒绝 `secret`、`token`、`password`、`private`、`credential` 语义 key 写入 System 表；操作记录只保存 route 模板、method、status、actor 和时间，并支持按 `method`、`path`、`status`、`actor_user_id`、`created_from`、`created_to`、`limit`、`offset` 查询与分页，反向时间范围和带 query/fragment 的 path 过滤会被拒绝；`/api/v1/system/operation-records/summary` 与 `operation-records-summary` CLI 复用同一过滤条件输出真实后端聚合的总量、状态段、method 分布和 top path；`/api/v1/system/operation-records/export.csv` 复用同一过滤条件和 `operation_record:read` 平台权限导出 UTF-8 CSV；`POST /api/v1/system/operation-records/prune`、`operation-records-prune` CLI 和 scheduler 均按 `audit.operation_record_retention_days` 与 `audit.operation_record_prune_batch_size` 清理过期记录。
- System 服务器状态只展示后端真实采集数据，已补 `sysinfo` 全局 CPU、进程 CPU、内存、进程内存、交换空间、磁盘容量/使用量、磁盘数量、网络接口数量、网络累计接收/发送字节、系统 uptime/boot time 和 1/5/15 分钟 load average；`/api/v1/system/metrics/prometheus` 已把同一组真实字段导出为 Prometheus text format。平台会话复用 `server:read` 平台权限，外部监控可使用 `observability.prometheus_scrape_token_hash` 配置的专用 Bearer scrape token 哈希；租户 API Token 不提升为平台指标凭据。目标验收脚本的 `metrics-policy` 同时要求匿名请求被 401/403 拒绝且不泄露指标正文、授权 scrape 返回 200 与真实指标正文且不泄露 raw token/API token/session token/secret，不 mock 未接入指标。

### 配置、迁移与运行态

- 新配置示例已覆盖 `configs/console.example.yaml`、`configs/console.production.example.yaml`、`configs/console.secrets.example.yaml` 和 `.env.example`，并新增 `audit.operation_record_retention_days`、`audit.operation_record_prune_batch_size` 与 `observability.prometheus_scrape_token_hash`；scrape token 只允许保存 64 位 SHA-256 hex 哈希，不允许写入 raw scrape token。
- 配置加载采用主 YAML、secrets YAML、环境变量三层覆盖。
- 生产环境会拒绝 dev、占位符和过短密钥，要求 Cookie/CSRF 安全配置。
- 当前默认数据库为 SQLite，迁移目录不包含插件表。
- PostgreSQL/MySQL 已在配置层识别并校验 URL scheme，且已有方言化 bootstrap schema、`database-plan` 迁移文件大小与 SHA-256 摘要输出、`database-ping` 连接探针、`database-insert-id-probe` 插入后 ID 策略探针、`database-setup-repository-probe`、`database-iam-repository-probe`、`database-notification-repository-probe`、`database-system-repository-probe`、`database-migrate`、`database-migration-history`、带 `repository_runtime` 能力矩阵的 `database-preflight` 聚合预检和扫描门禁；NotificationRepository probe 覆盖 claim/deliver/retry/fail、投递 secret 清理、已清除 secret 的 requeue 拒绝和 pending secret 的安全重排队；外部 pool 与 repository set 已接入 app runtime，未声明驱动会显式失败，不会 fallback 到 SQLite。
- `scripts/database-deploy-preflight.ps1` 已把 plan、ping、insert-id-probe、可选显式迁移、migration-history、schema-check 和 preflight 固化为部署前数据库门禁；SQLite、PostgreSQL 和 MySQL 都必须得到 `serve_ready=true`，否则不能放行 `serve`。
- `scripts/repository-dialect-audit.ps1` 已把当前 `SqliteRepository` 的方言隔离点做成审计基线，并确认 PostgreSQL/MySQL 的 `DatabaseRuntimeSupport` 保持 `ready`。
- `docs/deployment/database-runtime-matrix.md` 已明确数据库运行边界，`docs/deployment/target-environment-acceptance.md` 已提供目标环境验收记录模板，`scripts/target-environment-acceptance.ps1` 已把目标数据库 smoke/preflight 与目标 HTTP/WebUI 探针串成可归档 JSON 报告，`scripts/validate-target-acceptance-report.ps1` 已把最终报告必须 full/passed、非本地 HTTPS、数据库 preflight、HTTP/WebUI 探针、`entrypoint-security`、`csrf-policy` 和 `metrics-policy` 通过固化为机器校验，`metrics-policy` 现在同时校验匿名拒绝与授权 scrape 成功，`scripts/target-acceptance-report-validator-smoke.ps1` 已把报告校验器的正反样例固化为 CI smoke。

### WebUI 起点

- 新 WebUI 位于 `web/app`，由 Rust 服务按 `webui.dist_dir` 托管。
- WebUI 覆盖公开入口、Setup、Login、Account、Admin、IAM、System、版本包、媒体上传、对象存储对象浏览/删除、流量探针等当前 Rust 后端已暴露能力。
- API client 使用 HttpOnly Cookie、运行时 public settings、product/client headers、CSRF 双提交配置，不把 session token、refresh token、API Token、MFA secret 或 pending token 写入 URL、本地存储、日志或测试快照。
- `scripts/webui-capability-boundary-scan.ps1` 会验证 System WebUI 的服务器状态、操作记录汇总、版本包、媒体、对象存储对象和流量探针调用均有 Rust route registry 路由，并禁止旧插件、外部版本部署执行器、bucket 生命周期/权限策略或 mock/fake 生产能力回流。
- i18n 默认 `zh-CN`，`zh-CN` 与 `en` 通过 `lint:i18n` 保持 key parity。

### 自动化门禁

- `scripts/security-sensitive-scan.ps1`：扫描敏感 token/secret 泄漏出口。
- `scripts/runtime-config-boundary-scan.ps1`：扫描 WebUI 运行配置边界，确保产品码、client type、Header/CSRF 名称来自 public settings，API 路径集中在 API client 模块。
- `scripts/webui-capability-boundary-scan.ps1`：扫描 WebUI 能力边界，确保 System UI 只调用 Rust route registry 已暴露的生产能力，并禁止旧插件、外部部署执行器、bucket 策略和 mock/fake 能力回流。
- `scripts/governance-scan.ps1`：扫描内部工程命名、旧身份、旧插件运行契约和插件 schema 回流。
- `scripts/goal-deliverable-audit.ps1`：扫描 `/goal` 交付物存在性和禁入项，确保根 Cargo workspace、分层目录、配置/文档/AGENTS、项目 skills、迁移矩阵四类状态、OpenAPI/route registry、WebUI i18n、多方言迁移、参考目录边界和 Go/runtime 插件禁入规则都有当前文件证据。
- `scripts/goal-completion-audit.ps1`：扫描最终完成判定，默认输出 `ready=false/true` JSON；准备声明 `/goal` 完成时必须传入真实目标环境报告并加 `-RequireReady`，证明目标验收清单已回填、阶段报告不再声明未完成、最终目标 JSON 通过非本地 HTTPS/full/passed 校验。
- `scripts/aoi-admin-source-audit.ps1`：扫描旧 `aoi-admin/` 参考来源是否真实存在，并验证来源索引覆盖旧代码、文档、路由契约、迁移、配置、i18n、React、旧规则、旧 `docs/ai` 和插件删除证据。
- `scripts/stage-acceptance-report-audit.ps1`：扫描阶段验收报告结构，确保“已完成内容、进度百分比、剩余任务、删除的旧设计、验证结果、残留风险和下一步”仍按顺序存在，验证命令和未完成不得 100% 的边界仍可机器检查。
- `scripts/database-schema-dialect-scan.ps1`：扫描 SQLite、PostgreSQL、MySQL schema，确保必需表存在且旧插件运行时表不回流。
- `scripts/database-runtime-smoke.ps1`：使用临时 SQLite runtime 验证 `database-plan`、`database-ping`、`database-insert-id-probe`、`database-migrate`、`database-setup-repository-probe`、`database-iam-repository-probe`、`database-notification-repository-probe`、`database-system-repository-probe`、`database-migration-history`、`database-schema-check`、`database-preflight` 和二次迁移 skipped 报告；NotificationRepository probe 会同时断言 purged secret 不可重排队、pending secret 可安全重排队。
- `scripts/database-deploy-preflight.ps1`：使用目标或临时数据库执行部署前 plan、ping、insert-id-probe、显式迁移、迁移历史、核心表和 `serve_ready` 门禁。
- `scripts/database-external-smoke.ps1`：连接真实 PostgreSQL/MySQL 服务，验证外部 driver 的 `database-plan`、`database-ping`、`database-insert-id-probe`、`database-migrate`、`database-setup-repository-probe`、`database-iam-repository-probe`、`database-notification-repository-probe`、`database-system-repository-probe`、`database-migration-history`、`database-schema-check`、`database-preflight` 和二次迁移 skipped 报告，并比对 plan SHA-256 与 `schema_migrations` checksum；MySQL 会确认同连接 `select last_insert_id()` 策略，SetupRepository probe 会确认 `ExternalSetupRepository` 可完成 run/log 读写，IamRepository probe 会确认 `ExternalIamRepository` 可完成管理员、权限、会话、API Token、pending 通知、邀请、密码重置、邮箱验证、MFA 与审计读写，NotificationRepository probe 会确认 `ExternalNotificationRepository` 可完成 claim/deliver/retry/fail、投递 secret 清理和安全 requeue，SystemRepository probe 会确认 `ExternalSystemRepository` 可完成 catalog/menu/config/dictionary/parameter/operation/version/media/traffic 读写。CI 的 `external-database-smoke` job 还会对同一服务执行 `database-deploy-preflight.ps1 -ApplyMigrations`，验证外部部署门禁得到 `serve_ready=true`。
- `scripts/target-environment-acceptance.ps1`：在目标环境或等价网络边界内串联外部数据库 smoke、部署前数据库门禁、目标入口 HTTPS 策略、HTTP 探针、Cookie/CSRF 生产策略、受保护指标策略、WebUI fallback 和 `/api/*` fallback 边界，并输出脱敏 JSON 验收报告；默认必须同时覆盖数据库和 HTTP/WebUI，非本地入口必须是 `https://`，只有 `-AllowPartial` 才能生成 `result=partial` 的诊断报告，不能作为最终通过证据；`metrics-policy` 读取 `CONSOLE_ACCEPTANCE_METRICS_SCRAPE_TOKEN`，证明匿名指标访问被拒绝、授权 scrape 返回 200 且报告脱敏 raw token。
- `scripts/validate-target-acceptance-report.ps1`：校验目标验收 JSON 归档报告，拒绝 partial、failed、本地 HTTP final、非本地 HTTP、缺数据库 preflight、缺 HTTP/WebUI 探针、缺 `entrypoint-security`、缺 `csrf-policy`、缺 `metrics-policy`、缺 `authenticated_status=200` 或缺 `metrics_secret_leaked=false` 的报告；`-AllowLocalHttp` 只用于本地 smoke 报告结构校验。
- `scripts/target-acceptance-report-validator-smoke.ps1`：合成目标验收 JSON 正反样例，验证报告校验器接受非本地 HTTPS full/passed 报告和本地 smoke 报告，并拒绝本地 HTTP final、partial、failed、缺 `csrf-policy`、缺 `metrics-policy` 或缺授权 scrape 证据的报告；该脚本已接入 GitHub Actions。
- `scripts/architecture-boundary-scan.ps1`：扫描 handler/service/repository/infrastructure/types 分层边界和 service/repository trait 注入点。
- `scripts/repository-dialect-audit.ps1`：扫描 SQLite repository 方言阻断点，并防止外部 driver 在阻断点未清理时被宣称 ready。
- `scripts/trailing-whitespace-scan.ps1`：扫描当前 Rust 项目文本区的行尾空白，弥补新工程导入阶段未跟踪文件不能被 `git diff --check` 覆盖的问题。
- `scripts/deployment-smoke.ps1`：构建 WebUI 和后端，使用临时 SQLite runtime 启动真实 Rust 服务，验证 health、ready、OpenAPI、Setup、public settings、受保护 Prometheus 指标和 WebUI fallback，并生成 `target/deployment-smoke/target-acceptance-local.json` 后用 `validate-target-acceptance-report.ps1 -AllowLocalHttp` 校验本地目标验收报告结构。
- `.github/workflows/ci.yml`：在 GitHub Actions 上运行 Rust、契约、治理、安全、架构、空白、WebUI 和本地部署 smoke 验证链。

## 进度百分比

当前阶段进度评估为 **99.98%**。

评分依据：

- 共享产品底座主干、Rust 服务、核心 API、WebUI 起点、文档、迁移矩阵、规则和项目 skills 已落地。
- 安全、治理、目标交付物、aoi-admin 来源索引审计、阶段报告结构审计、目标验收报告校验器 smoke、运行配置、WebUI 能力边界、数据库方言 schema、架构边界、trait 注入边界、生命周期错误边界、后端、System 敏感配置拒写、操作记录过滤/分页/汇总/CSV 导出/留存清理、System 真实服务器指标与 Prometheus 文本导出、Prometheus 匿名访问拒绝与授权 scrape 策略、storage 对象浏览/删除、流量探针 SSE 事件流、WebUI 与本地部署 smoke 验证链已有可重复命令，并已固化为 CI 工作流。
- 仍有生产级外延和目标环境最终部署验收未完成，因此不能标记 100%；本轮已补齐操作记录审计汇总的 HTTP 契约、CLI、SQLite/PostgreSQL/MySQL SQL 模板、SystemRepository runtime probe、WebUI 总览展示和边界扫描证据，并继续保持 `metrics-policy` 目标验收、通知 dead-letter 安全重排队、PostgreSQL/MySQL 外部 pool 与 Setup/IAM/Notification/System repository set 的 app runtime 接入、所有 driver 必须 `serve_ready=true` 的部署门禁、目标验收报告必须通过独立 JSON 校验器后才能作为最终证据。

## 剩余任务

| 剩余项 | 当前状态 | 完成条件 |
| --- | --- | --- |
| 目标环境外部数据库验收 | 部分证明 | PostgreSQL/MySQL 已有配置识别、URL 校验、方言化 bootstrap schema、迁移文件大小与 SHA-256 摘要 CLI、sqlx 连接探针、插入 ID 策略探针、显式迁移执行器、SetupRepository/IamRepository/NotificationRepository/SystemRepository 外部读写路径、迁移历史反查、`serve_ready=true` 的运行前聚合预检、部署前数据库门禁、repository 方言隔离审计和扫描门禁；`scripts/target-environment-acceptance.ps1` 已把目标外部数据库 smoke、部署 preflight 和目标 HTTP/WebUI 探针串成 JSON artifact，且 `scripts/validate-target-acceptance-report.ps1` 会拒绝非 full/passed、非本地 HTTP、partial 和缺 `entrypoint-security`/`csrf-policy`/`metrics-policy` 授权 scrape 证据的报告；`scripts/target-acceptance-report-validator-smoke.ps1` 已在 CI 中回归校验器正反样例；还需在目标生产环境跑脚本、校验报告、备份/回滚演练和实际流量验收 |
| 外部版本部署执行器 | 暂缓 | 有真实制品分发、发布、回滚执行器和权限/审计闭环 |
| bucket 生命周期/权限策略控制面 | 暂缓 | 有真实 bucket 生命周期、对象存储策略管理、权限/审计闭环和部署边界说明，不由前端 mock |
| 外部指标导出/深度主机指标 | 部分证明 | 已提供受 `server:read` 或专用 scrape token 哈希保护的 `/api/v1/system/metrics/prometheus`，以 Prometheus text format 导出 `server-status` 同一组真实 `sysinfo` CPU/内存/磁盘/load/network 字段；目标验收新增 `metrics-policy`，会证明匿名访问该端点被拒绝且不泄露指标正文，并证明配置化 Bearer scrape token 可返回 200 与真实指标正文且不泄露敏感值；当前 WebUI 只展示 `server-status` 已返回的真实字段；目标环境监控系统配置和更深主机采集器仍需后续验收 |
| 托管外部 MQ/告警策略 | 暂缓 | 接入 RabbitMQ/Kafka/SQS 等真实队列接口、外部队列 redrive 和告警策略；当前已有本地加密 queue driver、脱敏 dead-letter 报表和只恢复 pending secret 的安全 requeue |
| 最终部署级验收 | 部分证明 | 本地部署 smoke 已证明临时 SQLite runtime 下的真实二进制、静态托管和核心探针，并覆盖匿名指标拒绝、专用 scrape token 授权，以及真实本地 `target-environment-acceptance.ps1` JSON 报告生成和 `-AllowLocalHttp` 结构校验；目标环境验收清单已列出外部数据库、HTTP 探针、WebUI 托管、Cookie/CSRF、指标访问策略、IAM、System、通知、存储、反向代理/TLS、备份/回滚和日志审计门禁；目标验收脚本可生成脱敏 JSON 证据，并自动验证非本地入口 HTTPS、CSRF 启用、Secure CSRF cookie、缺 CSRF 的非 GET `/api/*` 403 拒绝、匿名指标访问 401/403 拒绝且不泄露指标正文，以及授权 scrape 200 且不泄露敏感值；报告校验器会复核 `scope=full`、`result=passed`、非本地 HTTPS、数据库 preflight、HTTP/WebUI 探针、`entrypoint-security`、`csrf-policy` 和 `metrics-policy`，校验器 smoke 会持续证明这些规则不被误放宽；仍需在目标环境重跑完整后端、前端、脚本、契约、静态托管和运行手册验证 |

补充：`DatabaseDriver::runtime_support().required_work` 已把 PostgreSQL/MySQL runtime 阻断点细化为外部 pool/repository set 接入 `serve` 启动路径、SQLite runtime 绑定移除或隔离、业务 repository 写路径必须按 `InsertIdStrategy::DialectSpecificPostInsertRead` 与 `InsertIdRead::PostInsertQuery("select last_insert_id()")` 使用 MySQL 同连接 ID 读取策略；数据库报告中的 `repository_runtime` 会逐项暴露 Setup/IAM/Notification/System trait 覆盖状态，其中 SetupRepository 已有 `ExternalSetupRepository` 探针实现，IamRepository 已有 `ExternalIamRepository` 探针实现，NotificationRepository 已有 `ExternalNotificationRepository` 探针实现，SystemRepository 已有 `ExternalSystemRepository` 探针实现；这些条目必须与 `scripts/repository-dialect-audit.ps1` 的审计基线同步维护。

## 删除的旧设计

以下旧设计不得进入新项目运行路径：

- 旧 Plugins 模块、插件 HTTP/WS/RPC 协议、插件配置、插件迁移表、插件前端页面、插件权限、插件文档和插件示例。
- 旧 Go 后端 runtime 路径。
- `github.com/rei0721/go-scaffold`、`Aoi Admin`、`go-scaffold`、`go-admin` 旧项目身份。
- 为兼容旧项目引入的双轨、fallback、隐藏开关或过渡架构。
- 旧 `AGENTS.md` 和旧 `.agents/skills` 正文。

## 验证结果

本阶段应以当前工作树命令结果为准。最近一次完整本地验证链包括：

```powershell
cargo fmt --all --check
cargo check --workspace
cargo clippy --workspace --all-targets -- -D warnings
cargo test --workspace
cargo build --workspace
cargo run -p app -- check-config --config configs/console.example.yaml
$env:CONSOLE_OBSERVABILITY_SCRAPE_TOKEN = "stage-report-prometheus-scrape-token"
cargo run -p app -- observability-token-hash --config configs/console.example.yaml
Remove-Item Env:\CONSOLE_OBSERVABILITY_SCRAPE_TOKEN
cargo run -p app -- database-plan --config configs/console.example.yaml
cargo run -p app -- database-ping --config configs/console.example.yaml
cargo run -p app -- database-insert-id-probe --config configs/console.example.yaml
cargo run -p app -- database-migrate --config configs/console.example.yaml
cargo run -p app -- database-setup-repository-probe --config configs/console.example.yaml
cargo run -p app -- database-iam-repository-probe --config configs/console.example.yaml
cargo run -p app -- database-notification-repository-probe --config configs/console.example.yaml
cargo run -p app -- database-system-repository-probe --config configs/console.example.yaml
cargo run -p app -- database-migration-history --config configs/console.example.yaml
cargo run -p app -- database-schema-check --config configs/console.example.yaml
cargo run -p app -- database-preflight --config configs/console.example.yaml
cargo run -p app -- routes --config configs/console.example.yaml
cargo run -p app -- openapi --config configs/console.example.yaml --output docs/api/openapi.yaml
cargo run -p app -- notification-requeue-failed --config configs/console.example.yaml --limit 20
cargo run -p app -- operation-records-summary --config configs/console.example.yaml --top-limit 5
cargo run -p app -- operation-records-prune --config configs/console.example.yaml
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-deploy-preflight.ps1 -ApplyMigrations
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/security-sensitive-scan.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/runtime-config-boundary-scan.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/webui-capability-boundary-scan.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/governance-scan.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/goal-deliverable-audit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/goal-completion-audit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/aoi-admin-source-audit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/stage-acceptance-report-audit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-schema-dialect-scan.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-runtime-smoke.ps1
# 需要本地已有真实数据库服务；CI external-database-smoke job 会用服务容器运行：
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-external-smoke.ps1 -Driver postgres -Url "<postgres-url>"
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-external-smoke.ps1 -Driver mysql -Url "<mysql-url>"
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-deploy-preflight.ps1 -Driver postgres -Url "<postgres-url>" -ApplyMigrations
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-deploy-preflight.ps1 -Driver mysql -Url "<mysql-url>" -ApplyMigrations
$env:CONSOLE_ACCEPTANCE_METRICS_SCRAPE_TOKEN = "<raw-prometheus-scrape-token>"
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/target-environment-acceptance.ps1 -Driver postgres -Url "<postgres-url>" -BaseUrl "<base-url>" -ApplyMigrations
Remove-Item Env:\CONSOLE_ACCEPTANCE_METRICS_SCRAPE_TOKEN
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/validate-target-acceptance-report.ps1 -ReportPath "<report-json>"
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/target-acceptance-report-validator-smoke.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/architecture-boundary-scan.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/repository-dialect-audit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/trailing-whitespace-scan.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/deployment-smoke.ps1
npm --prefix web/app run typecheck
npm --prefix web/app run lint:i18n
npm --prefix web/app run build
npm --prefix web/app run test:e2e
git diff --check
```

本轮补测结果：已新增操作记录审计汇总闭环，覆盖 `/api/v1/system/operation-records/summary`、`operation-records-summary` CLI、route registry/OpenAPI schema、SQLite/PostgreSQL/MySQL SQL 模板、SystemRepository runtime probe、HTTP smoke 成功与越界拒绝、WebUI Admin 总览展示和 WebUI 能力边界扫描；已补充真实网络接口指标，`ServerStatus` 和 Prometheus 现在从 `sysinfo` 导出网络接口数量与累计接收/发送字节，WebUI 只展示这些后端返回字段；已新增 `scripts/goal-completion-audit.ps1`，把“能否声明 `/goal` 完成”做成机器可读 JSON，并在 `-RequireReady` 模式下要求目标验收清单、阶段报告和非本地 HTTPS/full/passed 目标 JSON 全部就绪。此前已覆盖 System 配置/参数敏感 key 家族拒写、操作记录过滤与分页、反向时间范围拒绝、带 query/fragment 的 path 过滤拒绝、CSV 导出、配置驱动的留存清理，并重新生成 `docs/api/openapi.yaml`。`deployment-smoke` 会启用本地 CSRF secure cookie、注入临时 Prometheus scrape token，调用 `target-environment-acceptance.ps1` 生成 `target/deployment-smoke/target-acceptance-local.json`，并用 `validate-target-acceptance-report.ps1 -AllowLocalHttp` 校验真实本地服务的目标验收报告结构。已覆盖 `cargo fmt --all --check`、`cargo check --workspace`、`cargo clippy --workspace --all-targets -- -D warnings`、`cargo test --workspace`、`cargo build --workspace`、`check-config`、`operation-records-summary`、`operation-records-prune`、`observability-token-hash`、route registry/OpenAPI query parameter、summary schema、CSV content type 与留存清理契约快照、Prometheus 指标导出 HTTP smoke、Prometheus 专用 scrape token 哈希校验、Prometheus 匿名访问拒绝与授权 Bearer scrape 目标验收策略、安全扫描、运行配置边界扫描、WebUI 能力边界扫描、治理扫描、目标交付物审计、目标完成度审计、aoi-admin 来源索引审计、阶段报告结构审计、目标验收报告校验器 smoke、数据库 schema 方言扫描、架构边界与 trait 注入扫描、repository 方言审计、SQLite runtime smoke、SQLite deploy preflight、本地 target acceptance 报告生成与 `-AllowLocalHttp` 校验、目标验收报告校验器本地允许/严格拒绝/partial 拒绝/failed 拒绝/缺授权 scrape 证据拒绝/合成 HTTPS 通过策略样例、deployment smoke、行尾空白扫描和 `git diff --check`。PostgreSQL/MySQL external smoke 仍需要真实数据库服务；本机复查未发现 Docker/Podman、psql、mysql 可执行文件或已运行的 PostgreSQL/MySQL/MariaDB Windows 服务，且未提供外部服务 URL。

本轮 repository 方言基线更新为：`Pool<Sqlite>=2`、`SqliteConnection=5`、`SqlDialect::Sqlite` 常量=1、生产 SQL literal 中 SQLite bind placeholder `?=0`、`insert_returning_id=0`、`sqlite_row_type=0`、`last_insert_rowid=0`，且所有已登记 scattered SQL 指标均为 0。SQLite upsert、`limit ?` 查询、Setup/IAM/Notification/System 相关 SQL 已集中到 `sql_templates.rs` 且提供 PostgreSQL/MySQL 模板分支；SQLite/PostgreSQL 插入后取 ID 使用显式 `returning id` 模板，MySQL 通过 `InsertIdStrategy::DialectSpecificPostInsertRead` 与 `InsertIdRead::PostInsertQuery("select last_insert_id()")` 固化后续读取策略；MySQL 操作记录留存删除使用 derived table，避免 `delete` 目标表直接出现在子查询里。

`runtime_support` 现在将 SQLite、PostgreSQL 与 MySQL 均报告为 `ready`；`repository_runtime` 进一步把 Setup/IAM/Notification/System 四类 trait 覆盖做成机器可读矩阵，其中 PostgreSQL/MySQL 通过 `ExternalSetupRepository`、`ExternalIamRepository`、`ExternalNotificationRepository` 与 `ExternalSystemRepository` 接入 app runtime，防止 CLI、部署脚本和文档对数据库能力产生漂移。

focused test 覆盖 SQLite runtime 迁移、SQLite ping、SQLite 插入 ID 探针、SQLite 显式迁移、SQLite SetupRepository 探针、SQLite IamRepository 安全关键路径探针、SQLite NotificationRepository 投递状态机、dead-letter 报表与安全 requeue 探针、System 敏感 key 拒写、操作记录过滤/分页/时间范围/汇总/CSV 导出/留存清理、scheduler 留存清理报告、route registry query parameter、summary schema、CSV content type 与留存清理快照、Prometheus 指标 label 转义、HTTP 权限边界、专用 scrape token 哈希授权、迁移历史反查、二次迁移 skipped 报告、SQLite schema-check、SQLite preflight、PostgreSQL/MySQL 方言计划和 no fallback。本地加密 queue notification driver 仍在验证链内：响应、outbox、队列文件不泄露 raw token，密文可由 worker 使用 queue secret 解密。

`scripts/security-sensitive-scan.ps1`、`scripts/runtime-config-boundary-scan.ps1`、`scripts/webui-capability-boundary-scan.ps1`、`scripts/governance-scan.ps1`、`scripts/goal-deliverable-audit.ps1`、`scripts/goal-completion-audit.ps1`、`scripts/aoi-admin-source-audit.ps1`、`scripts/stage-acceptance-report-audit.ps1`、`scripts/target-acceptance-report-validator-smoke.ps1`、`scripts/database-schema-dialect-scan.ps1`、`scripts/database-runtime-smoke.ps1`、`scripts/architecture-boundary-scan.ps1`、`scripts/repository-dialect-audit.ps1` 和 `scripts/trailing-whitespace-scan.ps1` 均通过；其中 architecture scan 同时验证 handler/service/repository/infrastructure/types 负向依赖边界和 service trait 注入点，WebUI capability scan 同时验证 System UI 的 Rust route registry 后端证据和旧/伪能力禁入规则，goal deliverable audit 直接验证 `/goal` 交付物和禁入项都有当前文件证据，goal completion audit 默认输出当前 `ready=false` 的阻断项而不破坏阶段验证，aoi-admin source audit 直接验证旧参考目录关键来源、来源索引和迁移矩阵范围，stage acceptance report audit 直接验证最终报告必备章节和未完成不得 100% 的边界，target acceptance report validator smoke 直接验证最终报告校验器不会接受 partial、failed、本地 HTTP final、缺 CSRF、缺 metrics-policy 或缺授权 scrape 证据的归档报告。由于当前仓库仍处于新工程导入阶段，文件整体尚未进入 git index，`git diff --check` 不能覆盖未跟踪文件；因此 `scripts/trailing-whitespace-scan.ps1` 覆盖当前 Rust 项目文本区，并在 CI 中固定执行。

## 残留风险

- PostgreSQL/MySQL 已接入 app runtime，并具备 schema、迁移计划预检、连接探针、插入 ID 探针、显式迁移执行器、SetupRepository/IamRepository/NotificationRepository/SystemRepository 外部读写路径和 CI 真实服务 smoke 配置；但本机没有可用 Docker/Podman、数据库 CLI 或已运行的外部数据库服务，目标生产环境仍需完成 smoke、部署 preflight、备份/回滚和实际流量验收。
- queue notification driver 是本地持久化 envelope，不等同于托管外部 MQ；当前已有脱敏 dead-letter 报表和只恢复 pending secret 的安全 requeue，但 RabbitMQ/Kafka/SQS 等托管队列的 redrive 与告警策略仍需补齐。
- System 版本包、媒体库、当前 storage driver 对象浏览/删除、流量探针已有基础闭环和 SSE 事件流，Prometheus 指标导出已复用真实 `sysinfo` CPU/内存/磁盘/load/network 字段，且目标验收会证明匿名访问不泄露指标正文、配置化 Bearer scrape token 可授权采集；但外部部署执行器、bucket 生命周期/权限策略控制面、目标环境监控系统配置和更深主机采集器仍不能由前端模拟成生产能力。
- 当前仓库仍处于新工程导入阶段，工作树文件尚未 stage/commit；最终交付前需要明确提交范围。
- 本地部署 smoke 只证明 Windows 本地/CI runner 上的 SQLite 单机闭环和目标验收脚本结构链路，目标验收脚本允许 loopback `http://` 仅用于本地 smoke；最终发布证据仍必须来自非本地 `https://` 入口，并通过 `validate-target-acceptance-report.ps1` 校验 `entrypoint-security`、`csrf-policy` 和同时覆盖匿名拒绝/授权 scrape 的 `metrics-policy` 等策略步骤，不能替代目标生产环境、外部数据库、SMTP/S3、反向代理或 TLS 验收。

## 下一步

1. 在真实 GitHub 分支推送后观察 `.github/workflows/ci.yml` 首次运行结果；远端 CI 只能证明已推送工作树，不能替代本地未提交状态。
2. 若继续推进生产能力，先用目标服务实际 `auth.session_secret` 生成 `observability.prometheus_scrape_token_hash`，将 raw scrape token 只交给监控系统和验收脚本进程，再在目标网络边界运行 `scripts/target-environment-acceptance.ps1`，并用 `scripts/validate-target-acceptance-report.ps1` 校验归档 JSON；目标环境结果回填到 `docs/deployment/target-environment-acceptance.md`，不引入旧兼容路径。
3. 最终完成前，按 `docs/ai/requirement-evidence-audit.md` 逐项复核，只有全部要求都有强证据时才允许标记 goal complete。
