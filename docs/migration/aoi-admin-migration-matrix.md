# aoi-admin 到 Aoi[葵] 迁移矩阵

本矩阵来自对 `aoi-admin/` 的代码、文档、路由契约、迁移、配置、i18n、React `web/app`、`AGENTS.md`、旧 `.agents/skills` 和 `docs/ai` 的审计。旧目录保留为参考，不作为新运行路径。

## 迁移

| 能力 | 旧证据 | 新落点 | 当前状态 |
| --- | --- | --- | --- |
| route contract 驱动 API | `internal/transport/http/contracts.go` | `crates/core/app/src/transport/http/route_registry.rs` | 已建立 registry，驱动路由、权限元数据和 OpenAPI；app 装配层把 registry 转换为领域 `ApiCatalogEntry`/`SystemMenuEntry` 后同步到仓储，避免 repository 依赖 HTTP contract；`cargo run -p app -- openapi --config configs/console.example.yaml --output docs/api/openapi.yaml` 生成契约快照；HTTP smoke 测试会比对启动后 `system_apis` 与 registry 集合 |
| 健康与就绪探针 | `/health`、`/ready` | `handler/http/probe.rs` | 已实现 |
| Setup 状态、schema、配置检测、运行列表、步骤日志、完成状态 | `internal/app/initcenter` | `service/setup.rs`、`setup_*` 表、`/api/v1/setup/config-checks`、`/api/v1/setup/runs`、`/api/v1/setup/complete` | 已实现初始化状态、schema、结构化配置检测、运行列表、步骤日志和完成状态；配置检测不返回密钥、SMTP 密码或 token 明文；完成初始化必须先通过配置检测并创建首个管理员；携带 `run_id` 时会事务更新 run 状态并追加完成日志 |
| IAM 首个管理员、登录、refresh 会话轮换、会话快照、登出 | `internal/modules/iam`、旧 refresh token 概念 | `service/iam.rs`、`iam_sessions`、`transport/http/csrf.rs` | 已实现 HttpOnly session/refresh cookie、服务端 hash 存储、refresh 防重放轮换与登出撤销；CSRF 采用配置驱动双提交 cookie/header，生产环境强制启用并要求 Secure |
| IAM 组织、用户、角色、权限管理基础 | 旧 IAM 管理页面与角色/权限表 | `iam_organizations`、`iam_users`、`iam_roles`、`iam_permissions` API | 已实现组织列表、组织用户列表、租户用户显示名/状态/角色更新、组织角色权限列表、当前产品权限列表；已开放租户自定义角色创建/更新/删除，禁止绑定平台权限并保护最后 active owner、内置角色和被引用角色 |
| IAM API Token | `20260612000400_create_iam_api_tokens.sql`、`iam.api-tokens.*` route | `iam_api_tokens`、`CreateAPITokenResult` | 已实现创建/列表/撤销和 Bearer 鉴权，明文只返回一次；`scripts/security-sensitive-scan.ps1` 固定检查 runtime、URL、本地存储、日志和文本测试产物中的 raw token 泄露出口 |
| IAM 自助注册、邀请、密码重置、邮箱验证 pending 数据 | `iam_invitations`、`iam_password_resets`、`iam_email_verifications` | `iam_*` 表、`iam_notification_outbox`、`iam_notification_delivery_secrets`、`notification-drain`、`notification-dead-letters`、`notification-requeue-failed` 与 IAM/Notification service | 已实现配置开关驱动的自助注册、邀请创建/列表/撤销/接受、密码重置、邮箱验证基础路径；自助注册会同事务创建 pending 用户、tenant 组织、owner 成员关系、邮箱验证记录、outbox 和加密投递 secret，邮箱确认前不能登录；WebUI `/account` 已接入自助注册、邀请接受、密码重置和邮箱验证操作面；pending token 只通过 JSON body 提交，不进入 URL/localStorage/sessionStorage；pending、outbox 与加密投递 secret 同事务写入，outbox 不保存 raw token；`pending_flows_roll_back_when_notification_outbox_fails` 已覆盖 outbox 中段失败时注册、邀请、密码重置、邮箱验证 pending 数据全量回滚；本地 file、SMTP 和本地加密 queue driver 都通过 `NotificationSender` 投递，临时失败按配置重试，达到最大尝试次数后最终 failed 并清空密文；`notification-dead-letters` 可输出脱敏失败通知报表，`notification-requeue-failed` 只对仍有 pending 投递密文的 failed 记录恢复调度，对已清空 secret 的最终失败记录保持跳过 |
| IAM TOTP MFA 与恢复码 | `iam.mfa.factors.list`、`iam.mfa.setup`、`iam.mfa.verify`、`iam.mfa.revoke`、`iam_mfa_factors` | Rust IAM service + AES-GCM 加密 secret + TOTP 校验 + `iam_mfa_recovery_codes` | 已实现 factor 元数据列表、setup/verify/revoke、登录 MFA_REQUIRED、恢复码一次性返回/哈希存储/消费/轮换/撤销和加密落库 |
| IAM 权限与审计 | `iam_permissions`、`iam_role_permissions`、`iam_audit_logs` | route registry 同步权限 + builtin role 绑定 + `record_audit` | 已实现基础权限执行和敏感 IAM 操作审计写入；HTTP smoke 测试会比对 `iam_permissions` 与 registry 中带 permission 的契约集合 |
| API catalog 同步 | `system_apis` 与权限同步 | `system_apis` + `SystemRepository::sync_api_catalog` | 已实现，读取 catalog 要求 `permission:read`；启动级测试验证 catalog 只来自 route registry；repository trait 只接收领域 catalog entry，不再直接引用 HTTP registry 类型 |
| System 菜单 | `system_menus`、旧管理端菜单 | `system_menus` + route registry 派生菜单 | 已实现当前 API 对应菜单，并按权限过滤 |
| System 配置、字典、参数 | `system_configs`、`system_dictionaries`、`system_parameters` | System service + repository CRUD | 已实现基础 CRUD，`secret`、`token`、`password`、`private`、`credential` 语义 key 拒绝写入 System 配置和参数表；WebUI 已接入创建和删除 |
| System 版本包 | 版本管理/包管理相关页面和表 | `system_version_packages` + `system_version_release_events` + manifest 元数据和生命周期 API | 已实现元数据登记/列表/软删除、单 active 发布/回滚状态切换和事件审计，WebUI 已接入创建、发布、回滚、删除和事件展示；不执行外部部署、机器回滚或制品分发 |
| System 媒体库与对象存储对象 | 媒体库相关页面和表 | `system_media_assets` + `MediaStorage` 接口 + `storage-objects` API | 已实现元数据登记/列表/软删除、`local`/S3 兼容上传、当前 storage driver 对象浏览和对象删除；WebUI 已接入上传、登记、删除和对象列表/删除；bucket 生命周期和权限策略控制面暂缓 |
| System 流量探针 | `system.traffic-hijack.targets.*`、`RunTrafficProbe` | `system_traffic_probe_targets/results/alerts` + `traffic-probes` API + scheduler + SSE events | 已实现目标登记、手动真实 HTTP 探测、结果查询、异常告警查询/确认/恢复、同目标健康结果自动恢复未关闭告警、`scheduler-run-once` 和可选后台周期采集；scheduler 同时复用 System service 执行操作记录留存清理；`/api/v1/system/traffic-probes/events` 已提供 `text/event-stream` 告警快照事件流，复用 `traffic_probe:read` 权限和配置化重连提示；WebUI 已接入目标创建/运行/删除、结果展示和告警确认/恢复 |
| System 操作记录 | 操作日志/记录相关模块 | `system_operation_records` + HTTP 中间件 | 已实现 method/path/status/actor/time 记录，不保存敏感请求内容；中间件存储匹配后的 route 模板，查询接口支持按 `method`、`path`、`status`、`actor_user_id`、`created_from`、`created_to`、`limit` 和 `offset` 过滤与分页，并拒绝带 query/fragment 的 path 过滤和反向时间范围；`/api/v1/system/operation-records/summary` 与 `operation-records-summary` CLI 复用同一过滤条件，输出真实后端聚合的总量、状态段、method 分布和 top path，HTTP smoke 与 WebUI e2e 覆盖总览展示；`/api/v1/system/operation-records/export.csv` 复用同一过滤条件与 `operation_record:read` 平台权限导出 UTF-8 CSV；`POST /api/v1/system/operation-records/prune`、`operation-records-prune` CLI 和 scheduler 均按 `audit.operation_record_retention_days`/`audit.operation_record_prune_batch_size` 清理过期记录，HTTP smoke 覆盖过期记录删除和当前记录保留 |
| System 服务器状态与指标导出 | 服务器状态页面 | `ServerStatus` + `SysinfoMetricsCollector` + `system.metrics.prometheus` | 已实现进程级真实状态，以及 `sysinfo` 采集的全局 CPU、进程 CPU、内存、进程内存、交换空间、磁盘容量/使用量、磁盘数量、网络接口数量、网络累计接收/发送字节、系统 uptime/boot time 和 1/5/15 分钟 load average；`/api/v1/system/metrics/prometheus` 以 Prometheus text format 导出同一组真实字段，平台会话复用 `server:read` 平台权限，外部监控可使用 `observability.prometheus_scrape_token_hash` 配置的专用 Bearer scrape token 哈希；租户 API Token 不提升为平台指标凭据；不 mock 未接入指标 |
| 默认 SQLite 迁移 | `internal/migrations` | `migrations/20260621000100_init_core.sql` | 已实现，不含插件表 |
| 数据库运行时装配边界 | 旧 Go 数据库工具与业务路径耦合 | `infrastructure::database::DatabaseConnection` + `RepositorySet` trait 集合 | 已把 app 生命周期层从具体 repository 直接依赖中拆出；`/ready` 通过数据库连接句柄 ping；`check-config` 输出 `database_runtime` 能力报告；`database-plan`、`database-ping`、`database-insert-id-probe`、`database-migrate`、`database-migration-history`、`database-schema-check` 和 `database-preflight` 覆盖 SQLite/PostgreSQL/MySQL 的迁移计划、连接、插入后 ID 策略、显式迁移、历史反查、核心表和 `serve_ready`；PostgreSQL/MySQL 已通过 `ExternalSetupRepository`、`ExternalIamRepository`、`ExternalNotificationRepository` 与 `ExternalSystemRepository` 接入 `RepositorySet`；`scripts/database-deploy-preflight.ps1` 对所有 driver 要求 `serve_ready=true`；`scripts/repository-dialect-audit.ps1` 继续扫描 SQLite-only 方言隔离点和类型化 repository 能力矩阵；部署边界记录于 `docs/deployment/database-runtime-matrix.md` |
| WebUI Public/Setup/Login/Account/Admin 起点 | 旧 `web/app` 的 public/setup/admin/i18n/API client/验证链 | 新 `web/app` React/Vite/TypeScript 应用 | 已实现公开入口、Setup 状态/schema/配置检测展示、Login、Account 自助注册/邀请/找回/验证流程、Admin 总览、IAM 组织/用户/权限视图、租户用户管理、租户角色管理、API Token 元数据、邀请、TOTP MFA 因子列表/setup/verify/revoke、恢复码元数据/轮换、System 配置/字典/参数创建删除、操作记录审计汇总、版本包创建/发布/回滚/删除和事件展示、媒体上传/登记/删除、对象存储对象浏览/删除、流量探针目标创建/运行/删除和 Rust 静态托管；只调用 Rust route registry 已暴露 API |

## 重写

| 能力 | 重写原因 | 新方向 |
| --- | --- | --- |
| 配置系统 | 旧配置包含 Go runtime、插件、RPC 和危险 dev secret | Rust `config` + 主 YAML/secrets YAML/env 三层覆盖，生产密钥、Cookie 与 CSRF 安全校验；`database.driver` 已改为类型化枚举并校验 URL scheme；配置单元测试覆盖生产样例无 secrets/模板 secrets 拒绝启动与真实外部 secrets 通过 |
| IAM scope | 旧项目近期才修正 platform/tenant 混用风险 | 新 schema 显式记录 `platform`、`tenant`、`product` |
| Token 与会话 | 旧 JWT/refresh/token 能力不应原样复制 | 使用 HttpOnly Cookie + 服务端 session/refresh hash；refresh 原子轮换，旧 token 立即失效 |
| 加密与 token 工具 | 旧项目工具、业务和基础设施边界混杂 | `crates/tools/crypto` 返回 `CryptoError`，IAM/Notification service 显式转换为应用错误 |
| OpenAPI | 旧 OpenAPI 标题和描述仍含 `go-scaffold` | 新 registry 直接生成 Aoi[葵] OpenAPI；`docs/api/openapi.yaml` 作为生成快照提交，不手写维护，并由 route registry 单元测试防漂移 |
| WebUI | 旧 React `web/app` 带有旧插件页面和旧 Go API 假设 | 新 `web/app` 仅参考信息架构、i18n parity 和验证链，不复制旧运行路径或插件页面 |
| AI skills | 旧技能是旧项目和旧前端阶段产物 | 新建 console-rs 专属 skills |

## 删除

| 旧设计 | 删除原因 |
| --- | --- |
| `internal/plugin`、`pkg/plugin`、`pkg/pluginapi` | 新架构禁止插件系统和插件市场 |
| `/api/v1/plugins`、`/plugin-api/v1/*` | 禁止迁移插件 HTTP/WS/RPC 协议 |
| `internal/migrations/*plugin*` | 新 schema 不包含插件表 |
| `docs/api/plugin-protocol/*`、`docs/modules/plugins.md` | 插件协议文档不进入新项目 |
| `configs.plugins`、`rpc` 插件相关配置 | 新项目不提供远程插件宿主 |
| `github.com/rei0721/go-scaffold` 旧模块身份 | 新项目是 `github.com/rin721/console-rs` |

## 暂缓

| 能力 | 暂缓原因 | 进入条件 |
| --- | --- | --- |
| 托管外部 MQ/告警策略 | 当前已实现本地 file/log、真实 SMTP、本地加密 queue driver、脱敏 dead-letter 报表和仅恢复 pending secret 的安全 requeue；仍未接托管 MQ/告警系统 | 接入 RabbitMQ/Kafka/SQS 等真实队列接口、外部队列 redrive 和告警策略 |
| 短信/邮件 MFA | 当前只实现 TOTP 因子与一次性恢复码 | 短信/邮件因子模型、风控策略和通知通道策略就绪 |
| 外部指标导出/深度主机指标、bucket 生命周期/权限策略控制面、外部版本部署执行器 | 基础 schema 或本地能力已预留，不能 mock；Prometheus 文本导出已接入真实 `sysinfo` CPU/内存/磁盘/load/network 字段，并支持专用 scrape token 哈希作为外部监控凭据 | 目标环境监控系统配置 raw scrape token 并通过 `metrics-policy` 授权 scrape 验收；更深采集器、bucket 生命周期/权限策略控制面、制品发布/机器回滚执行器和权限闭环就绪 |
| WebUI 后续深水区 | 当前已建立可用起点和 Rust 静态托管 | 继续补表格分页、错误恢复、细粒度表单校验和真实后端联调截图 |

