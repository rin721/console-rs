# Aoi[葵] 架构说明

## 分层

Aoi[葵] 是产品叙事代号；工程内部使用中性 crate 和模块命名。当前采用模块化单体：

- `crates/core/app`：应用生命周期、装配、启动/关闭、服务组合和运行状态。
- `crates/core/app/src/error.rs`：`AppResult`、`AppError` 等 app 生命周期错误契约；错误枚举本身保持中性，不把 `sqlx::Error` 或 `std::io::Error` 作为全局类型暴露。
- `crates/core/types`：未来生命周期共享类型的占位 crate；当前不导出业务模型、HTTP DTO、数据库行结构或前端展示类型。
- `crates/core/config`：主 YAML、可选 secrets YAML、环境变量映射、类型化 driver 枚举、数据库运行时能力报告和生产安全校验。
- `crates/tools/crypto`：密码 hash、secret 加密、token 生成和 TOTP 计算；只返回 `CryptoError`，由应用服务转换为 `AppError`。
- `handler/http`：HTTP 输入输出、Header/Cookie 适配、JSON 序列化和最外层错误响应映射。
- `service`：Setup、IAM、System 用例编排，承载事务意图和安全判断。
- `domain`：面向 API 与业务语义的模型。
- `repository`：service 依赖的接口边界。
- `infrastructure`：SQLite、PostgreSQL、MySQL、迁移、通知、存储和指标等具体实现；数据库启动入口返回中性 `DatabaseConnection` 和 `RepositorySet` trait 集合，外部 driver 已接入对应 pool、显式迁移执行器和 Setup/IAM/Notification/System repository set，不允许静默回退到 SQLite。
- `scheduler`：后台调度编排，只调用 service 用例，不直接访问数据库或外部 SDK。
- `transport/http/route_registry.rs`：路由、权限、API catalog、OpenAPI 的事实来源。
- `web/app`：新的 React/Vite WebUI 起点，只消费 Rust route registry 已暴露的 API，不复用旧 Go 前端运行路径。

错误只向上转换：tools 返回工具错误，domain 返回业务语义错误，repository/infrastructure 返回各自边界错误，service/usecase 汇总为 app 层错误，handler 再映射为 HTTP 状态和响应体。基础设施错误在 app 生命周期层转换为中性 `Infrastructure(String)`/`Storage(String)`，避免全局类型层依赖具体数据库或文件系统错误。禁止底层直接构造 HTTP 响应，也禁止用全局 `types` 模块收纳业务 DTO。

## 当前闭环

- `/health`、`/ready` 提供存活和就绪探针；`/ready` 通过基础设施层数据库连接句柄执行 ping，不在 handler 中硬编码具体 SQL driver。
- `/openapi.yaml` 从 route registry 生成；仓库内 `docs/api/openapi.yaml` 由 `cargo run -p app -- openapi --config configs/console.example.yaml --output docs/api/openapi.yaml` 生成，不能手写维护。
- `/api/v1/setup/*` 提供初始化状态、schema、结构化配置检测、运行列表、步骤日志和完成状态。配置检测在 Setup service 中读取当前 Settings，报告数据库、迁移、secrets、Cookie/CSRF、通知、存储和 WebUI 托管状态；响应只包含状态和说明，不包含密钥、SMTP 密码或 token 明文。运行列表按 `setup_runs.updated_at` 返回最近记录；完成初始化同样在 service 层校验配置检测无 error 且首个管理员已存在，handler 只负责 HTTP 输入输出；携带 `run_id` 时，repository 在同一事务内更新 `setup_runs.status`、追加 `complete` 日志并写入全局完成状态。
- `/api/v1/auth/setup/initial-admin` 创建首个管理员，使用事务写入组织、用户、角色、成员和审计日志。
- `/api/v1/auth/login`、`/api/v1/auth/refresh` 与 `/api/v1/me/session` 建立、轮换并读取 HttpOnly Cookie 会话，权限从 `iam_memberships -> iam_roles -> iam_role_permissions -> iam_permissions` 解析。
- `iam_sessions.session_token_hash` 与 `iam_sessions.refresh_token_hash` 只保存哈希值；refresh 使用旧 refresh hash 查询后，以 `id + old refresh hash` 原子更新新 session/refresh hash，防止旧 refresh 被重复使用。
- CSRF 保护由 `transport/http/csrf.rs` 在 HTTP 中间件层统一处理，配置来自 `auth.csrf`。安全方法和非 `/api/*` 路径豁免；启用后，非安全 `/api/*` 请求必须同时携带配置化 CSRF cookie 与 header。`/api/v1/system/public-settings` 负责下发新的 CSRF cookie；该 cookie 故意不加 `HttpOnly`，只用于双提交校验，不承载身份或服务端会话。
- `/api/v1/iam/orgs` 与 `/api/v1/iam/permissions` 是平台级只读管理面，分别要求 `org:read` 与 `permission:read`；`/api/v1/iam/orgs/{orgId}/users` 是租户级用户只读管理面，要求当前会话属于该组织并具备 `user:read`；`PUT /api/v1/iam/orgs/{orgId}/users/{userId}` 要求 `user:write`，只能更新当前租户用户的显示名、`active`/`disabled` 状态和租户角色集合，并保护组织最后一个 active owner。
- `/api/v1/iam/orgs/{orgId}/roles` 支持租户角色读取和创建，`/api/v1/iam/orgs/{orgId}/roles/{roleId}` 支持租户角色更新和删除，分别由 `role:read` 与 `role:write` 保护。自定义租户角色只能绑定当前产品的 `tenant`/`product` 权限，不能绑定平台级权限；内置角色不可修改或删除，仍被成员或 pending 邀请引用的角色不可删除。
- IAM API Token 只在创建响应返回一次明文，持久化只保存 hash、prefix、状态和过期时间；Bearer Token 只解析租户/产品权限，不继承平台权限。
- `/api/v1/auth/register` 由 `auth.self_signup_enabled` 显式控制，默认关闭。启用后，注册路径在同一事务内创建 pending 用户、tenant 组织、owner 成员关系、邮箱验证 pending 数据、通知 outbox 和加密投递 secret；邮箱确认前用户保持 `pending_verification`，不能登录。
- TOTP MFA 通过 `/api/v1/auth/mfa/factors` 返回当前账号因子元数据，通过 `/api/v1/auth/mfa/setup` 生成 pending 因子，setup 响应一次性返回 secret 和 otpauth URL；`/api/v1/auth/mfa/verify` 验证当前验证码后激活因子，并一次性返回新恢复码明文；`GET/POST /api/v1/auth/mfa/recovery-codes` 只返回恢复码元数据或轮换后的新明文，数据库只保存 hash 与 prefix；`DELETE /api/v1/auth/mfa/factors/{factorId}` 只允许撤销当前账号自己的因子，并在同一事务内吊销 active 恢复码。
- 登录会在密码正确后检查已启用的 TOTP 因子；缺少验证码返回 `MFA_REQUIRED`，验证码错误返回 `UNAUTHORIZED`；未消费恢复码可作为一次性 MFA 凭证，消费后不可复用。
- 自助注册、邀请、密码重置、邮箱验证建立 pending 数据并在同一事务中写入 `iam_notification_outbox` 和 `iam_notification_delivery_secrets`；outbox 只保存模板、收件人、关联 pending id 和安全 payload，不保存 raw token。一次性令牌只以 AES-GCM 密文保存在投递 secret vault 中，`notification-drain` 命令会领取到期 outbox，通过 `NotificationSender` driver 处理并写入 delivered/retry/final failed 状态；临时失败只增加 `attempt_count` 并重排 `available_at`，达到 `notification.max_attempts` 后才标记最终 failed 并清空密文。当前默认 file driver 用于本地开发投递，`smtp` driver 通过 `lettre` 在基础设施层投递真实邮件。接受邀请、重置密码和确认邮箱验证时，客户端必须通过 JSON body 提交 token，service 用 token hash 查询 pending 数据，并在事务中更新用户、成员关系或 consumed 状态。
- `/api/v1/system/apis` 返回由 route registry 同步的 API 目录，并要求 `permission:read` 平台权限。
- `/api/v1/system/menus` 返回当前已实现 API 对应的菜单入口，并按调用者权限过滤。
- `/api/v1/system/configs`、`/api/v1/system/dictionaries`、`/api/v1/system/parameters` 提供系统配置、字典、参数基础 CRUD，权限来自 route registry。
- `/api/v1/system/version-packages` 管理版本包 manifest 元数据；`publish`、`rollback` 和 `releases` 端点只负责单 active 版本状态切换与事件审计，不执行外部部署、机器回滚或制品分发。
- `/api/v1/system/media-assets` 管理媒体资产元数据；`/api/v1/system/media-assets/upload` 通过 `MediaStorage` 接口写入配置化 `local` 或 S3 兼容存储。`/api/v1/system/storage-objects` 通过同一注入式 storage 接口列出和删除当前 driver 下的真实对象；S3 场景会限制在配置前缀内，不返回访问密钥。bucket 生命周期和对象存储权限策略仍属于部署/云控制面，当前不由本服务管理。
- `/api/v1/system/traffic-probes/*` 管理流量探针目标、手动执行一次真实 HTTP 探测并保存结果；`scheduler-run-once` 和可选后台 scheduler 会按配置复用同一 `TrafficProbeRunner` 周期采集并保存结果。非 `healthy` 结果会派生 `system_traffic_probe_alerts` 持久化告警，可通过 HTTP API 查询、确认和恢复；后续同目标 `healthy` 结果会自动把未关闭告警标记为 `resolved`。`/api/v1/system/traffic-probes/events` 是 `text/event-stream` SSE 事件流，复用告警查询、`traffic_probe:read` 权限和配置化 `scheduler.event_stream_heartbeat_seconds`，首个事件立即返回真实告警快照并携带浏览器 `retry` 重连提示。
- `/api/v1/system/operation-records` 返回 HTTP 中间件写入的 API 操作记录；`/api/v1/system/operation-records/summary` 只基于已落库的 route 模板、method、status、actor 和时间做聚合，不反查请求正文；`/openapi.yaml` 不进入记录。
- `/api/v1/system/server-status` 返回进程号、启动时间、运行时长、操作系统、架构、并行度、产品版本、数据库驱动，以及 `sysinfo` 真实采集的全局 CPU、进程 CPU、内存、进程内存、交换空间、磁盘容量/使用量、磁盘数量、网络接口数量、网络累计接收/发送字节、系统 uptime/boot time 与 1/5/15 分钟 load average；`/api/v1/system/metrics/prometheus` 复用同一真实采集结果导出 Prometheus text exposition format。平台会话由 `server:read` 平台权限保护，外部监控可走 `observability.prometheus_scrape_token_hash` 配置的专用 Bearer scrape token 哈希；租户 API Token 仍保持租户 scope，不能借此读取平台级指标。尚未接入采集器的主机指标不返回。

## WebUI 边界

- 新 WebUI 首屏是公开平台入口和 Setup/Login/Account/Admin 可操作界面，不是营销页。公开入口只读取 Rust health/ready/public-settings/setup 状态并展示平台定位、后端在线状态、初始化状态、默认语言和 CSRF 状态；Setup 使用 Rust setup/IAM 初始化接口并展示后端结构化配置检测结果，Login 使用 HttpOnly Cookie 会话，Account 页按 public settings 决定是否展示自助注册，并使用 Rust 邀请接受、密码重置和邮箱验证 pending 流程接口，Admin 只展示当前 Rust 后端已经实现的 IAM、System、版本包、媒体上传/元数据、对象存储对象和流量探针能力；System 页可通过 Rust 接口创建/删除配置、字典、参数、版本包、媒体资产和流量探针目标，可读取真实后端操作记录审计汇总，可发布/回滚版本包状态并查看发布事件，浏览/删除 storage 对象，手动执行真实流量探针，并处理后端持久化探针告警；IAM 页可通过 Rust 接口管理租户用户资料/状态/角色、租户自定义角色、API Token 元数据、邀请、TOTP MFA 因子列表/setup/verify/revoke 和恢复码元数据/轮换；不展示未实现的平台角色编辑、跨产品授权、外部版本部署执行器或 bucket 生命周期策略。
- Rust 服务通过 `webui.enabled` 与 `webui.dist_dir` 托管构建后的 WebUI；前端 history 路由走 SPA fallback，`/api/*`、`/health`、`/ready` 和 `/openapi.yaml` 保持后端语义，避免静态托管掩盖契约漂移。
- API endpoint 常量和请求行为集中在 `web/app/src/lib/api`；页面组件不得散落 `/api/v1` 字符串，不得把 token、MFA secret、pending 通知令牌或私密 payload 写入 URL、localStorage、sessionStorage、日志、截图或测试快照。Account 页会清理历史 URL 中的查询参数，但不会从查询参数读取邀请、密码重置或邮箱验证 token。
- 前端启动时读取 public settings，并把 product/client Header、locale 与 CSRF header/cookie 名称注入 API client。CSRF token 只保留在浏览器 cookie 中，不进入 localStorage、sessionStorage、URL 或测试快照。
- i18n 默认 `zh-CN`，`zh-CN` 与 `en` 资源必须保持 key parity；可见文案进入 locale JSON。
- 当前前端不会展示插件、外部版本部署执行器、bucket 生命周期策略或 Rust 后端尚未暴露的生产能力；服务器资源指标只展示 `/api/v1/system/server-status` 已返回的真实采集字段，外部 scrape 只能使用 route registry 已暴露且受 `server:read` 或专用 scrape token 哈希保护的 `/api/v1/system/metrics/prometheus`；操作记录汇总只调用 route registry 已暴露的 `/api/v1/system/operation-records/summary`；版本包只调用 route registry 已暴露的元数据、发布/回滚状态和事件 API；媒体上传和对象浏览/删除只调用 route registry 已暴露的 storage 接口能力；探针告警页面当前只调用 route registry 中已经暴露的查询、确认和恢复 API，后续实时订阅也只能接入已经暴露的 `/api/v1/system/traffic-probes/events` SSE 事件流。

## 权限与审计

- Route registry 中带 `permission` 的契约会同步到 `iam_permissions`，同时写入 `system_apis`；HTTP smoke 测试会在真实 SQLite 启动后逐项比对 registry、API catalog 与权限表，禁止后续出现手写漂移。
- 首个管理员创建时会获得 `platform_owner` 平台角色和当前组织 `owner` 租户角色。
- Builtin 角色会在 API catalog 同步时补齐 role-permission 关系，避免新增 route 后只更新展示目录、不更新可执行权限。
- IAM 用户/组织/角色/权限 API 不返回密码哈希、会话 token、refresh token、API Token、MFA secret 或 pending 通知 token；角色只展示 permission code 列表。租户用户更新在同一事务内校验角色归属、保护最后一个 active owner 并替换该组织内 membership；角色写入路径会在修改绑定前先校验权限 scope，避免事务失败污染原有权限关系。
- 自助注册失败时，pending 用户、组织、成员关系、邮箱验证记录、outbox 和投递 secret 必须随事务一起回滚；同邮箱或同组织编码冲突不能留下污染唯一约束的半成品数据。
- 邀请接受流程会先校验绑定角色仍存在；用户创建、membership 写入和邀请 accepted 标记必须在同一事务中完成，受邀邮箱已存在时邀请保持 pending。
- API Token、refresh token、邀请、密码重置、邮箱验证、MFA setup/verify/revoke、恢复码消费/轮换等敏感 IAM 操作写入 `iam_audit_logs` 时不得包含 raw token、MFA secret、恢复码明文、重置链接或其他秘密。
- `iam_notification_outbox.payload_json` 只能承载投递元数据和 pending 关联，不得保存邀请、密码重置、邮箱验证 raw token；未知邮箱请求不创建 pending 数据，也不创建 outbox，避免污染唯一索引和泄露账号存在性。
- `iam_notification_delivery_secrets.secret_ciphertext` 是 worker 投递一次性令牌的临时密文，投递完成或达到最终失败后必须清空；可重试失败期间保留密文以供下一轮 worker 安全投递。通知 worker 日志只允许出现模板、通道、关联对象和脱敏收件人提示。SMTP 和 queue 都已通过 `NotificationSender` 接口注入，IAM service 不依赖 SMTP SDK 或队列文件实现；queue driver 会写入结构化 envelope，并用 `notification.queue.secret_key` 再次加密令牌，不能泄露 token 到日志或 URL。
- 操作记录只保存 route registry 的匹配模板，不保存原始 path；审计 summary 只聚合这些最小字段，不保存或派生请求体、Cookie、Authorization；pending 通知令牌必须通过 JSON body 提交，不能进入 URL 或 `system_operation_records`。
- `iam_mfa_factors.secret_ciphertext` 保存 AES-GCM 密文；加密密钥来自 YAML/env/secrets 分层，生产环境必须覆盖为强随机值。`iam_mfa_recovery_codes` 只保存恢复码 hash、prefix 与生命周期状态；明文只在 verify/rotate 响应中返回一次，不进入日志、URL、localStorage 或测试快照。
- `system_operation_records` 由 Axum 中间件在响应后写入，只记录 method、path、status、actor_user_id 和时间，不记录请求体、Cookie、Authorization 或 token。
- CSRF 中间件拒绝请求时只返回通用 `FORBIDDEN`，不记录或回显 cookie/header 内容；生产配置校验要求 `auth.csrf.enabled=true` 且 `auth.csrf.secure=true`，避免跨站表单在公网环境绕过 Cookie 会话保护。
- `system_configs` 和 `system_parameters` 不是 secrets 存储；包含 `secret`、`token`、`password`、`private`、`credential` 的 key 会被 service 拒绝，真实密钥继续走 YAML/env/secrets 分层。
- 配置优先级为主 YAML < secrets YAML < 环境变量。生产环境必须通过 secrets 文件或环境变量覆盖会话、MFA 与通知投递密钥；`dev-`、`${...}`、`change-me`、`replace-with-*`、`example` 和过短密钥都会被拒绝。
- `system_media_assets.storage_key` 只保存存储引用，不保存文件内容；service 会拒绝路径穿越、绝对路径和带敏感字样的 key。上传接口生成 `local/<uuid>.<ext>` 或 `s3/<prefix>/<uuid>.<ext>` 形式的 storage key，不复用客户端文件路径。S3 endpoint 不允许携带用户名密码，生产环境禁止明文 http，访问密钥只走配置/secrets/env 注入。
- `system_traffic_probe_targets.url` 由平台权限保护，service 会拒绝用户信息和敏感 query key；探测结果只记录状态码、耗时、最终 URL 和分类原因，不保存请求头或错误原文。
- `scheduler.enabled` 默认关闭，避免本地服务启动后主动探测外部 URL；启用后只执行已登记目标的真实 HTTP 采集，不生成 mock 指标。

## 禁迁内容

旧插件系统、插件远程协议、RPC/WS 插件传输、插件配置、插件迁移表、插件前端页面和插件文档都不进入新架构。
