# 项目代号 ｢<ruby>Aoi<rp>（</rp><rt>[葵](https://www.anisearch.com/character/43848,aoi-fukasaku)</rt><rp>）</rp></ruby>｣ / 工程文档

[![CI](https://github.com/rin721/console-rs/actions/workflows/ci.yml/badge.svg)](https://github.com/rin721/console-rs/actions/workflows/ci.yml)
[![Rust](https://img.shields.io/badge/Rust-workspace-dea584?logo=rust)](https://www.rust-lang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/rin721/console-rs)

Aoi 是以 Rust 构建的共享产品底座与管理控制台。它不是旧 `aoi-admin`、不是 Go 脚手架、也不是旧项目改名；`aoi-admin/` 在本仓库中只作为迁移参考资料保留。

<p align="center">
  <img src="./qwq.png" alt="Aoi[葵] product mark" width="180">
</p>

Aoi 是以 Rust 构建的共享产品底座与管理控制台。

当前阶段优先交付后端基础闭环与新 WebUI 起点：配置加载、SQLite 迁移、健康探针、Setup 状态、配置检测与日志、IAM 首个管理员与 Cookie 会话、refresh cookie 轮换、CSRF 双提交保护、组织/用户/角色/权限读取、租户用户资料/状态/角色更新、租户角色创建/更新/删除、API Token、配置开关驱动的自助注册、邀请、密码重置、邮箱验证 pending 数据与通知 outbox、本地/SMTP/queue notification drain worker、通知失败重试与最终失败清理、脱敏 dead-letter 报表与安全重排队、TOTP MFA 与一次性恢复码、权限同步、API 目录、系统菜单、系统配置、字典、参数、版本包、媒体库、流量探针、可选后台 scheduler、操作记录查询/汇总/导出/留存清理、真实服务器状态、Prometheus 文本指标导出、OpenAPI 生成、IAM 审计写入，以及只连接 Rust 已暴露 API 的 React 管理控制台和账号流程页。

当前阶段验收状态见 [Aoi[葵] Rust 重构阶段验收报告](docs/ai/stage-acceptance-report.md)。

## 本地运行

```powershell
cargo run -p app -- serve --config configs/console.example.yaml
```

默认监听 `127.0.0.1:8080`，默认数据库为 `data/console.sqlite`。`database.driver` 已类型化识别 `sqlite`、`postgres`、`mysql` 并校验 URL 协议；`cargo run -p app -- check-config` 会输出 `database_runtime` 能力报告，`cargo run -p app -- database-plan` 会输出当前 driver 的 runtime 状态、迁移目录、迁移文件大小和 SHA-256 摘要，`cargo run -p app -- database-ping` 会使用对应 sqlx pool 执行连接探针，`cargo run -p app -- database-insert-id-probe` 会用临时表验证当前方言的插入后 ID 读取策略，`cargo run -p app -- database-migrate` 会显式执行当前 driver 的迁移脚本并输出 applied/skipped 清单，`cargo run -p app -- database-setup-repository-probe` 会在已迁移数据库上验证 SetupRepository 读写路径，`cargo run -p app -- database-iam-repository-probe` 会验证 IamRepository 管理员、权限、会话、API Token、pending 通知、邀请、密码重置、邮箱验证、MFA 与审计读写路径，`cargo run -p app -- database-notification-repository-probe` 会验证 NotificationRepository claim/deliver/retry/fail、投递 secret 清理和失败通知安全重排队路径，`cargo run -p app -- database-system-repository-probe` 会验证 SystemRepository catalog/menu/config/dictionary/parameter/operation/version/media/traffic probe 读写路径，`cargo run -p app -- database-migration-history` 会反查当前数据库已应用迁移记录，`cargo run -p app -- database-preflight` 会聚合迁移计划、连接、迁移历史、核心表和 repository readiness，并输出 `repository_runtime` 能力矩阵与是否可进入 `serve`。当前 `serve` 运行时通过中性 `DatabaseConnection` 装配 SQLite、PostgreSQL 和 MySQL；外部 driver 已接入对应 pool、显式迁移执行器、SetupRepository、IamRepository、NotificationRepository 和 SystemRepository，启用外部库不会静默回退到 SQLite；部署侧边界见 [数据库运行矩阵](docs/deployment/database-runtime-matrix.md)。

新 WebUI 位于 `web/app`。构建后，Rust 服务会按 `webui.dist_dir` 托管静态产物，并对 `/admin/*`、`/setup/*` 等前端路由返回 SPA 入口；`/api/*`、`/health`、`/ready` 和 `/openapi.yaml` 始终保留给后端，不会被前端 fallback 吞掉。

```powershell
npm --prefix web/app install
npm --prefix web/app run build
cargo run -p app -- serve --config configs/console.example.yaml
```

开发时也可单独启动 Vite，并通过 proxy 访问 Rust API：

```powershell
cd web/app
npm install
npm run dev
```

默认监听 `127.0.0.1:3002`。WebUI 首屏是公开平台入口和可操作的 Setup/Login/Admin 控制台，不是营销落地页；它不会显示插件、外部版本部署执行器、bucket 生命周期策略、周期调度或未接入采集器的指标能力。公开入口只读取 health/ready/public-settings/setup 状态；System 配置/字典/参数、版本包发布/回滚生命周期、媒体资产、对象存储对象、流量探针目标和探针告警处置都只调用当前 Rust 已暴露 API；媒体上传通过当前 Rust `MediaStorage` 接口进入配置化 `local` 或 S3 兼容存储。

常用检查：

```powershell
cargo run -p app -- check-config --config configs/console.example.yaml
cargo run -p app -- database-plan --config configs/console.example.yaml
cargo run -p app -- database-ping --config configs/console.example.yaml
cargo run -p app -- database-insert-id-probe --config configs/console.example.yaml
cargo run -p app -- database-migrate --config configs/console.example.yaml
cargo run -p app -- database-setup-repository-probe --config configs/console.example.yaml
cargo run -p app -- database-iam-repository-probe --config configs/console.example.yaml
cargo run -p app -- database-notification-repository-probe --config configs/console.example.yaml
cargo run -p app -- database-system-repository-probe --config configs/console.example.yaml
cargo run -p app -- database-migration-history --config configs/console.example.yaml
cargo run -p app -- database-preflight --config configs/console.example.yaml
cargo run -p app -- routes --config configs/console.example.yaml
cargo run -p app -- openapi --config configs/console.example.yaml --output docs/api/openapi.yaml
cargo run -p app -- notification-drain --config configs/console.example.yaml --limit 20
cargo run -p app -- notification-dead-letters --config configs/console.example.yaml --limit 20
cargo run -p app -- notification-requeue-failed --config configs/console.example.yaml --limit 20
cargo run -p app -- operation-records-summary --config configs/console.example.yaml --top-limit 5
cargo run -p app -- operation-records-prune --config configs/console.example.yaml
cargo run -p app -- scheduler-run-once --config configs/console.example.yaml
curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8080/openapi.yaml
```

生产配置请从 `configs/console.production.example.yaml` 起步，并把真实密钥放到被 gitignore 的 secrets 文件或环境变量中：

```powershell
Copy-Item configs/console.secrets.example.yaml configs/console.secrets.yaml
# 编辑 configs/console.secrets.yaml，填入真实随机值和 SMTP 密码
cargo run -p app -- check-config `
  --config configs/console.production.example.yaml `
  --secrets configs/console.secrets.yaml
```

配置优先级是主 YAML < secrets YAML < `CONSOLE__...` 环境变量。secrets 模板中的 `replace-with-*` 占位符会被生产校验拒绝，不能直接用于部署。

## 当前 API 闭环

- `/health`、`/ready` 提供存活和就绪探针。
- `/openapi.yaml` 从 route registry 生成。
- `/api/v1/setup/*` 提供初始化状态、schema、结构化配置检测、运行列表、步骤日志和完成状态。配置检测会报告数据库、迁移、secrets、Cookie/CSRF、通知、存储和 WebUI 托管状态，但不会返回密钥、SMTP 密码或 token 明文；`/api/v1/setup/complete` 只有在配置检测无 error 且首个管理员已创建后才会写入完成状态。请求携带 `run_id` 时，会在同一事务内更新该初始化运行状态并追加完成日志。
- `/api/v1/auth/setup/initial-admin` 创建首个管理员，使用事务写入组织、用户、角色、成员和审计日志。
- `/api/v1/auth/login`、`/api/v1/auth/refresh` 与 `/api/v1/me/session` 建立、轮换并读取 HttpOnly Cookie 会话，会话权限来自数据库角色绑定。
- Session token 与 refresh token 均只保存 hash；refresh 成功会同时轮换 session cookie 和 refresh cookie，旧 refresh 立即失效，响应体、日志、URL、localStorage 和测试快照不得保存原始值。
- `/api/v1/auth/register` 在 `auth.self_signup_enabled=true` 时开放自助注册；它会在一个事务内创建 pending 用户、租户组织、owner 成员关系、邮箱验证 pending 数据、通知 outbox 和加密投递 secret。默认配置关闭自助注册，注册完成前账号不能登录，响应不返回邮箱验证 raw token。
- `/api/v1/system/public-settings` 返回产品、i18n、上下文 Header 和 CSRF 配置；启用 `auth.csrf.enabled` 时会下发可被浏览器脚本读取的 CSRF cookie。所有非 GET/HEAD/OPTIONS 的 `/api/*` 请求必须同时携带同名 cookie 与配置化 header，生产环境强制 `auth.csrf.enabled=true` 且 `auth.csrf.secure=true`。
- `/api/v1/iam/orgs`、`/api/v1/iam/orgs/{orgId}/users`、`/api/v1/iam/orgs/{orgId}/users/{userId}`、`/api/v1/iam/orgs/{orgId}/roles` 和 `/api/v1/iam/permissions` 提供 IAM 组织、用户、角色、权限管理基础；平台权限与租户权限分别由 `org:read`/`permission:read` 和 `user:read`/`user:write`/`role:read`/`role:write` 控制。租户用户更新只允许改当前组织内用户的显示名、`active`/`disabled` 状态和租户角色集合，并保护组织最后一个 active owner。
- `/api/v1/iam/orgs/{orgId}/roles` 支持租户角色创建，`/api/v1/iam/orgs/{orgId}/roles/{roleId}` 支持租户角色更新和删除；自定义租户角色只能绑定当前产品的 `tenant`/`product` 权限，不能绑定 `permission:read` 等平台级权限。内置角色不可修改或删除，仍被成员或待处理邀请使用的角色不可删除。
- `/api/v1/auth/mfa/factors`、`/api/v1/auth/mfa/setup`、`/api/v1/auth/mfa/verify`、`/api/v1/auth/mfa/recovery-codes` 和 `/api/v1/auth/mfa/factors/{factorId}` 支持当前账号 TOTP MFA 因子列表、创建、验证启用、恢复码元数据列表/轮换和撤销；secret 加密落库，只在 setup 响应中显示一次；恢复码明文只在 verify/rotate 响应中显示一次，数据库只保存 hash 与 prefix。
- 已启用 MFA 的账号登录时必须提交 `mfa_code` 或 `mfaCode`，可使用认证器 TOTP 或未消费的一次性恢复码；缺失时返回 `MFA_REQUIRED`，恢复码消费后不可复用，撤销 MFA 会吊销剩余 active 恢复码。
- `/api/v1/orgs/{orgId}/api-tokens` 支持 API Token 创建、列表和撤销。明文 token 只在创建响应出现一次，数据库只保存 hash 与 prefix；Bearer API Token 只能访问租户/产品权限，不能继承平台权限。
- `/api/v1/orgs/{orgId}/users/invitations` 与 `/api/v1/orgs/{orgId}/invitations` 支持邀请创建、列表和撤销；`/api/v1/auth/invitations/accept` 使用 JSON body 中的邀请 token 创建用户、组织成员关系和 Cookie 会话；响应不暴露邀请 raw token。
- `/api/v1/auth/password/forgot`、`/api/v1/auth/password/reset`、`/api/v1/auth/email-verifications` 和 `/api/v1/auth/email-verifications/confirm` 支持重置/验证 pending 数据基础闭环；重置和邮箱验证 token 均通过 JSON body 提交，不进入 URL；未知邮箱不制造 pending 数据或通知 outbox。
- `iam_notification_outbox` 保存邮件模板、收件人、pending 关联和安全 payload；一次性通知令牌只以 AES-GCM 密文写入 `iam_notification_delivery_secrets`。`notification-drain` 会领取到期 outbox，通过注入式 driver 投递并记录结果；当前支持本地 `file`、安全元数据 `log`、真实 SMTP driver 和本地加密 `queue` driver。queue driver 会写入 JSON envelope，令牌以 `notification.queue.secret_key` 重新加密为 `secret_ciphertext`，不写 raw token；临时失败会按 `notification.retry_backoff_seconds` 重新排队，达到 `notification.max_attempts` 后才标记最终失败并清空密文。`notification-dead-letters` 会输出最终失败通知的脱敏 dead-letter 报表，只包含模板、关联对象、收件人提示、失败原因、尝试次数和 secret 清理状态，不返回 payload、raw token 或密文；`notification-requeue-failed` 只会把仍有 pending 投递密文的失败通知恢复为 `pending`，对已清空或缺失 secret 的记录只输出跳过原因，不重建一次性令牌；托管外部 MQ 和告警策略仍需后续接入。
- `/api/v1/system/apis` 返回由 route registry 同步的 API 目录，要求 `permission:read` 平台权限。
- `/api/v1/system/menus` 返回当前已实现后端能力对应的系统菜单，要求 `menu:read` 平台权限并按调用者权限过滤。
- `/api/v1/system/configs`、`/api/v1/system/dictionaries`、`/api/v1/system/parameters` 提供系统配置、字典、参数基础 CRUD，分别要求 `config:*`、`dictionary:*`、`parameter:*` 平台权限。
- System 配置和参数拒绝 `secret`、`token`、`password`、`private`、`credential` 等敏感 key；此类材料必须走 secrets/env。
- `/api/v1/system/version-packages` 提供版本包 manifest 元数据登记、列表和软删除；`/api/v1/system/version-packages/{id}/publish`、`/api/v1/system/version-packages/{id}/rollback` 与 `/api/v1/system/version-packages/releases` 提供单 active 版本生命周期切换和发布事件审计，要求 `version_package:*` 平台权限。当前只更新平台版本状态并记录事件，不执行外部部署、机器回滚或制品分发。
- `/api/v1/system/media-assets` 提供媒体资产元数据登记、列表和软删除；`/api/v1/system/media-assets/upload` 通过注入式 storage 接口提供本地文件或 S3 兼容对象存储上传，要求 `media:*` 平台权限。`/api/v1/system/storage-objects` 提供当前 storage driver 下的对象浏览和对象删除控制面，要求 `storage_object:*` 平台权限；当前不管理 bucket 生命周期或对象存储权限策略。
- `/api/v1/system/traffic-probes/*` 提供流量探针目标登记、手动执行真实 HTTP 探测、结果查询和异常告警查询/确认/恢复，要求 `traffic_probe:*` 平台权限；`scheduler-run-once` 和可选后台 scheduler 会复用同一真实采集器定期写入结果，并按 `audit.operation_record_retention_days` 清理过期操作记录；非 `healthy` 结果会派生持久化告警，后续同目标 `healthy` 结果会自动恢复未关闭告警。`/api/v1/system/traffic-probes/events` 提供 `text/event-stream` SSE 告警快照事件流，复用同一告警查询和 `traffic_probe:read` 权限，并通过 `scheduler.event_stream_heartbeat_seconds` 输出浏览器重连提示。
- `/api/v1/system/operation-records` 返回后端中间件写入的 API 操作记录，要求 `operation_record:read` 平台权限；支持按 `method`、route 模板 `path`、`status`、`actor_user_id`、`created_from`、`created_to`、`limit` 和 `offset` 过滤与分页，时间范围使用 RFC3339，`path` 过滤不接受 query/fragment；`/api/v1/system/operation-records/summary` 复用同一过滤条件输出真实后端聚合的总量、状态段、method 分布和 top path，`operation-records-summary` CLI 使用同一 service/repository 用例；`/api/v1/system/operation-records/export.csv` 复用同一过滤条件和权限导出 UTF-8 CSV；`POST /api/v1/system/operation-records/prune` 按 `audit.operation_record_retention_days` 与 `audit.operation_record_prune_batch_size` 清理过期记录，要求 `operation_record:write` 平台权限；`/openapi.yaml` 不进入操作记录。
- `/api/v1/system/server-status` 返回当前进程真实运行时状态和 `sysinfo` 采集的全局 CPU、进程 CPU、内存、进程内存、交换空间、磁盘容量/使用量、磁盘数量、网络接口数量、网络累计接收/发送字节、系统 uptime/boot time 与 1/5/15 分钟 load average，要求 `server:read` 平台权限；不会 mock 尚未接入采集器的主机指标。
- `/api/v1/system/metrics/prometheus` 以 `text/plain; version=0.0.4` 导出同一组真实采集字段的 Prometheus 文本指标；平台会话按 `server:read` 平台权限授权，外部监控可使用 `observability.prometheus_scrape_token_hash` 配置的专用 Bearer scrape token 哈希授权。它不是匿名公网 `/metrics`，也不会导出 Cookie、Authorization、raw token、secret 或请求体。

## 当前 WebUI 闭环

- `web/app` 是新的 React/Vite/TypeScript 前端，不复用旧 `aoi-admin/web/app` 的运行路径。
- `webui.enabled` 与 `webui.dist_dir` 控制 Rust 静态托管；禁用时后端只提供 API 与探针。
- API 入口集中在 `src/lib/api/endpoints.ts` 和 `src/lib/api/client.ts`；请求使用 HttpOnly Cookie、`credentials: include`、`X-Locale`、product/client headers 和 401 refresh retry，不把 session token、refresh token、API Token 或 MFA secret 写入 localStorage/sessionStorage/URL/日志/测试快照。
- WebUI 会先读取 `/api/v1/system/public-settings` 并缓存运行时配置；当后端启用 CSRF 时，API client 会从配置化 cookie 读取 token，并只在非 GET 请求中发送配置化 CSRF header。
- i18n 默认 `zh-CN`，`src/i18n/locales/zh-CN.json` 与 `en.json` 通过 `npm run lint:i18n` 保持 key parity。
- 公开入口读取 Rust runtime state 和 public settings，展示平台定位、后端在线状态、初始化状态、默认语言和 CSRF 状态；Setup 页调用 Rust setup/IAM 初始化接口，并展示后端结构化配置检测结果；Login 页调用 Cookie 会话接口；Account 页在后端公开设置允许时展示自助注册，并调用邀请接受、密码重置、邮箱验证 pending 流程接口；Admin 页展示/操作当前 Rust 已有的 IAM、System、版本包、媒体上传/元数据和流量探针 API；System 页已接入配置、字典、参数创建/删除，版本包创建/发布/回滚/删除和发布事件展示，媒体上传/元数据登记/删除，对象存储对象浏览/删除，流量探针目标创建/运行/删除和探针告警查询/确认/恢复；IAM 页已接入租户用户资料/状态/角色管理、租户自定义角色、API Token 元数据、邀请、TOTP MFA 因子列表/setup/verify/revoke 和恢复码元数据/轮换；不展示未实现的平台角色编辑、跨产品授权、外部版本部署执行器或 bucket 生命周期策略。

## 架构边界

- Handler 只做 HTTP 输入输出、Cookie/Header 适配和 JSON 响应。
- Service/Usecase 承载业务规则、事务意图、安全判断和跨仓储编排。
- Repository trait 是业务层依赖边界；SQLite、PostgreSQL、MySQL、缓存、邮件、存储、指标采集都必须通过接口注入。数据库驱动由配置枚举和 `database_runtime` 能力报告控制，App 生命周期层只持有中性数据库连接句柄，禁止在连接层对未实现驱动做静默 fallback。
- 配置加载采用主 YAML、可选 secrets YAML、环境变量三层覆盖；生产环境会拒绝 dev、占位符和过短密钥，真实密钥不得写入 System 配置表或前端产物。
- `storage.driver` 支持 `local` 和 `s3`；S3 兼容实现作为媒体上传和对象浏览/删除后端注入，不管理 bucket 生命周期或权限策略。生产环境禁止 S3 明文 http，并要求对象存储 secret 通过 secrets/env 覆盖为强随机值。
- Notification outbox worker 通过 `NotificationRepository` 与 `NotificationSender` 注入，IAM 只负责在事务中产生 pending 数据、outbox 和加密投递 secret，不直接发送邮件；SMTP SDK 和 queue envelope 写入都只存在于基础设施层。
- Scheduler 层只编排已有 service 用例；流量探针和操作记录留存清理 scheduler 复用 `SystemService` 与注入式 repository，不绕过 System service 直接读写数据库。
- 加密、hash、token 和 TOTP 辅助位于 `crates/tools/crypto`，工具库返回 `CryptoError`；IAM/Notification service 再转换为应用错误、审计和响应语义。
- 操作记录中间件存储匹配后的 route 模板，而不是原始 URL path；查询和汇总过滤在 service 层做输入校验和规范化，CSV 导出也由 service 层复用同一查询结果完成格式化；留存清理由 `audit` 配置驱动，CLI、HTTP 管理动作和 scheduler 都复用同一 usecase；repository trait 只接收领域化过滤条件，并由静态 SQL 模板完成过滤、分页、聚合与删除；pending 通知 token 不应进入 URL，也不会写入 `system_operation_records`。
- Route registry 是当前 API 的单一事实来源；真实路由注册、权限元数据、API catalog、IAM permission 同步和 OpenAPI 都从它派生。HTTP smoke 测试会把启动后数据库中的 `system_apis`、`iam_permissions` 与 route registry 集合逐项比对，防止手写漂移。
- `docs/api/openapi.yaml` 是由 `cargo run -p app -- openapi --config configs/console.example.yaml --output docs/api/openapi.yaml` 生成的契约快照，不能手写维护；route registry 单元测试会比对快照与当前生成结果。
- 不引入插件系统、插件市场、远程插件协议、旧 Go runtime 路径或旧项目身份。

## 验证

```powershell
cargo fmt --all --check
cargo check --workspace
cargo clippy --workspace --all-targets -- -D warnings
cargo test --workspace
cargo build --workspace
cargo run -p app -- routes --config configs/console.example.yaml
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
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-runtime-smoke.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-deploy-preflight.ps1 -ApplyMigrations
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/security-sensitive-scan.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/runtime-config-boundary-scan.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/webui-capability-boundary-scan.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/governance-scan.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/goal-deliverable-audit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/goal-completion-audit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/aoi-admin-source-audit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/stage-acceptance-report-audit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/target-acceptance-report-validator-smoke.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-schema-dialect-scan.ps1
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

`scripts/architecture-boundary-scan.ps1` 会同时做负向依赖扫描和正向注入扫描：handler 不能依赖 infrastructure/repository/SQL，service 不能依赖具体数据库 runtime，repository trait 不能依赖 HTTP transport，全局 `types` 不能承载业务/DTO；同时固定检查 `IamService`、`SetupService`、`NotificationService`、`SystemService` 只接收 `Arc<dyn ...>` trait 对象，SQLite/PostgreSQL/MySQL repository、notification sender、media storage、metrics collector 和 traffic probe runner 都在 infrastructure 边界实现对应接口。

`scripts/database-runtime-smoke.ps1` 会在 `target/database-runtime-smoke` 下使用临时 SQLite，验证 `database-plan`、`database-ping`、`database-insert-id-probe`、`database-migrate`、`database-setup-repository-probe`、`database-iam-repository-probe`、`database-notification-repository-probe`、`database-system-repository-probe`、`database-migration-history`、`database-schema-check`、`database-preflight` 和第二次迁移的 skipped 报告；SystemRepository probe 会同时证明操作记录汇总 SQL 可用；它不会修改默认 `data/console.sqlite`。

`scripts/runtime-config-boundary-scan.ps1` 会验证前端运行时产品码、client type、产品/header/CSRF 名称均来自 `/api/v1/system/public-settings`，并确认 WebUI 页面不散落 `/api/v1` 路径字符串。

`scripts/webui-capability-boundary-scan.ps1` 会验证 System WebUI 的服务器状态、操作记录汇总、版本包、媒体、对象存储对象和流量探针调用都能在 Rust route registry 找到对应路由，并扫描前端运行源码，防止旧插件入口、外部版本部署执行器、bucket 生命周期/权限策略控制面或 mock/fake 生产能力回流。

`scripts/goal-deliverable-audit.ps1` 会把 `/goal` 里的交付物清单转成静态审计：根 Cargo workspace、分层目录、配置示例、README/docs/AGENTS、项目 skills、迁移矩阵四类状态、OpenAPI/route registry、WebUI i18n、SQLite/PostgreSQL/MySQL 迁移和“Go 代码只能留在 `aoi-admin/` 参考目录”都必须存在，并再次拒绝新迁移中的插件 schema。

`scripts/goal-completion-audit.ps1` 面向最终完成判定：默认输出当前 `ready` 状态和阻断项 JSON，便于阶段推进时查看缺口；准备声明 `/goal` 完成前必须传入真实目标环境 JSON 并加 `-RequireReady`，例如 `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/goal-completion-audit.ps1 -TargetAcceptanceReport "target/target-environment-acceptance/<report>.json" -RequireReady`。它会复用交付物审计、阶段报告审计、目标报告校验器 smoke，并要求目标验收清单不再有 `待填写`、阶段报告不再声明未完成、最终目标报告通过非本地 HTTPS/full/passed 校验。本地 `-AllowLocalHttp` smoke 报告不能让该审计进入 ready 状态。

`scripts/aoi-admin-source-audit.ps1` 会交叉检查 `aoi-admin/` 参考目录中的旧代码、文档、路由契约、迁移、配置、i18n、React、旧 `AGENTS.md`、旧 `.agents/skills`、旧 `docs/ai`、旧插件代码/配置/文档/示例都已在来源索引和迁移矩阵中覆盖，防止后续把“先审计旧项目”的前置要求退化为人工记忆。

`scripts/stage-acceptance-report-audit.ps1` 会检查阶段验收报告仍包含“已完成内容、进度百分比、剩余任务、删除的旧设计、验证结果、残留风险和下一步”，验证命令列表包含 Rust/WebUI/证据链门禁，并在目标环境验收模板仍有待填写项时拒绝 100% 进度声明。

`scripts/target-acceptance-report-validator-smoke.ps1` 会合成目标验收 JSON 正反样例，验证 `validate-target-acceptance-report.ps1` 能接受非本地 HTTPS full/passed 报告和本地 smoke 报告，并拒绝本地 HTTP final、partial、failed、缺 CSRF、缺匿名指标保护证据或缺授权 Prometheus scrape 证据的报告。

`scripts/database-external-smoke.ps1` 面向真实 PostgreSQL/MySQL 服务，验证对应 driver 的 `database-plan`、`database-ping`、`database-insert-id-probe`、`database-migrate`、`database-setup-repository-probe`、`database-iam-repository-probe`、`database-notification-repository-probe`、`database-system-repository-probe`、`database-migration-history`、`database-schema-check`、`database-preflight` 和二次迁移 skipped 报告，并比对迁移计划 SHA-256 与 `schema_migrations` 记录；MySQL 会额外确认插入后 ID 通过同一连接内 `select last_insert_id()` 获取，SetupRepository probe 会写入一次性 smoke run/log 证明外部 Setup trait 读写路径可用，IamRepository probe 会写入一次性管理员、权限、会话、API Token、pending 通知、邀请、密码重置、邮箱验证、MFA 和审计记录验证外部 IamRepository trait 读写路径，NotificationRepository probe 会写入一次性 outbox 记录并验证 claim/deliver/retry/fail、dead-letter 查询、投递 secret 清理、已清除 secret 的 requeue 拒绝和仍有 pending secret 的安全重排队，SystemRepository probe 会写入一次性 catalog/menu/config/dictionary/parameter/operation/version/media/traffic probe 记录并校验操作记录汇总报表，验证 System trait 读写路径。GitHub Actions 的 `external-database-smoke` job 会用服务容器运行 PostgreSQL/MySQL runtime smoke，并继续执行 `database-deploy-preflight.ps1 -ApplyMigrations` 验证部署门禁；外部库的 `database-preflight` 必须保持 `schema_ready=true`、`repository_ready=true` 且 `serve_ready=true`。目标环境验收记录模板见 [目标环境验收清单](docs/deployment/target-environment-acceptance.md)。

`scripts/database-deploy-preflight.ps1` 是部署前数据库门禁，会按目标配置执行 plan、ping、insert-id-probe、可选显式迁移、migration-history、schema-check 和 preflight。SQLite、PostgreSQL 和 MySQL 都必须得到 `serve_ready=true`。

`scripts/target-environment-acceptance.ps1` 面向真实目标环境，把 PostgreSQL/MySQL 的 external smoke、部署前数据库门禁、目标入口 HTTPS 策略、HTTP 探针、Cookie/CSRF 生产策略、受保护指标策略、WebUI fallback 和 `/api/*` 不被 SPA fallback 吞掉的检查串成一份 JSON 验收报告。示例：`powershell -NoProfile -ExecutionPolicy Bypass -File scripts/target-environment-acceptance.ps1 -Driver postgres -Url "<postgres-url>" -BaseUrl "https://console.example.com" -ApplyMigrations`。默认 full scope 必须同时包含数据库和 HTTP/WebUI 探针，缺 `-BaseUrl` 或跳过任一侧都会失败；非本地入口必须使用 `https://`，本地 loopback 的 `http://` 只用于 smoke；只有显式传入 `-AllowPartial` 时才允许生成 `result=partial` 的诊断报告，不能作为最终发布证据。HTTP 验收会要求 public settings 暴露 `csrf_enabled=true`、下发非 HttpOnly 且 Secure 的 CSRF cookie，并证明缺 CSRF 的非 GET `/api/*` 请求会被 403 拒绝；`metrics-policy` 会先以匿名请求探测 `/api/v1/system/metrics/prometheus`，要求返回 401/403 且不泄露 Prometheus 指标正文，再读取 `CONSOLE_ACCEPTANCE_METRICS_SCRAPE_TOKEN` 发起专用 Bearer scrape 请求，要求返回 200、包含真实指标正文且不泄露 raw token、API token、session token 或 secret。报告默认写入 `target/target-environment-acceptance/`，数据库 URL 和 scrape token 会被脱敏为占位符。最终发布前还必须运行 `scripts/validate-target-acceptance-report.ps1 -ReportPath "<report-json>"` 校验归档报告，确保 `scope=full`、`result=passed`、非本地 HTTPS、数据库 preflight、HTTP/WebUI 探针、`entrypoint-security`、`csrf-policy` 和 `metrics-policy` 都是通过状态；`-AllowLocalHttp` 只允许校验本地 smoke 报告结构，不能用于发布证据。

`scripts/repository-dialect-audit.ps1` 会扫描当前 repository 实现里的 SQLite 方言隔离点，例如 `Pool<Sqlite>`、`SqliteConnection`、`SqlDialect::Sqlite` 常量和生产 SQL literal 中的 SQLite `?` bind，并确认 PostgreSQL/MySQL 的 `DatabaseRuntimeSupport` 仍报告 `ready`。当前审计基线为 `Pool<Sqlite>=2`、`SqliteConnection=5`、`SqlDialect::Sqlite` 常量=1、生产 SQL literal 中 SQLite bind placeholder `?=0`、`insert_returning_id=0`、`sqlite_row_type=0`、`last_insert_rowid=0`、`scattered_setup_state_sql=0`、`scattered_setup_run_sql=0`、`scattered_session_sql=0`、`scattered_api_token_read_sql=0`、`scattered_system_settings_sql=0`、`scattered_system_registry_sql=0`、`scattered_traffic_probe_state_sql=0`、`scattered_media_version_read_delete_sql=0`、`scattered_version_package_read_update_sql=0`、`scattered_system_insert_sql=0`、`scattered_identity_insert_sql=0`、`scattered_identity_count_sql=0`、`scattered_membership_insert_sql=0`、`scattered_audit_insert_sql=0`、`scattered_org_user_management_sql=0`、`scattered_iam_permission_list_sql=0`、`scattered_role_permission_state_sql=0`、`scattered_iam_role_state_sql=0`、`scattered_iam_pending_state_sql=0`、`scattered_user_secret_state_sql=0`、`scattered_iam_workflow_insert_sql=0`、`scattered_notification_insert_sql=0`、`scattered_notification_state_sql=0`、`scattered_mfa_factor_state_sql=0`、`scattered_mfa_recovery_state_sql=0`、`scattered_version_release_event_insert_sql=0`。

`DatabaseDriver::runtime_support()` 与该审计基线保持同一事实来源；`database-preflight`、`database-schema-check`、`database-ping`、`database-insert-id-probe`、`database-setup-repository-probe`、`database-iam-repository-probe`、`database-notification-repository-probe`、`database-system-repository-probe` 和 `database-migrate` 会输出或验证当前 driver 的方言边界，逐项说明 Setup/IAM/Notification/System trait 是否已接入当前 driver。PostgreSQL/MySQL 的 SetupRepository、IamRepository、NotificationRepository 与 SystemRepository 已接入外部 repository set；MySQL 业务写路径按 `InsertIdStrategy::DialectSpecificPostInsertRead` 与 `InsertIdRead::PostInsertQuery("select last_insert_id()")` 处理插入后 ID 获取，不能依赖 `returning id` 语义。

SQLite `on conflict(...)` upsert、`limit ?` 查询、Setup 完成状态/run/log SQL、会话 SQL、API Token 读取/列表/撤销 SQL、System 配置/字典/参数读删 SQL、API catalog/menu 读取、操作记录查询/写入/汇总/留存删除 SQL、流量探针目标读删/status 更新、告警状态更新、媒体库列表/软删除、版本包列表/按 ID 查询/active 查询、版本包退役/激活更新、版本包/媒体库/流量探针目标/结果/告警插入、IAM 组织/用户创建、成员关系写入、审计日志写入、组织用户管理、IAM 权限列表、角色权限关系、租户角色读取/更新/删除、IAM pending 状态读写、用户密码/邮箱验证状态更新、IAM 角色/API Token/邀请/密码重置/邮箱验证/MFA factor 插入、通知 outbox/投递 secret 插入与状态读写、MFA 恢复码插入、MFA factor 状态读写、MFA 恢复码状态读写、版本发布事件插入、版本发布事件列表/版本包软删除和删除前状态查询 SQL 已集中到 `sql_templates.rs`。仓储源码不再散落 `returning id`，但 SQLite/PostgreSQL 模板仍使用显式 `returning id`，MySQL 模板不假装支持该语义，外部 repository 接入时必须按 `InsertIdRead::PostInsertQuery("select last_insert_id()")` 在同一连接/事务内读取生成 ID；MySQL 操作记录留存删除通过 derived table 避免目标表子查询限制；行解码已收敛到 infrastructure-local `FromRow` DTO，不再依赖 `SqliteRow` 或 `last_insert_rowid()`；它不是迁移替代品，而是防止在方言阻断点未消除时误宣称外部数据库可运行。

`scripts/deployment-smoke.ps1` 会构建 WebUI 和后端，在 `target/deployment-smoke` 下使用临时 SQLite、通知和媒体目录先执行 `database-deploy-preflight -ApplyMigrations`，再启动真实服务，并验证 `/health`、`/ready`、`/openapi.yaml`、Setup API、public settings、匿名 Prometheus 拒绝、专用 scrape token 授权与 WebUI fallback；随后它会调用 `target-environment-acceptance.ps1` 生成 `target/deployment-smoke/target-acceptance-local.json`，并用 `validate-target-acceptance-report.ps1 -AllowLocalHttp` 校验本地报告结构。该本地 JSON 只证明脚本链路和 loopback HTTP smoke，不是最终发布证据；结束后自动停止服务。

前端变更额外运行：

```powershell
npm --prefix web/app run typecheck
npm --prefix web/app run lint:i18n
npm --prefix web/app run build
npm --prefix web/app run test:e2e
```
